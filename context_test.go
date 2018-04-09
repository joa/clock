package clock_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/tilinna/clock"
)

var mockTime = time.Date(2018, 1, 1, 10, 0, 0, 0, time.UTC)

func TestContext(t *testing.T) {
	ctx := context.Background()

	c := clock.FromContext(ctx)
	if c != clock.Realtime() {
		t.Fatalf("want realtime clock, got %T", c)
	}

	ctx = clock.Context(ctx, clock.New(mockTime))
	m, ok := clock.FromContext(ctx).(*clock.Mock)
	if !ok {
		t.Fatalf("want *clock.Mock, got %T", m)
	}

	tm := clock.NewTimer(ctx, 5*time.Second)
	ctx1, cfn1 := clock.TimeoutContext(ctx, 10*time.Second)
	defer cfn1()
	ctx2, cfn2 := clock.DeadlineContext(ctx, mockTime.Add(15*time.Second))
	defer cfn2()
	ctx3, cfn3 := clock.TimeoutContext(ctx, 10*time.Second)
	cfn3()
	<-ctx3.Done()

	if got, want := ctx3.Err(), context.Canceled; want != got {
		t.Fatalf("want ctx3.Err(): %q, got: %q", want, got)
	}

	if d, ok := ctx2.Deadline(); !ok || !d.Equal(mockTime.Add(15*time.Second)) {
		t.Fatalf("want ctx2.Deadline(): %q, got: %q", mockTime.Add(15*time.Second), d)
	}

	var wg sync.WaitGroup
	wg.Add(3)

	var timeout time.Time
	go func() {
		timeout = <-tm.C
		wg.Done()
	}()

	go func() {
		<-ctx1.Done()
		wg.Done()
	}()

	go func() {
		<-ctx2.Done()
		wg.Done()
	}()

	m.Add(20 * time.Second) // fires all timers simultaneously
	wg.Wait()

	if !timeout.Equal(mockTime.Add(5 * time.Second)) {
		t.Fatalf("want tm timer to expire after 5 seconds, got %q", timeout)
	}
	if got, want := ctx1.Err(), context.DeadlineExceeded; want != got {
		t.Fatalf("want ctx1.Err(): %q, got: %q", want, got)
	}
	if got, want := ctx2.Err(), context.DeadlineExceeded; want != got {
		t.Fatalf("want ctx2.Err(): %q, got: %q", want, got)
	}

	<-ctx3.Done()
	if got, want := ctx3.Err(), context.Canceled; want != got {
		t.Fatalf("want ctx3.Err(): %q, got: %q", want, got)
	}

	// Test chained contexts
	dctx1, _ := ctx1.Deadline()
	ctx4, cfn4 := clock.DeadlineContext(ctx1, dctx1.Add(5*time.Second))
	defer cfn4()
	dctx4, _ := ctx4.Deadline()
	if !dctx4.Equal(dctx1) {
		t.Fatalf("want earlier deadline: %q, got: %q", dctx1, dctx4)
	}

	ctx5, cfn5 := clock.DeadlineContext(ctx1, dctx1.Add(-5*time.Second))
	defer cfn5()
	dctx5, _ := ctx5.Deadline()
	if dctx5.Equal(dctx1) {
		t.Fatalf("want earlier deadline: %q, got: %q", dctx5, dctx1)
	}
	<-ctx4.Done()
	<-ctx5.Done()
	if got, want := ctx5.Err(), context.DeadlineExceeded; want != got {
		t.Fatalf("want ctx5.Err(): %q, got: %q", want, got)
	}
}

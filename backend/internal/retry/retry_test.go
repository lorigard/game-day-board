package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDelayDoublesAndCaps(t *testing.T) {
	// Rand always returns just under 1, so each delay sits at the top of its range.
	p := Policy{BaseDelay: 100 * time.Millisecond, MaxDelay: 500 * time.Millisecond, Rand: func() float64 { return 0.999999 }}
	tests := []struct {
		retry   int
		wantMax time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{4, 500 * time.Millisecond},  // 800ms would exceed MaxDelay, so it is capped
		{80, 500 * time.Millisecond}, // a huge shift must not overflow into a negative wait
	}
	for _, tt := range tests {
		got := p.Delay(tt.retry)
		if got <= 0 || got > tt.wantMax || got < tt.wantMax*99/100 {
			t.Errorf("Delay(%d) = %v, want just under %v", tt.retry, got, tt.wantMax)
		}
	}
}

func TestDelayUsesJitter(t *testing.T) {
	p := Policy{BaseDelay: time.Second, MaxDelay: time.Minute, Rand: func() float64 { return 0.25 }}
	if got, want := p.Delay(1), 250*time.Millisecond; got != want {
		t.Errorf("Delay(1) = %v, want %v", got, want)
	}
}

func TestSleepStopsWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	err := Sleep(ctx, time.Hour)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Sleep() error = %v, want context.Canceled", err)
	}
	if time.Since(start) > time.Second {
		t.Error("Sleep() waited instead of returning at once")
	}
}

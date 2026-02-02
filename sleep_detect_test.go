package sleep_detect

import (
	"testing"
	"time"
)

func TestBobTheStrider_DetectWakeupEvents(t *testing.T) {
	mock := NewMockClock()
	period := 100 * time.Millisecond
	margin := 10 * time.Millisecond

	bob := newBobTheStriderWithClock(period, margin, mock)
	defer func() { _ = bob.Close() }()
	ch := bob.DetectWakeupEvents()

	mock.WaitForSleepers(2) // wait for Bob's 2 legs to be in sleep
	mock.Advance(time.Hour) // simulate one-hour sleep duration

	select {
	case detected := <-ch:
		if detected < 59*time.Minute+59*time.Second {
			t.Errorf("expected ~1h sleep detected, got %v", detected)
		}
	case <-time.After(time.Second):
		t.Fatal("wakeup event not received")
	}
}

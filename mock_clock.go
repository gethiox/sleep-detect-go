package sleep_detect

import (
	"sync"
	"time"
)

type mockTimer struct {
	deadline time.Time
	ch       chan time.Time
}

type MockClock struct {
	mu          sync.Mutex
	cond        *sync.Cond
	currentTime time.Time
	listeners   []chan struct{}
	timers      []*mockTimer
}

func NewMockClock() *MockClock {
	m := &MockClock{
		currentTime: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
	}
	m.cond = sync.NewCond(&m.mu)
	return m
}

func (m *MockClock) Now() time.Time                  { m.mu.Lock(); defer m.mu.Unlock(); return m.currentTime }
func (m *MockClock) Since(t time.Time) time.Duration { return m.Now().Sub(t) }
func (m *MockClock) Until(t time.Time) time.Duration { return t.Sub(m.Now()) }

func (m *MockClock) Sleep(d time.Duration) {
	if d <= 0 {
		return
	}
	ch := make(chan struct{})
	m.mu.Lock()
	m.listeners = append(m.listeners, ch)
	m.cond.Broadcast()
	m.mu.Unlock()
	<-ch
}

func (m *MockClock) After(d time.Duration) <-chan time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch := make(chan time.Time, 1)
	m.timers = append(m.timers, &mockTimer{
		deadline: m.currentTime.Add(d),
		ch:       ch,
	})
	return ch
}

// Advance moves virtual time forward, waking up sleepers and firing timers.
func (m *MockClock) Advance(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.currentTime = m.currentTime.Add(d)

	for _, ch := range m.listeners {
		close(ch)
	}
	m.listeners = nil

	var activeTimers []*mockTimer
	for _, t := range m.timers {
		if !t.deadline.After(m.currentTime) {
			select {
			case t.ch <- m.currentTime:
				break
			default:
				break
			}
		} else {
			activeTimers = append(activeTimers, t)
		}
	}
	m.timers = activeTimers
}

// WaitForSleepers blocks until 'n' goroutines are inside Clock.Sleep().
func (m *MockClock) WaitForSleepers(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for len(m.listeners) < n {
		m.cond.Wait()
	}
}

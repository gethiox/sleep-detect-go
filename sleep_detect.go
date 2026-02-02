package sleep_detect

import (
	"context"
	"sync"
	"time"
)

const numberOfBobsLegs = 2

type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	Sleep(d time.Duration)
	Until(t time.Time) time.Duration
	After(d time.Duration) <-chan time.Time
}

type stdClock struct{}

func (stdClock) Now() time.Time                         { return time.Now() }
func (stdClock) Since(t time.Time) time.Duration        { return time.Since(t) }
func (stdClock) Sleep(d time.Duration)                  { time.Sleep(d) }
func (stdClock) Until(t time.Time) time.Duration        { return time.Until(t) }
func (stdClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

type BobTheStrider struct {
	clock  Clock
	period time.Duration
	margin time.Duration

	ctx     context.Context
	cancel  context.CancelFunc
	closeWg sync.WaitGroup
}

// NewBobTheStrider returns a Bob that can detect wakeup events.
// period is the duration of every performed time.Sleep() for measurements,
// effective minimum response time is 1/2 of that duration as there are two background goroutines, in half out phase of each other.
// margin is the minimum duration needed to be additionally exceeded to evaluate a time.Sleep() measurement as device-level sleep/wake event.
func NewBobTheStrider(period, margin time.Duration) *BobTheStrider {
	return newBobTheStriderWithClock(period, margin, stdClock{})
}

func newBobTheStriderWithClock(period, margin time.Duration, clock Clock) *BobTheStrider {
	ctx, cancel := context.WithCancel(context.Background())
	return &BobTheStrider{
		period:  period,
		margin:  margin,
		clock:   clock,
		ctx:     ctx,
		cancel:  cancel,
		closeWg: sync.WaitGroup{},
	}
}
func (b *BobTheStrider) Close() error {
	b.cancel()
	b.closeWg.Wait()
	return nil
}

// DetectWakeupEvents returns a channel of device-level sleep/wake events with its duration as a value.
func (b *BobTheStrider) DetectWakeupEvents() <-chan time.Duration {
	filtered := make(chan time.Duration)
	notFiltered := make(chan time.Duration, numberOfBobsLegs) // buffered to prevent legs from slightly blocking

	b.closeWg.Add(1)
	go func() { // The Great Filter of Bob's Legs
		defer b.closeWg.Done()
		defer close(filtered)

		var timeout = b.period/numberOfBobsLegs + time.Millisecond*100

		for {
			select {
			case <-b.ctx.Done():
				return
			case v1, ok := <-notFiltered:
				if !ok {
					return
				}

				var valueToSend time.Duration

				select {
				case <-b.ctx.Done():
					return
				case v2 := <-notFiltered:
					// usual case
					valueToSend = (v1 + v2) / 2
				case <-b.clock.After(timeout):
					// rare one-leg report
					valueToSend = v1
				}

				select {
				case filtered <- valueToSend:
					break
				case <-b.ctx.Done():
					return
				}
			}
		}
	}()

	legsWg := sync.WaitGroup{}
	startBase := b.clock.Now()

	for leg := 0; leg < numberOfBobsLegs; leg++ {
		phaseShift := b.period / time.Duration(numberOfBobsLegs) * time.Duration(leg)
		initStart := startBase.Add(phaseShift)

		legsWg.Add(1)
		go func(startAt time.Time) { // Bob's Leg
			defer legsWg.Done()

			b.clock.Sleep(b.clock.Until(startAt)) // phase alignment

			// careful, expectedEnd needs to always stay in phase (b.peroid steps only)
			var expectedEnd = startAt.Add(b.period)
			for {
				b.clock.Sleep(b.clock.Until(expectedEnd))

				now := b.clock.Now()

				if now.After(expectedEnd.Add(b.margin)) {
					duration := now.Sub(expectedEnd)

					select {
					case notFiltered <- duration:
						break
					case <-b.ctx.Done():
						return
					}

					fullPeriodCycles := duration / b.period
					expectedEnd = expectedEnd.Add(b.period * (fullPeriodCycles + 1))
				} else {
					expectedEnd = expectedEnd.Add(b.period)
				}

				if b.ctx.Err() != nil {
					return
				}
			}
		}(initStart)
	}

	b.closeWg.Add(1)
	go func() { // Torso entity reporting there are no legs anymore
		defer b.closeWg.Done()

		legsWg.Wait()
		close(notFiltered)
	}()

	return filtered
}

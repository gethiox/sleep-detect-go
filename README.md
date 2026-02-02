# sleep/wakeup event detection
[![GoDoc](https://godoc.org/github.com/gethiox/sleep-detect-go?status.svg)](https://godoc.org/github.com/gethiox/sleep-detect-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/gethiox/sleep-detect-go)](https://goreportcard.com/report/github.com/gethiox/sleep-detect-go)

Revolutionary invention of phase-shifted sleep time measuring solution.

### Installation

```shell
go get -v -u github.com/gethiox/sleep-detect-go
```

### Example usage

```go
package main

import (
	"log"
	"time"

	"github.com/gethiox/sleep-detect-go"
)

func main() {
	// note: peroid and margin values might require some adjusting for heavily overloaded systems,
	//       but flawless behavior on such systems is not guaranteed.
	peroid := time.Second            // reasonable measurement peroid
	margin := time.Millisecond * 100 // reasonable margin for measurement checks 
	bob := sleep_detect.NewBobTheStrider(peroid, margin)
	defer bob.Close()

	wakeupEvents := bob.DetectWakeupEvents()

	go func() {
		for event := range wakeupEvents {
			log.Printf("wakeup event, duration: %v", event)
		}
	}()

	// do whatever, make sure `Close()` will be called, it will close `wakupEvents` channel
	time.Sleep(time.Second * 10)
}
```

### Questions nobody asked

- Q: Why Bob The Strider have two legs? One is not enough?
- A: With sleep and measure approach there is non-zero possibility where device sleep event may happen exactly when
     "detector" is not actively sleeping, it has to periodically measure time delta to detect longer than expected
     periods after all, it is that exact moment where device sleep could happen.
     This is the only reason Bob The Strider got two legs, just to cover this specific edge-case.

### Tests Needed!

I've tried to prepare meaningful unit tests that utilizes `testing/synctest` package that is available since Go 1.25
(experiment since 1.24) but failed to do so. Extensive fuzzy tests would be a nice addition.
There is a concept of a `Clock` already integrated into implementation so a proper mock of this clock should cover
all testing needs.
For now, all I have to offer is just one happy path unit test and a stick of a hot glue just in case of hotfixes.

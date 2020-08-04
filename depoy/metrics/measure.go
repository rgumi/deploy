package metrics

import "time"

// Time returns a chan for a int64 chan
// immediatly after calling Time() a timestamp is saved
// after inserting a chan int64 into ch a new timestamp is created
// the delta between both timestamps is then inserted into chan int64
func Time() chan chan int64 {
	ch := make(chan chan int64)
	go func() {
		before := time.Now()
		select {
		case rch := <-ch:
			after := time.Now()
			delta := after.Sub(before)
			rch <- int64(delta)
			close(ch)
		}
	}()
	return ch
}

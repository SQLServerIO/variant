/* 

 Package variant provides some (well, at the time of this writing)
 useful implementations of the expvar.Var interface (which is just a
 Stringer).

*/
package variant

import (
	"container/ring"
	"expvar"
	"fmt"
	"math"
	"sort"
	"sync"
)

// represents a size bounded simple moving average
// it is thread/goroutine safe
type SimpleMovingStat struct {
	size      int
	mutex     *sync.Mutex
	values    *ring.Ring
	calculate func(*SimpleMovingStat) float64
}

// Create a new simple moving median expvar.Var. It will be
// published under `name` and maintain `size` values for
// calculating the median. 
//
// An empty name will cause it to not be published
//
// This is just a convenience helper for a SimpleMovingPercentile
func NewSimpleMovingMedian(name string, size int) *SimpleMovingStat {
	return NewSimpleMovingPercentile(name, 0.50, size)

}

// Create a new simple moving percentile expvar.Var. It will be
// published under `name` and maintain `size` values for
// calculating the percentile.
//
// percentile must be between 0 and 1 
//
// An empty name will cause it to not be published
func NewSimpleMovingPercentile(name string, percentile float64, size int) *SimpleMovingStat {
	sm := new(SimpleMovingStat)
	sm.size = size
	sm.mutex = new(sync.Mutex)
	sm.values = ring.New(size)

	sm.calculate = func(s *SimpleMovingStat) float64 {
		ary := make([]float64, 0)
		s.values.Do(func(val interface{}) {
			if val != nil {
				ary = append(ary, val.(float64))
			}
		})
		length := len(ary)
		if length == 0 {
			return 0.0
		}
		sort.Float64s(ary)
		mid := int(float64(len(ary)) * percentile)
		return ary[mid]
	}

	if name != "" {
		expvar.Publish(name, sm)
	}
	return sm

}

// Create a new simple moving average expvar.Var. It will be
// published under `name` and maintain `size` values for
// calculating the average. 
//
// An empty name will cause it to not be published
func NewSimpleMovingAverage(name string, size int) *SimpleMovingStat {
	sma := new(SimpleMovingStat)
	sma.size = size
	sma.mutex = new(sync.Mutex)
	sma.values = ring.New(size)

	sma.calculate = func(s *SimpleMovingStat) float64 {
		var sum float64 = 0.0
		var cnt int = 0

		s.values.Do(func(val interface{}) {
			if val != nil {
				cnt++
				sum = sum + val.(float64)
			}
		})
		return sum / float64(cnt)
	}

	if name != "" {
		expvar.Publish(name, sma)
	}
	return sma
}

// display the value as a string
func (s *SimpleMovingStat) String() string {
	v := s.Value()
	if math.IsNaN(v) {
		return `"NaN"`
	}
	if math.IsInf(v, 1) {
		return `"+Infinity"`
	}
	if math.IsInf(v, -1) {
		return `"-Infinity"`
	}
	return fmt.Sprintf("%f", s.Value())
}

// Append a new value to the stat
func (s *SimpleMovingStat) Update(val float64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.values.Value = val
	s.values = s.values.Next()
}

// obtain the current value
func (s *SimpleMovingStat) Value() float64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.calculate(s)
}

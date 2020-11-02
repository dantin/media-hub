package asset

import (
	"encoding/json"
	"expvar"
	"net/http"
	"runtime"
	"sort"
	"time"

	"github.com/dantin/logger"
)

var (
	statsUpdate chan *varUpdate
)

// histogram is a simple implementation of histogram expvar.Var.
// `Bounds` specifies the histogram buckets as follows (length = len(bounds)):
//     (-inf, Bounds[i]) for i = 0
//     [Bounds[i-1], Bounds[i]] for 0 < i < length
//     [Bounds[i-1], +inf]) for i = length
type histogram struct {
	Count          int64     `json:"count"`
	Sum            float64   `json:"sum"`
	CountPerBucket []int64   `json:"count_per_bucket"`
	Bounds         []float64 `json:"bounds"`
}

func (h *histogram) addSample(v float64) {
	h.Count++
	h.Sum += v
	idx := sort.SearchFloat64s(h.Bounds, v)
	h.CountPerBucket[idx]++
}

func (h *histogram) String() string {
	if r, err := json.Marshal(h); err == nil {
		return string(r)
	}
	return ""
}

type varUpdate struct {
	// Name of the variable to update
	varname string
	// Value to publish (int, float, etc.)
	value interface{}
	// Treat the count as an increment as opposite to the final value.
	inc bool
}

// Initialize stats reporting through expvar.
func statsInit(mux *http.ServeMux, path string) {
	if path == "" || path == "-" {
		return
	}

	mux.Handle(path, expvar.Handler())
	statsUpdate = make(chan *varUpdate, 1024)

	start := time.Now()
	expvar.Publish("Uptime", expvar.Func(func() interface{} {
		return time.Since(start).Seconds()
	}))
	expvar.Publish("NumGoroutines", expvar.Func(func() interface{} {
		return runtime.NumGoroutine()
	}))

	go statsUpdater()

	logger.Infof("stats: variables exposed at '%s'", path)
}

// Register integer variable. Don't check for initialization.
func statsRegisterInt(name string) {
	expvar.Publish(name, new(expvar.Int))
}

// Register histogram variable. `bounds` specifies histogram buckets/bins
// (see comment next to the `histogram` struct definition).
func statsRegisterHistogram(name string, bounds []float64) {
	numBuckets := len(bounds) + 1
	expvar.Publish(name, &histogram{
		CountPerBucket: make([]int64, numBuckets),
		Bounds:         bounds})
}

// Async publish int variable.
func statsSet(name string, val int64) {
	if statsUpdate != nil {
		select {
		case statsUpdate <- &varUpdate{name, val, false}:
		default:
		}
	}
}

// Async publish an increment (decrement) to int variable.
func statsInc(name string, val int) {
	if statsUpdate != nil {
		select {
		case statsUpdate <- &varUpdate{name, int64(val), true}:
		default:
		}
	}
}

// Async publish a value (add a sample) to a histogram variable.
func statsAddHistSample(name string, val float64) {
	if statsUpdate != nil {
		select {
		case statsUpdate <- &varUpdate{varname: name, value: val}:
		default:
		}
	}
}

// Stop publish stats.
func statsShutdown() {
	if statsUpdate != nil {
		statsUpdate <- nil
	}
}

// The go routine which actually publishes stats updates.
func statsUpdater() {
	for upd := range statsUpdate {
		if upd == nil {
			statsUpdate = nil
			// Don't care to close the channel.
			break
		}

		// Handle var update
		if ev := expvar.Get(upd.varname); ev != nil {
			switch v := ev.(type) {
			case *expvar.Int:
				count := upd.value.(int64)
				if upd.inc {
					v.Add(count)
				} else {
					v.Set(count)
				}
			case *histogram:
				val := upd.value.(float64)
				v.addSample(val)
			default:
				logger.Warnf("stats: unsupported expvar type %T", ev)
			}
		} else {
			panic("stats: update to unknown variable " + upd.varname)
		}
	}
	logger.Infof("stats: shutdown")
}
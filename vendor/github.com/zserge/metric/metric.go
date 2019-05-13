package metric

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// To mock time in tests
var now = time.Now

// Metric is a single meter (counter, gauge or histogram, optionally - with history)
type Metric interface {
	Add(n float64)
	String() string
}

// metric is an extended private interface with some additional internal
// methods used by timeseries. Counters, gauges and histograms implement it.
type metric interface {
	Metric
	Reset()
	Aggregate(roll int, samples []metric)
}

var _, _, _ metric = &counter{}, &gauge{}, &histogram{}

// NewCounter returns a counter metric that increments the value with each
// incoming number.
func NewCounter(frames ...string) Metric {
	return newMetric(func() metric { return &counter{} }, frames...)
}

// NewGauge returns a gauge metric that sums up the incoming values and returns
// mean/min/max of the resulting distribution.
func NewGauge(frames ...string) Metric {
	return newMetric(func() metric { return &gauge{} }, frames...)
}

// NewHistogram returns a histogram metric that calculates 50%, 90% and 99%
// percentiles of the incoming numbers.
func NewHistogram(frames ...string) Metric {
	return newMetric(func() metric { return &histogram{} }, frames...)
}

type timeseries struct {
	sync.Mutex
	now      time.Time
	size     int
	interval time.Duration
	total    metric
	samples  []metric
}

func (ts *timeseries) Reset() {
	ts.total.Reset()
	for _, s := range ts.samples {
		s.Reset()
	}
}

func (ts *timeseries) roll() {
	t := now()
	roll := int((t.Round(ts.interval).Sub(ts.now.Round(ts.interval))) / ts.interval)
	ts.now = t
	n := len(ts.samples)
	if roll <= 0 {
		return
	}
	if roll >= len(ts.samples) {
		ts.Reset()
	} else {
		for i := 0; i < roll; i++ {
			tmp := ts.samples[n-1]
			for j := n - 1; j > 0; j-- {
				ts.samples[j] = ts.samples[j-1]
			}
			ts.samples[0] = tmp
			ts.samples[0].Reset()
		}
		ts.total.Aggregate(roll, ts.samples)
	}
}

func (ts *timeseries) Add(n float64) {
	ts.Lock()
	defer ts.Unlock()
	ts.roll()
	ts.total.Add(n)
	ts.samples[0].Add(n)
}

func (ts *timeseries) MarshalJSON() ([]byte, error) {
	ts.Lock()
	defer ts.Unlock()
	ts.roll()
	return json.Marshal(struct {
		Interval float64  `json:"interval"`
		Total    Metric   `json:"total"`
		Samples  []metric `json:"samples"`
	}{float64(ts.interval) / float64(time.Second), ts.total, ts.samples})
}

func (ts *timeseries) String() string {
	ts.Lock()
	defer ts.Unlock()
	ts.roll()
	return ts.total.String()
}

type multimetric []*timeseries

func (mm multimetric) Add(n float64) {
	for _, m := range mm {
		m.Add(n)
	}
}

func (mm multimetric) MarshalJSON() ([]byte, error) {
	b := []byte(`{"metrics":[`)
	for i, m := range mm {
		if i != 0 {
			b = append(b, ',')
		}
		x, _ := json.Marshal(m)
		b = append(b, x...)
	}
	b = append(b, ']', '}')
	return b, nil
}

func (mm multimetric) String() string {
	return mm[len(mm)-1].String()
}

type counter struct {
	count uint64
}

func (c *counter) String() string { return strconv.FormatFloat(c.value(), 'g', -1, 64) }
func (c *counter) Reset()         { atomic.StoreUint64(&c.count, math.Float64bits(0)) }
func (c *counter) value() float64 { return math.Float64frombits(atomic.LoadUint64(&c.count)) }
func (c *counter) Add(n float64) {
	for {
		old := math.Float64frombits(atomic.LoadUint64(&c.count))
		new := old + n
		if atomic.CompareAndSwapUint64(&c.count, math.Float64bits(old), math.Float64bits(new)) {
			return
		}
	}
}
func (c *counter) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type  string  `json:"type"`
		Count float64 `json:"count"`
	}{"c", c.value()})
}

func (c *counter) Aggregate(roll int, samples []metric) {
	c.Reset()
	for _, s := range samples {
		c.Add(s.(*counter).value())
	}
}

type gauge struct {
	sync.Mutex
	value float64
	sum   float64
	min   float64
	max   float64
	count int
}

func (g *gauge) String() string { return strconv.FormatFloat(g.value, 'g', -1, 64) }
func (g *gauge) Reset() {
	g.Lock()
	defer g.Unlock()
	g.value, g.count, g.sum, g.min, g.max = 0, 0, 0, 0, 0
}
func (g *gauge) Add(n float64) {
	g.Lock()
	defer g.Unlock()
	if n < g.min || g.count == 0 {
		g.min = n
	}
	if n > g.max || g.count == 0 {
		g.max = n
	}
	g.value = n
	g.sum += n
	g.count++
}
func (g *gauge) MarshalJSON() ([]byte, error) {
	g.Lock()
	defer g.Unlock()
	return json.Marshal(struct {
		Type  string  `json:"type"`
		Value float64 `json:"value"`
		Mean  float64 `json:"mean"`
		Min   float64 `json:"min"`
		Max   float64 `json:"max"`
	}{"g", g.value, g.mean(), g.min, g.max})
}
func (g *gauge) mean() float64 {
	if g.count == 0 {
		return 0
	}
	return g.sum / float64(g.count)
}
func (g *gauge) Aggregate(roll int, samples []metric) {
	g.Reset()
	g.Lock()
	defer g.Unlock()
	for i := len(samples) - 1; i >= 0; i-- {
		s := samples[i].(*gauge)
		s.Lock()
		if s.count == 0 {
			s.Unlock()
			continue
		}
		if g.min > s.min || g.count == 0 {
			g.min = s.min
		}
		if g.max < s.max || g.count == 0 {
			g.max = s.max
		}
		g.count += s.count
		g.sum += s.sum
		g.value = s.value
		s.Unlock()
	}
}

const maxBins = 100

type bin struct {
	value float64
	count float64
}

type histogram struct {
	sync.Mutex
	bins  []bin
	total float64
}

func (h *histogram) String() string {
	return fmt.Sprintf(`{"p50":%g,"p90":%g,"p99":%g}`, h.quantile(0.5), h.quantile(0.9), h.quantile(0.99))
}

func (h *histogram) Reset() {
	h.Lock()
	defer h.Unlock()
	h.bins = nil
	h.total = 0
}

func (h *histogram) Add(n float64) {
	h.Lock()
	defer h.Unlock()
	defer h.trim()
	h.total = h.total + 1
	newbin := bin{value: n, count: 1}
	for i := range h.bins {
		if h.bins[i].value > n {
			h.bins = append(h.bins[:i], append([]bin{newbin}, h.bins[i:]...)...)
			return
		}
	}

	h.bins = append(h.bins, newbin)
}

func (h *histogram) MarshalJSON() ([]byte, error) {
	h.Lock()
	defer h.Unlock()
	return json.Marshal(struct {
		Type string  `json:"type"`
		P50  float64 `json:"p50"`
		P90  float64 `json:"p90"`
		P99  float64 `json:"p99"`
	}{"h", h.quantile(0.5), h.quantile(0.9), h.quantile(0.99)})
}

func (h *histogram) trim() {
	for len(h.bins) > maxBins {
		d := float64(0)
		i := 0
		for j := 1; j < len(h.bins); j++ {
			if dv := h.bins[j].value - h.bins[j-1].value; dv < d || j == 1 {
				d = dv
				i = j
			}
		}
		count := h.bins[i-1].count + h.bins[i].count
		merged := bin{
			value: (h.bins[i-1].value*h.bins[i-1].count + h.bins[i].value*h.bins[i].count) / count,
			count: count,
		}
		h.bins = append(h.bins[:i-1], h.bins[i:]...)
		h.bins[i-1] = merged
	}
}

func (h *histogram) bin(q float64) bin {
	count := q * h.total
	for i := range h.bins {
		count -= float64(h.bins[i].count)
		if count <= 0 {
			return h.bins[i]
		}
	}
	return bin{}
}

func (h *histogram) quantile(q float64) float64 {
	return h.bin(q).value
}

func (h *histogram) Aggregate(roll int, samples []metric) {
	h.Lock()
	defer h.Unlock()
	alpha := 2 / float64(len(samples)+1)
	h.total = 0
	for i := range h.bins {
		h.bins[i].count = h.bins[i].count * math.Pow(1-alpha, float64(roll))
		h.total = h.total + h.bins[i].count
	}
}

func newTimeseries(builder func() metric, frame string) *timeseries {
	var (
		totalNum, intervalNum   int
		totalUnit, intervalUnit rune
	)
	units := map[rune]time.Duration{
		's': time.Second,
		'm': time.Minute,
		'h': time.Hour,
		'd': time.Hour * 24,
		'w': time.Hour * 24 * 7,
		'M': time.Hour * 24 * 30,
		'y': time.Hour * 24 * 365,
	}
	fmt.Sscanf(frame, "%d%c%d%c", &totalNum, &totalUnit, &intervalNum, &intervalUnit)
	interval := units[intervalUnit] * time.Duration(intervalNum)
	if interval == 0 {
		interval = time.Minute
	}
	totalDuration := units[totalUnit] * time.Duration(totalNum)
	if totalDuration == 0 {
		totalDuration = interval * 15
	}
	n := int(totalDuration / interval)
	samples := make([]metric, n, n)
	for i := 0; i < n; i++ {
		samples[i] = builder()
	}
	totalMetric := builder()
	return &timeseries{interval: interval, total: totalMetric, samples: samples}
}

func newMetric(builder func() metric, frames ...string) Metric {
	if len(frames) == 0 {
		return builder()
	}
	if len(frames) == 1 {
		return newTimeseries(builder, frames[0])
	}
	mm := multimetric{}
	for _, frame := range frames {
		mm = append(mm, newTimeseries(builder, frame))
	}
	sort.Slice(mm, func(i, j int) bool {
		a, b := mm[i], mm[j]
		return a.interval.Seconds()*float64(len(a.samples)) < b.interval.Seconds()*float64(len(b.samples))
	})
	return mm
}

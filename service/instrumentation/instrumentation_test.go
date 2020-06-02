package instrumentation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultInstance(t *testing.T) {
	Counter(1.0, "foo.bar", 10)
	Timing(1.0, "foo.bar", time.Second)
	SetNamespace("baz")

	assert.Equal(t, "baz", defaultClient.namespace, "Expecting namespace of baz")
}

func TestWithNilStatsd(t *testing.T) {
	i := New()
	i.mtx.Lock()
	i.statsd = nil
	i.mtx.Unlock()
	assert.NotPanics(t, func() {
		i.Counter(1.0, "foo.bar", 10)
		i.Timing(1.0, "foo.bar", time.Second)
	})
}

func TestNotSavedByDefault(t *testing.T) {
	i := New()
	i.Counter(1.0, "foo.bar", 10)
	i.Timing(1.0, "foo.bar", time.Second)
	i.Gauge(1.0, "foo.bar", 10)

	assert.Nil(t, i.GetCounter("foo.bar"), "Counter should not be saved by default")
	assert.Nil(t, i.GetTiming("foo.bar"), "Timer should not be saved by default")
	assert.Nil(t, i.GetGauge("foo.bar"), "Gauge should not be saved by default")
}

func TestCounterSaved(t *testing.T) {
	i := New()
	i.SaveCounter("foo.bar")
	i.Counter(1.0, "foo.bar", 10)

	c := i.GetCounter("foo.bar")
	assert.NotNil(t, c, "Counter should be saved")
	assert.Equal(t, int64(10), c.Count(), "Counter should be 10")
	i.Counter(0.0, "foo.bar", -5)
	assert.Equal(t, int64(5), c.Count(), "Counter should be 5")
}

func TestGaugeSaved(t *testing.T) {
	i := New()
	i.SaveGauge("foo.bar")
	i.Gauge(1.0, "foo.bar", 10)

	g := i.GetGauge("foo.bar")
	assert.NotNil(t, g, "Gauge should be saved")
	assert.Equal(t, int64(10), g.Value(), "Gauge should be 10")
	i.Gauge(1.0, "foo.bar", -5)
	assert.Equal(t, int64(-5), g.Value(), "Gauge should be -5")
}

package collector

import (
	"math/rand"

	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const testSubsystem = "test"

var r *rand.Rand

func init() {
	defer trace()()
	Factories[testSubsystem] = TestTemplateCollector
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

type testMetrics struct {
	test1 float64
	test2 float64
	test3 float64
	test4 float64
	test5 float64
	test6 float64
	test7 float64
}

// TestTemplateCollector is a test collector to validate templated collection
func TestTemplateCollector() (Collector, error) {
	defer trace()()
	return NewTemplateCollector(testSubsystem, &testMetrics{})
}

func (c *testMetrics) getValue(name string) interface{} {
	defer trace()()
	var t float64
	switch name {
	case "test1":
		t = c.test1
	case "test2":
		t = c.test2
	case "test3":
		t = c.test3
	case "test4":
		t = c.test4
	case "test5":
		t = c.test5
	case "test6":
		t = c.test6
	case "test7":
		t = c.test7
	}
	return t
}

func random() float64 {
	defer trace()()
	return r.Float64() * 10
}

func (c *testMetrics) collect(m map[string]*prometheus.Desc, ch chan<- prometheus.Metric) (CollectableTemplate, error) {
	defer trace()()

	//get the actual metric from somewhere
	t := testMetrics{
		test1: random(),
		test2: random(),
		test3: random(),
		test4: random(),
		test5: random(),
		test6: random(),
		test7: random(),
	}

	//write out metric if requested in config
	core := "encore"
	if _, ok := m["Test1"]; ok {
		ch <- prometheus.MustNewConstMetric(
			m["Test1"],
			prometheus.GaugeValue,
			t.test1,
			GetLabelValues(core, "c1")...,
		)
	}
	if _, ok := m["Test2"]; ok {
		ch <- prometheus.MustNewConstMetric(
			m["Test2"],
			prometheus.GaugeValue,
			t.test2,
			GetLabelValues(core, "c2")...,
		)
	}
	if _, ok := m["Test3"]; ok {
		ch <- prometheus.MustNewConstMetric(
			m["Test3"],
			prometheus.GaugeValue,
			t.test3,
			GetLabelValues(core, "c3")...,
		)
	}
	if _, ok := m["Test4"]; ok {
		ch <- prometheus.MustNewConstMetric(
			m["Test4"],
			prometheus.GaugeValue,
			t.test4,
			GetLabelValues(core, "idle")...,
		)
	}
	if _, ok := m["Test5"]; ok {
		ch <- prometheus.MustNewConstMetric(
			m["Test5"],
			prometheus.GaugeValue,
			t.test5,
			GetLabelValues(core, "interrupt")...,
		)
	}
	if _, ok := m["Test6"]; ok {
		ch <- prometheus.MustNewConstMetric(
			m["Test6"],
			prometheus.GaugeValue,
			t.test6,
			GetLabelValues(core, "dpc")...,
		)
	}
	if _, ok := m["Test7"]; ok {
		ch <- prometheus.MustNewConstMetric(
			m["Test7"],
			prometheus.GaugeValue,
			t.test7,
			GetLabelValues(core, "privileged")...,
		)
	}

	//return the original metric in case needed for computed metrics
	return &t, nil
}

func (c *testMetrics) getMetricDesc(m map[string]*prometheus.Desc) error {
	defer trace()()
	m["Test1"] = prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, testSubsystem, "test1"),
		"Test Metric number 1",
		GetLabelNames("core", "state"),
		nil,
	)
	m["Test2"] = prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, testSubsystem, "test2"),
		"Test Metric number 2",
		GetLabelNames("core", "state"),
		nil,
	)
	m["Test3"] = prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, testSubsystem, "test3"),
		"Test Metric number 3",
		GetLabelNames("core", "state"),
		nil,
	)
	m["Test4"] = prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, testSubsystem, "test4"),
		"Test Metric number 4",
		GetLabelNames("core", "state"),
		nil,
	)
	m["Test5"] = prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, testSubsystem, "test5"),
		"Test Metric number 5",
		GetLabelNames("core", "state"),
		nil,
	)
	m["Test6"] = prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, testSubsystem, "test6"),
		"Test Metric number 6",
		GetLabelNames("core", "state"),
		nil,
	)
	m["Test7"] = prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, testSubsystem, "test7"),
		"Test Metric number 7",
		GetLabelNames("core", "state"),
		nil,
	)
	return nil
}

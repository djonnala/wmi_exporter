package collector

import (
	"strings"

	"github.com/Knetic/govaluate"
	"github.com/prometheus/common/log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sujitvp/wmi_exporter/conf"
	"github.com/sujitvp/wmi_exporter/utils"
)

//NewTemplateCollector ...
func NewTemplateCollector(subsystem string, t CollectableTemplate) (Collector, error) {
	defer trace()()
	coll := TemplateCollector{
		metricDescList: make(map[string]*prometheus.Desc),
		metricExprList: make(map[string]*govaluate.EvaluableExpression),
		tobj:           t,
	}

	// Process this collector's overrides
	v := conf.UCMConfig.Collectors.EnabledCollectors[subsystem]
	if !v.DefaultDrop {
		t.getMetricDesc(coll.metricDescList)
	}
	for _, k := range v.ExportedMetrics {
		coll.metricDescList[k.ExportName] = prometheus.NewDesc(
			prometheus.BuildFQName(v.Namespace, subsystem, k.ExportName),
			"Dynamic help for "+k.ExportName+" from config - "+k.Desc,
			GetLabelNames(),
			nil,
		)
		coll.metricMapList = append(coll.metricMapList, k)
		expr, err := govaluate.NewEvaluableExpressionWithFunctions(k.ComputeLogic, functions)
		if err != nil {
			log.Errorln("Error parsing metric expression for custom metric", k.ExportName, ". Error=>", err)
		}
		coll.metricExprList[k.ExportName] = expr
	}

	// return processed struct
	return &coll, nil
}

// A TemplateCollector is a Prometheus collector for WMI Win32_PerfRawData_PerfOS_Processor metrics
type TemplateCollector struct {
	metricDescList map[string]*prometheus.Desc
	metricMapList  []conf.MetricMap
	metricExprList map[string]*govaluate.EvaluableExpression
	tobj           CollectableTemplate
}

// Collect sends the metric values for each metric
// to the provided prometheus Metric channel.
func (c *TemplateCollector) Collect(ch chan<- prometheus.Metric) error {
	defer trace()()

	//populate with any built-in metrics
	ct, _ := c.tobj.collect(c.metricDescList, ch)

	//compute all overridden metrics
	for _, v := range c.metricMapList {
		ch <- prometheus.MustNewConstMetric(
			c.metricDescList[v.ExportName],
			getType(v.MetricType),
			compute(ct, v, c.metricExprList[v.ExportName]),
			GetLabelValues()...,
		)
	}
	return nil
}

// GetLabelNames builds Label names with configured labels from tags and metric-specific labels
func GetLabelNames(m ...string) []string {
	defer trace()()
	return append(utils.TagLabelNames, m...)
}

// GetLabelValues builds label values with configured labels from tags and metric-specific labels
func GetLabelValues(m ...string) []string {
	defer trace()()
	return append(utils.TagLabelValues, m...)
}

func getType(m string) prometheus.ValueType {
	defer trace()()
	t := prometheus.GaugeValue
	switch m {
	case "Gauge":
		t = prometheus.GaugeValue
	case "Counter":
		t = prometheus.CounterValue
	}
	return t
}

func compute(t CollectableTemplate, m conf.MetricMap, e *govaluate.EvaluableExpression) float64 {
	defer trace()()
	log.Debugf("compute: %s, %s", m.ExportName, m.ComputeLogic)
	//process
	if m.ComputedMetric {
		log.Debugf("compute(%s, %s, %s)", m.ExportName, m.ComputeLogic, e)
		params := make(map[string]interface{})
		for _, k := range e.Vars() {
			params[k] = t.getValue(k)
		}
		res, err := e.Evaluate(params)
		if err != nil {
			log.Errorf("Error computing from logic (%s). Error: %s", e, err)
			return 0
		}
		return res.(float64)
	}
	return t.getValue(m.SourceName[0]).(float64)
	//	conf.UCMConfig.Collectors.EnabledCollectors[subsystem].ExportedMetrics[]
}

//ProcessVarName processes a varname to return the metric and its labels
func ProcessVarName(name string) (string, map[string]string) {
	defer trace()()
	n := strings.Split(name, ".")
	m := make(map[string]string)
	var v []string
	for i := 1; i < len(n); i++ {
		v = strings.Split(n[i], "@")
		m[v[0]] = v[1]
	}
	return n[0], m
}

//CollectableTemplate is an interface to support templated collectors
type CollectableTemplate interface {
	getValue(name string) interface{}
	collect(m map[string]*prometheus.Desc, ch chan<- prometheus.Metric) (CollectableTemplate, error)
	getMetricDesc(m map[string]*prometheus.Desc) error
}

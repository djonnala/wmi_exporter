// Package collector returns data points from Win32_PerfRawData_PerfOS_Processor
// https://msdn.microsoft.com/en-us/library/aa394317(v=vs.90).aspx - Win32_PerfRawData_PerfOS_Processor class
package collector

import (
	"github.com/StackExchange/wmi"
	"github.com/prometheus/client_golang/prometheus"
)

const tcpuSubsystem = "tcpu"

func init() {
	defer trace()()
	Factories[tcpuSubsystem] = cpuTemplateCollector
}

type Win32_PerfFormattedData_PerfOS_Processor struct {
	Name                  string
	C1TransitionsPersec   uint64
	C2TransitionsPersec   uint64
	C3TransitionsPersec   uint64
	DPCRate               uint32
	DPCsQueuedPersec      uint32
	InterruptsPersec      uint32
	PercentC1Time         uint64
	PercentC2Time         uint64
	PercentC3Time         uint64
	PercentDPCTime        uint64
	PercentIdleTime       uint64
	PercentInterruptTime  uint64
	PercentPrivilegedTime uint64
	PercentProcessorTime  uint64
	PercentUserTime       uint64
}

// A CPUCollector is a Prometheus collector for WMI Win32_PerfFormattedData_PerfOS_Processor metrics
type tCPUCollector struct {
	lmap map[string]Win32_PerfFormattedData_PerfOS_Processor
}

// TestTemplateCollector is a test collector to validate templated collection
func cpuTemplateCollector() (Collector, error) {
	defer trace()()
	return NewTemplateCollector(tcpuSubsystem, &tCPUCollector{})
}

func (c *tCPUCollector) getValue(varname string) interface{} {
	defer trace()()
	name, m := ProcessVarName(varname)

	core := m["core"]
	mode := m["mode"]
	state := m["state"]

	if core == "" {
		t := make([]interface{}, len(c.lmap))
		i := 0
		for k := range c.lmap {
			t[i] = c.getCoreValue(name, k, mode, state)
			i++
		}
		return t
	}
	return c.getCoreValue(name, core, mode, state)
}

func (c *tCPUCollector) getCoreValue(name, core, mode, state string) interface{} {
	defer trace()()
	switch name {
	case "C1TransitionsPersec":
		return float64(c.lmap[core].C1TransitionsPersec)
	case "C2TransitionsPersec":
		return float64(c.lmap[core].C2TransitionsPersec)
	case "C3TransitionsPersec":
		return float64(c.lmap[core].C3TransitionsPersec)
	case "DPCRate":
		return float64(c.lmap[core].DPCRate)
	case "DPCsQueuedPersec":
		fallthrough
	case "dpcs_total":
		return float64(c.lmap[core].DPCsQueuedPersec)
	case "InterruptsPersec":
		fallthrough
	case "interrupts_total":
		return float64(c.lmap[core].InterruptsPersec)
	case "PercentC1Time":
		return float64(c.lmap[core].PercentC1Time)
	case "PercentC2Time":
		return float64(c.lmap[core].PercentC2Time)
	case "PercentC3Time":
		return float64(c.lmap[core].PercentC3Time)
	case "PercentDPCTime":
		return float64(c.lmap[core].PercentDPCTime)
	case "PercentIdleTime":
		return float64(c.lmap[core].PercentIdleTime)
	case "PercentInterruptTime":
		return float64(c.lmap[core].PercentInterruptTime)
	case "PercentPrivilegedTime":
		return float64(c.lmap[core].PercentPrivilegedTime)
	case "PercentProcessorTime":
		return float64(c.lmap[core].PercentProcessorTime)
	case "PercentUserTime":
		return float64(c.lmap[core].PercentUserTime)
	case "cstate_seconds_total":
		switch state {
		case "C1":
			return float64(c.lmap[core].PercentC1Time) * ticksToSecondsScaleFactor
		case "C2":
			return float64(c.lmap[core].PercentC2Time) * ticksToSecondsScaleFactor
		case "C3":
			return float64(c.lmap[core].PercentC3Time) * ticksToSecondsScaleFactor
		default:
			return []float64{float64(c.lmap[core].PercentC1Time) * ticksToSecondsScaleFactor, float64(c.lmap[core].PercentC2Time) * ticksToSecondsScaleFactor, float64(c.lmap[core].PercentC3Time) * ticksToSecondsScaleFactor}
		}
	case "time_total":
		switch mode {
		case "user":
			return float64(c.lmap[core].PercentUserTime) * ticksToSecondsScaleFactor
		case "privileged":
			return float64(c.lmap[core].PercentPrivilegedTime) * ticksToSecondsScaleFactor
		case "interrupt":
			return float64(c.lmap[core].PercentInterruptTime) * ticksToSecondsScaleFactor
		case "idle":
			return float64(c.lmap[core].PercentIdleTime) * ticksToSecondsScaleFactor
		case "dpc":
			return float64(c.lmap[core].PercentDPCTime) * ticksToSecondsScaleFactor
		default:
			return []float64{float64(c.lmap[core].PercentUserTime) * ticksToSecondsScaleFactor, float64(c.lmap[core].PercentPrivilegedTime) * ticksToSecondsScaleFactor, float64(c.lmap[core].PercentInterruptTime) * ticksToSecondsScaleFactor,
				float64(c.lmap[core].PercentIdleTime) * ticksToSecondsScaleFactor, float64(c.lmap[core].PercentDPCTime) * ticksToSecondsScaleFactor}
		}
	}
	return 0.
}

func (c *tCPUCollector) getMetricDesc(m map[string]*prometheus.Desc) error {
	defer trace()()
	m["cstate_seconds_total"] = prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, tcpuSubsystem, "cstate_seconds_total"),
		"Time spent in low-power idle state",
		GetLabelNames("core", "state"),
		nil,
	)
	m["time_total"] = prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, tcpuSubsystem, "time_total"),
		"Time that processor spent in different modes (idle, user, system, ...)",
		GetLabelNames("core", "mode"),
		nil,
	)
	m["interrupts_total"] = prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, tcpuSubsystem, "interrupts_total"),
		"Total number of received and serviced hardware interrupts",
		GetLabelNames("core"),
		nil,
	)
	m["dpcs_total"] = prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, tcpuSubsystem, "dpcs_total"),
		"Total number of received and serviced deferred procedure calls (DPCs)",
		GetLabelNames("core"),
		nil,
	)
	m["raw_metrics"] = prometheus.NewDesc(
		prometheus.BuildFQName(Namespace, tcpuSubsystem, "raw_metrics"),
		"Raw metrics returned from WMI",
		GetLabelNames("core", "wminame"),
		nil,
	)
	return nil
}

/* NOTE: This is an alternative class, but it is not as widely available. Decide which to use
type Win32_PerfRawData_Counters_ProcessorInformation struct {
	Name                        string
	AverageIdleTime             uint64
	C1TransitionsPersec         uint64
	C2TransitionsPersec         uint64
	C3TransitionsPersec         uint64
	ClockInterruptsPersec       uint64
	DPCRate                     uint64
	DPCsQueuedPersec            uint64
	IdleBreakEventsPersec       uint64
	InterruptsPersec            uint64
	ParkingStatus               uint64
	PercentC1Time               uint64
	PercentC2Time               uint64
	PercentC3Time               uint64
	PercentDPCTime              uint64
	PercentIdleTime             uint64
	PercentInterruptTime        uint64
	PercentofMaximumFrequency   uint64
	PercentPerformanceLimit     uint64
	PercentPriorityTime         uint64
	PercentPrivilegedTime       uint64
	PercentPrivilegedUtility    uint64
	PercentProcessorPerformance uint64
	PercentProcessorTime        uint64
	PercentProcessorUtility     uint64
	PercentUserTime             uint64
	PerformanceLimitFlags       uint64
	ProcessorFrequency          uint64
	ProcessorStateFlags         uint64
}*/

func (c *tCPUCollector) collect(m map[string]*prometheus.Desc, ch chan<- prometheus.Metric) (CollectableTemplate, error) {
	defer trace()()
	var dst []Win32_PerfFormattedData_PerfOS_Processor
	q := wmi.CreateQuery(&dst, "")
	if err := wmi.Query(q, &dst); err != nil {
		return nil, err
	}

	if c.lmap == nil {
		c.lmap = make(map[string]Win32_PerfFormattedData_PerfOS_Processor, len(dst))
	}

	for _, data := range dst {
		c.lmap[data.Name] = data
		core := data.Name
		if _, ok := m["cstate_seconds_total"]; ok {
			ch <- prometheus.MustNewConstMetric(
				m["cstate_seconds_total"],
				prometheus.GaugeValue,
				float64(data.PercentC1Time)*ticksToSecondsScaleFactor,
				GetLabelValues(core, "c1")...,
			)
			ch <- prometheus.MustNewConstMetric(
				m["cstate_seconds_total"],
				prometheus.GaugeValue,
				float64(data.PercentC2Time)*ticksToSecondsScaleFactor,
				GetLabelValues(core, "c2")...,
			)
			ch <- prometheus.MustNewConstMetric(
				m["cstate_seconds_total"],
				prometheus.GaugeValue,
				float64(data.PercentC3Time)*ticksToSecondsScaleFactor,
				GetLabelValues(core, "c3")...,
			)
		}
		if _, ok := m["time_total"]; ok {
			ch <- prometheus.MustNewConstMetric(
				m["time_total"],
				prometheus.GaugeValue,
				float64(data.PercentIdleTime)*ticksToSecondsScaleFactor,
				GetLabelValues(core, "idle")...,
			)
			ch <- prometheus.MustNewConstMetric(
				m["time_total"],
				prometheus.GaugeValue,
				float64(data.PercentInterruptTime)*ticksToSecondsScaleFactor,
				GetLabelValues(core, "interrupt")...,
			)
			ch <- prometheus.MustNewConstMetric(
				m["time_total"],
				prometheus.GaugeValue,
				float64(data.PercentDPCTime)*ticksToSecondsScaleFactor,
				GetLabelValues(core, "dpc")...,
			)
			ch <- prometheus.MustNewConstMetric(
				m["time_total"],
				prometheus.GaugeValue,
				float64(data.PercentPrivilegedTime)*ticksToSecondsScaleFactor,
				GetLabelValues(core, "privileged")...,
			)
			ch <- prometheus.MustNewConstMetric(
				m["time_total"],
				prometheus.GaugeValue,
				float64(data.PercentUserTime)*ticksToSecondsScaleFactor,
				GetLabelValues(core, "user")...,
			)
		}
		if _, ok := m["interrupts_total"]; ok {
			ch <- prometheus.MustNewConstMetric(
				m["interrupts_total"],
				prometheus.CounterValue,
				float64(data.InterruptsPersec),
				GetLabelValues(core)...,
			)
		}
		if _, ok := m["dpcs_total"]; ok {
			ch <- prometheus.MustNewConstMetric(
				m["dpcs_total"],
				prometheus.CounterValue,
				float64(data.DPCsQueuedPersec),
				GetLabelValues(core)...,
			)
		}
		if _, ok := m["raw_metrics"]; ok {
			ch <- prometheus.MustNewConstMetric(
				m["raw_metrics"],
				prometheus.GaugeValue,
				float64(data.PercentUserTime),
				GetLabelValues(core, "PercentUserTime")...,
			)
			ch <- prometheus.MustNewConstMetric(
				m["raw_metrics"],
				prometheus.GaugeValue,
				float64(data.PercentIdleTime),
				GetLabelValues(core, "PercentIdleTime")...,
			)
			ch <- prometheus.MustNewConstMetric(
				m["raw_metrics"],
				prometheus.GaugeValue,
				float64(data.PercentDPCTime),
				GetLabelValues(core, "PercentDPCTime")...,
			)
			ch <- prometheus.MustNewConstMetric(
				m["raw_metrics"],
				prometheus.GaugeValue,
				float64(data.PercentC3Time),
				GetLabelValues(core, "PercentC3Time")...,
			)
			ch <- prometheus.MustNewConstMetric(
				m["raw_metrics"],
				prometheus.GaugeValue,
				float64(data.PercentC2Time),
				GetLabelValues(core, "PercentC2Time")...,
			)
			ch <- prometheus.MustNewConstMetric(
				m["raw_metrics"],
				prometheus.GaugeValue,
				float64(data.PercentC1Time),
				GetLabelValues(core, "PercentC1Time")...,
			)
			ch <- prometheus.MustNewConstMetric(
				m["raw_metrics"],
				prometheus.GaugeValue,
				float64(data.PercentInterruptTime),
				GetLabelValues(core, "PercentInterruptTime")...,
			)
			ch <- prometheus.MustNewConstMetric(
				m["raw_metrics"],
				prometheus.GaugeValue,
				float64(data.PercentPrivilegedTime),
				GetLabelValues(core, "PercentPrivilegedTime")...,
			)
			ch <- prometheus.MustNewConstMetric(
				m["raw_metrics"],
				prometheus.GaugeValue,
				float64(data.PercentProcessorTime),
				GetLabelValues(core, "PercentProcessorTime")...,
			)
		}
	}
	return c, nil
}

// Package collector returns data points from Win32_PerfRawData_PerfOS_Processor
// https://msdn.microsoft.com/en-us/library/aa394317(v=vs.90).aspx - Win32_PerfRawData_PerfOS_Processor class
package collector

import (
	"log"
	"strings"

	"github.com/StackExchange/wmi"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	defer trace(O())
	Factories["cpu"] = NewCPUCollector
}

// A CPUCollector is a Prometheus collector for WMI Win32_PerfRawData_PerfOS_Processor metrics
type CPUCollector struct {
	CStateSecondsTotal *prometheus.Desc
	TimeTotal          *prometheus.Desc
	InterruptsTotal    *prometheus.Desc
	DPCsTotal          *prometheus.Desc
}

func NewCPUCollector() (Collector, error) {
	defer trace(O())
	const subsystem = "cpu"
	return &CPUCollector{
		CStateSecondsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, subsystem, "cstate_seconds_total"),
			"Time spent in low-power idle state",
			GetLabelNames("core", "state"),
			nil,
		),
		TimeTotal: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, subsystem, "time_total"),
			"Time that processor spent in different modes (idle, user, system, ...)",
			GetLabelNames("core", "mode"),
			nil,
		),

		InterruptsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, subsystem, "interrupts_total"),
			"Total number of received and serviced hardware interrupts",
			GetLabelNames("core"),
			nil,
		),
		DPCsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, subsystem, "dpcs_total"),
			"Total number of received and serviced deferred procedure calls (DPCs)",
			GetLabelNames("core"),
			nil,
		),
	}, nil
}

// Collect sends the metric values for each metric
// to the provided prometheus Metric channel.
func (c *CPUCollector) Collect(ch chan<- prometheus.Metric) error {
	defer trace(O())
	if desc, err := c.collect(ch); err != nil {
		log.Println("[ERROR] failed collecting os metrics:", desc, err)
		return err
	}
	return nil
}

type Win32_PerfRawData_PerfOS_Processor struct {
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

func (c *CPUCollector) collect(ch chan<- prometheus.Metric) (*prometheus.Desc, error) {
	defer trace(O())
	var dst []Win32_PerfRawData_PerfOS_Processor
	q := wmi.CreateQuery(&dst, "")
	if err := wmi.Query(q, &dst); err != nil {
		return nil, err
	}

	for _, data := range dst {
		log.Println("wmi.cpu->", data.Name)

		if strings.Contains(data.Name, "_Total") {
			continue
		}

		core := data.Name

		// These are only available from Win32_PerfRawData_Counters_ProcessorInformation, which is only available from Win2008R2+
		/*ch <- prometheus.MustNewConstMetric(
			c.ProcessorFrequency,
			prometheus.GaugeValue,
			float64(data.ProcessorFrequency),
			socket, core,
		)
		ch <- prometheus.MustNewConstMetric(
			c.MaximumFrequency,
			prometheus.GaugeValue,
			float64(data.PercentofMaximumFrequency)/100*float64(data.ProcessorFrequency),
			socket, core,
		)*/

		ch <- prometheus.MustNewConstMetric(
			c.CStateSecondsTotal,
			prometheus.GaugeValue,
			float64(data.PercentC1Time)*ticksToSecondsScaleFactor,
			GetLabelValues(core, "c1")...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.CStateSecondsTotal,
			prometheus.GaugeValue,
			float64(data.PercentC2Time)*ticksToSecondsScaleFactor,
			GetLabelValues(core, "c2")...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.CStateSecondsTotal,
			prometheus.GaugeValue,
			float64(data.PercentC3Time)*ticksToSecondsScaleFactor,
			GetLabelValues(core, "c3")...,
		)

		ch <- prometheus.MustNewConstMetric(
			c.TimeTotal,
			prometheus.GaugeValue,
			float64(data.PercentIdleTime)*ticksToSecondsScaleFactor,
			GetLabelValues(core, "idle")...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.TimeTotal,
			prometheus.GaugeValue,
			float64(data.PercentInterruptTime)*ticksToSecondsScaleFactor,
			GetLabelValues(core, "interrupt")...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.TimeTotal,
			prometheus.GaugeValue,
			float64(data.PercentDPCTime)*ticksToSecondsScaleFactor,
			GetLabelValues(core, "dpc")...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.TimeTotal,
			prometheus.GaugeValue,
			float64(data.PercentPrivilegedTime)*ticksToSecondsScaleFactor,
			GetLabelValues(core, "privileged")...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.TimeTotal,
			prometheus.GaugeValue,
			float64(data.PercentUserTime)*ticksToSecondsScaleFactor,
			GetLabelValues(core, "user")...,
		)

		ch <- prometheus.MustNewConstMetric(
			c.InterruptsTotal,
			prometheus.CounterValue,
			float64(data.InterruptsPersec),
			GetLabelValues(core)...,
		)
		ch <- prometheus.MustNewConstMetric(
			c.DPCsTotal,
			prometheus.CounterValue,
			float64(data.DPCsQueuedPersec),
			GetLabelValues(core)...,
		)
	}

	return nil, nil
}

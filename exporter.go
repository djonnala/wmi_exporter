package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"sync"
	"time"

	"github.com/sujitvp/go-tracey"

	"golang.org/x/sys/windows/svc"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"github.com/sujitvp/wmi_exporter/collector"
	"github.com/sujitvp/wmi_exporter/conf"
	"github.com/sujitvp/wmi_exporter/utils"
)

var trace = tracey.New(&conf.TraceConfig)

// WmiCollector implements the prometheus.Collector interface.
type WmiCollector struct {
	collectors map[string]collector.Collector
}

var (
	scrapeDurations = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace: collector.Namespace,
			Subsystem: "exporter",
			Name:      "scrape_duration_seconds",
			Help:      "wmi_exporter: Duration of a scrape job.",
		},
		[]string{"collector", "result"},
	)
)

// Describe sends all the descriptors of the collectors included to
// the provided channel.
func (coll WmiCollector) Describe(ch chan<- *prometheus.Desc) {
	defer trace()()
	scrapeDurations.Describe(ch)
}

// Collect sends the collected metrics from each of the collectors to
// prometheus. Collect could be called several times concurrently
// and thus its run is protected by a single mutex.
func (coll WmiCollector) Collect(ch chan<- prometheus.Metric) {
	defer trace()()
	wg := sync.WaitGroup{}
	wg.Add(len(coll.collectors))
	for name, c := range coll.collectors {
		go func(name string, c collector.Collector) {
			execute(name, c, ch)
			wg.Done()
		}(name, c)
	}
	wg.Wait()
	scrapeDurations.Collect(ch)
}

func execute(name string, c collector.Collector, ch chan<- prometheus.Metric) {
	defer trace()()
	begin := time.Now()
	err := c.Collect(ch)
	duration := time.Since(begin)
	var result string

	if err != nil {
		log.Errorf("ERROR: %s collector failed after %fs: %s", name, duration.Seconds(), err)
		result = "error"
	} else {
		log.Debugf("OK: %s collector succeeded after %fs.", name, duration.Seconds())
		result = "success"
	}
	scrapeDurations.WithLabelValues(name, result).Observe(duration.Seconds())
}

func loadCollectors(coll map[string]conf.CollectorSpec) (map[string]collector.Collector, error) {
	defer trace()()
	collectors := map[string]collector.Collector{}

	// remove any unsupported collectors
	for k := range coll {
		fn, ok := collector.Factories[k]
		if !ok {
			return nil, fmt.Errorf("collector '%s' not available", k)
		}
		c, err := fn()
		if err != nil {
			return nil, err
		}
		collectors[k] = c
	}
	// return final list of collectors
	return collectors, nil
}

func init() {
	defer trace()()
	prometheus.MustRegister(version.NewCollector("wmi_exporter"))
}

func main() {
	defer trace()()
	var (
		showVersion       = flag.Bool("version", false, "Print version information.")
		printCollectors   = flag.Bool("collectors.print", false, "If true, print available collectors and exit.")
		configFile        = flag.String("config.file", "", "complete path to configuration file")
		listenAddress     = flag.String("telemetry.addr", conf.DefaultPlaceholder, "host:port for WMI exporter.")
		metricsPath       = flag.String("telemetry.path", conf.DefaultPlaceholder, "URL path for surfacing collected metrics.")
		enabledCollectors = flag.String("collectors.enabled", conf.DefaultPlaceholder, "Comma-separated list of collectors to use. Use '[default]' as a placeholder for all the collectors enabled by default")
	)
	flag.Parse()

	if *showVersion {
		fmt.Fprintln(os.Stdout, version.Print("wmi_exporter"))
		os.Exit(0)
	}

	if *printCollectors {
		collectorNames := make(sort.StringSlice, 0, len(collector.Factories))
		for n := range collector.Factories {
			collectorNames = append(collectorNames, n)
		}
		collectorNames.Sort()
		fmt.Printf("Available collectors:\n")
		for _, n := range collectorNames {
			fmt.Printf(" - %s\n", n)
		}
		return
	}

	//get all configurations loaded
	conf.InitializeFromConfig(*configFile, *listenAddress, *metricsPath, *enabledCollectors)

	//fetch AWS metadata and initialize it for registration
	utils.FetchAWSMetadata()

	//schedule fetching AWS tags as labels
	quitCh := make(chan bool)
	if conf.UCMConfig.AwsTagsToLabels.Enabled {
		ticker := time.NewTicker(time.Duration(conf.UCMConfig.AwsTagsToLabels.RefreshPeriod) * time.Second)
		go func() {
			for {
				select {
				case <-ticker.C:
					utils.FetchAWSLabelTags()
				case <-quitCh:
					ticker.Stop()
					return
				}
			}
		}()
	}

	isInteractive, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatal(err)
	}

	stopCh := make(chan bool)
	if !isInteractive {
		go svc.Run(conf.UCMConfig.Service.ServiceName, &wmiExporterService{stopCh: stopCh})
	}

	// adding handler for SIGINT
	go func() {
		sigchan := make(chan os.Signal, 10)
		signal.Notify(sigchan, os.Interrupt)
		<-sigchan

		log.Infoln("Detected SIGINT. handling shutdown.")
		stopCh <- true
	}()

	collectors, err := loadCollectors(conf.UCMConfig.Collectors.EnabledCollectors)
	if err != nil {
		log.Fatalf("Couldn't load collectors: %s", err)
	}

	//	log.Infof("Enabled collectors: %v", strings.Join(keys(collectors), ", "))

	nodeCollector := WmiCollector{collectors: collectors}
	prometheus.MustRegister(nodeCollector)

	http.Handle(conf.UCMConfig.Service.MetricPath, prometheus.Handler())
	http.HandleFunc("/health", healthCheck)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, conf.UCMConfig.Service.MetricPath, http.StatusMovedPermanently)
	})

	log.Infoln("Starting WMI exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	//register to consul
	utils.Register()
	//deregister from consul when done
	defer utils.DeRegister()

	go func() {
		log.Infoln("Starting server on", conf.UCMConfig.Service.GetAddress())
		if err := http.ListenAndServe(conf.UCMConfig.Service.GetAddress(), nil); err != nil {
			log.Fatalf("cannot start WMI exporter: %s", err)
		}
	}()

	for {
		if <-stopCh {
			log.Info("Shutting down WMI exporter")
			quitCh <- true
			close(quitCh)
			break
		}
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	defer trace()()
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"status":"ok"}`)
}

func keys(m map[string]collector.Collector) []string {
	defer trace()()
	ret := make([]string, 0, len(m))
	for key := range m {
		ret = append(ret, key)
	}
	return ret
}

type wmiExporterService struct {
	stopCh chan<- bool
}

func (s *wmiExporterService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	defer trace()()
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				s.stopCh <- true
				break loop
			default:
				log.Errorf("unexpected control request #%d", c)
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

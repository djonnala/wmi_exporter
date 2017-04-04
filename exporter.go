package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/windows/svc"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"github.com/sujitvp/wmi_exporter/collector"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	consul "github.com/hashicorp/consul/api"
)

const (
	// Untagged specifies the default string to be populated when a required tag is missing/empty
	// NOTE: this value must align with the one declared in the Java code for the aws-surveiller project. See:
	// https://github.com/dev9com/aws-surveiller/src/main/java/com/tmobile/ucm/surveiller/model/Constants.java
	Untagged = "UNTAGGED"

	// timeout specifies the number of iterations after which a metric times out,
	// i.e. becomes stale and is removed from collectdCollector.valueLists. It is
	// modeled and named after the top-level "Timeout" setting of collectd.
	timeout = 2

	hostname       = "testServer"
	hostip         = "10.0.0.1"
	serviceID      = "ABCD"
	dc             = "ucm-west"
	port           = 9103
	region         = "us-west-2"
	consulEndpoint = "dev-ucm-con-w2a-a.corporate.t-mobile.com"
)

// WmiCollector implements the prometheus.Collector interface.
type WmiCollector struct {
	collectors map[string]collector.Collector
}

// newLabels converts the plugin and type instance of vl to a set of prometheus.Labels.
/*func newLabels(vl api.ValueList, md metadata) prometheus.Labels {
	labels := prometheus.Labels{}

	// Process the expectedTags. At this point of this function call, all the expectedTags should be present
	// where any missing, expected tag should have already been backfilled with a default value
	var stackValue, roleValue *string = nil, nil
	for eTag, _ := range expectedTags {
		if tagVal, ok := md.tags[eTag]; ok {
			// Special case for Stack and Role as we need to merge their tag keys/values into a single tag
			if eTag == ETagStack {
				stackValue = &tagVal
			} else if eTag == ETagRole {
				roleValue = &tagVal
			} else {
				labels[strings.ToLower(eTag)] = tagVal
			}
		}
	}

	// Special case: merge the Stack and Role into a single tag
	labels[strings.Join([]string{strings.ToLower(ETagStack), strings.ToLower(ETagRole)}, "_")] =
		strings.Join([]string{*stackValue, *roleValue}, "_")

	// TODO: Extra-defensive? Validate all required tags are present?

	// Additional tags
	labels["host"] = md.instanceId
	labels["instance"] = vl.Host

	if vl.PluginInstance != "" {
		labels[vl.Plugin] = vl.PluginInstance
	}

	if vl.TypeInstance != "" {
		if vl.PluginInstance == "" {
			labels[vl.Plugin] = vl.TypeInstance
		} else {
			labels["type"] = vl.TypeInstance
		}
	}

	log.Debugf("DSNames: %v, Values: %v, Type: %v, labels: %v", vl.DSNames, vl.Values, vl.Type, labels)

	return labels
}
*/

// Register this node to consul
func Register() {
	//get EC2 metadata
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))
	svc := ec2metadata.New(sess)
	var tags []string
	if svc.Available() {
		iddoc, err := svc.GetInstanceIdentityDocument()
		if err != nil {
			log.Error(err)
		}
		tags := make([]string, 5)
		tags[0] = "AvailabilityZone=" + iddoc.AvailabilityZone + ";"
		tags[1] = "Region=" + iddoc.Region + ";"
		tags[2] = "InstanceID=" + iddoc.InstanceID + ";"
		tags[3] = "InstanceType=" + iddoc.InstanceType + ";"
		tags[4] = "AccountID=" + iddoc.AccountID + ";"

	}

	//prepare for consul registration
	reg := consul.CatalogRegistration{
		Node:       hostname,
		Address:    hostip,
		Datacenter: dc,
		Service: &consul.AgentService{
			ID:      serviceID,
			Service: serviceName,
			Tags:    tags,
			Port:    port,
			Address: hostip,
		},
		Check: &consul.AgentCheck{
			Node:      hostname,
			CheckID:   "service:" + serviceID,
			Name:      serviceID + " health check",
			Status:    "passing",
			ServiceID: serviceID,
		},
	}

	//Get the Consul client
	config := consul.DefaultNonPooledConfig()
	config.Address = consulEndpoint
	client, err := consul.NewClient(config)
	if err != nil {
		log.Error(err)
	}
	catalog := client.Catalog()

	//make the API call to register
	w, err := catalog.Register(&reg, &consul.WriteOptions{})
	if err != nil {
		log.Error(err)
	} else {
		log.Debugf("OK: Consul registration succeeded after %f ns.", w.RequestTime.Nanoseconds())
	}

}

// DeRegister a service with consul local agent
func DeRegister() {
	//func (c *Catalog) Deregister(dereg *CatalogDeregistration, q *WriteOptions) (*WriteMeta, error)
	dereg := consul.CatalogDeregistration{
		Node:       hostname,
		Datacenter: dc,
		ServiceID:  serviceID,
	}
	//Get the Consul client
	config := consul.DefaultNonPooledConfig()
	config.Address = consulEndpoint
	client, err := consul.NewClient(config)
	if err != nil {
		log.Error(err)
	}
	catalog := client.Catalog()

	//make the API call to register
	w, err := catalog.Deregister(&dereg, nil)
	if err != nil {
		log.Error(err)
	} else {
		log.Debugf("OK: Consul deregistration succeeded after %f ns.", w.RequestTime.Nanoseconds())
	}
}

const (
	defaultCollectors            = "cpu,cs,logical_disk,net,os,service,system"
	defaultCollectorsPlaceholder = "[defaults]"
	serviceName                  = "wmi_exporter"
)

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

//	consulConfig = <needs to be created & populated based on the input flags for endpoing>
//  hostname = <need to get the hostname resolved and into this variable/constant>
//  hostip = <need to get the IP resolved and into this variable/constant>
//  consulConfigFile = <needs to be setup based on the flag >
)

// Describe sends all the descriptors of the collectors included to
// the provided channel.
func (coll WmiCollector) Describe(ch chan<- *prometheus.Desc) {
	scrapeDurations.Describe(ch)
}

// Collect sends the collected metrics from each of the collectors to
// prometheus. Collect could be called several times concurrently
// and thus its run is protected by a single mutex.
func (coll WmiCollector) Collect(ch chan<- prometheus.Metric) {
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

func filterAvailableCollectors(collectors string) string {
	var availableCollectors []string
	for _, c := range strings.Split(collectors, ",") {
		_, ok := collector.Factories[c]
		if ok {
			availableCollectors = append(availableCollectors, c)
		}
	}
	return strings.Join(availableCollectors, ",")
}

func execute(name string, c collector.Collector, ch chan<- prometheus.Metric) {
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

func expandEnabledCollectors(enabled string) []string {
	expanded := strings.Replace(enabled, defaultCollectorsPlaceholder, defaultCollectors, -1)
	separated := strings.Split(expanded, ",")
	unique := map[string]bool{}
	for _, s := range separated {
		if s != "" {
			unique[s] = true
		}
	}
	result := make([]string, 0, len(unique))
	for s, _ := range unique {
		result = append(result, s)
	}
	return result
}

func loadCollectors(list string) (map[string]collector.Collector, error) {
	collectors := map[string]collector.Collector{}
	enabled := expandEnabledCollectors(list)

	for _, name := range enabled {
		fn, ok := collector.Factories[name]
		if !ok {
			return nil, fmt.Errorf("collector '%s' not available", name)
		}
		c, err := fn()
		if err != nil {
			return nil, err
		}
		collectors[name] = c
	}
	return collectors, nil
}

func init() {
	prometheus.MustRegister(version.NewCollector("wmi_exporter"))
}

func main() {
	var (
		showVersion       = flag.Bool("version", false, "Print version information.")
		listenAddress     = flag.String("telemetry.addr", ":9182", "host:port for WMI exporter.")
		metricsPath       = flag.String("telemetry.path", "/metrics", "URL path for surfacing collected metrics.")
		enabledCollectors = flag.String("collectors.enabled", filterAvailableCollectors(defaultCollectors), "Comma-separated list of collectors to use. Use '[default]' as a placeholder for all the collectors enabled by default")
		printCollectors   = flag.Bool("collectors.print", false, "If true, print available collectors and exit.")
		//		metadataRefreshPeriod = flag.Int("metadata.refresh.period.min", 1, "refresh period in mins for retrieving metadata")

		// Required resource tags used for mapping to Prometheus metric labels. This set of tags needs to align with
		// those defined by a shared, UCM configuration
		/*		ETagApplication = "Application"
				ETagEnvironment = "Environment"
				ETagStack       = "Stack"
				ETagRole        = "Role"
				ETagName        = "Name"
				expectedTags    = map[string]int{
					ETagName:        1,
					ETagApplication: 1,
					ETagEnvironment: 1,
					ETagStack:       1,
					ETagRole:        1,
				}
		*/
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

	isInteractive, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatal(err)
	}

	stopCh := make(chan bool)
	if !isInteractive {
		go svc.Run(serviceName, &wmiExporterService{stopCh: stopCh})
	}

	collectors, err := loadCollectors(*enabledCollectors)
	if err != nil {
		log.Fatalf("Couldn't load collectors: %s", err)
	}

	log.Infof("Enabled collectors: %v", strings.Join(keys(collectors), ", "))

	nodeCollector := WmiCollector{collectors: collectors}
	prometheus.MustRegister(nodeCollector)

	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/health", healthCheck)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, *metricsPath, http.StatusMovedPermanently)
	})

	log.Infoln("Starting WMI exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	go func() {
		log.Infoln("Starting server on", *listenAddress)
		if err := http.ListenAndServe(*listenAddress, nil); err != nil {
			log.Fatalf("cannot start WMI exporter: %s", err)
		}
	}()

	for {
		if <-stopCh {
			log.Info("Shutting down WMI exporter")
			break
		}
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"status":"ok"}`)
}

func keys(m map[string]collector.Collector) []string {
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
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	//register to consul
	Register()
loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				//deregister from consul
				DeRegister()
				s.stopCh <- true
				break loop
			default:
				log.Error(fmt.Sprintf("unexpected control request #%d", c))
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

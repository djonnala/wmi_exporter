package conf

import (
	"strconv"

	"github.com/prometheus/common/log"

	"github.com/djonnala/go-tracey"

	"strings"

	"github.com/BurntSushi/toml"
)

//TraceConfig provides the global config for tracing method calls
var TraceConfig = tracey.Options{
	DisableTracing: true,
}
var trace = tracey.New(&TraceConfig)

// UCMConfig is the pre-loaded configuration for the service
var UCMConfig ConfigurationParameters

const (
	// defaultCollectors provides a default list of common collectors
	defaultCollectors = "cpu,cs,logical_disk,net,os,service,system"
	// defaultListenAddress points to all NICs on port 9182
	defaultListenAddress = ":9182"
	// defaultMetricsPath defines the constant /metrics
	defaultMetricsPath = "/metrics"
	// DefaultPlaceholder is the string template that consumers can use to include the entire list of default collectors when providing their list of enabled collectors
	DefaultPlaceholder = "[defaults]"
)

//ConfigurationParameters provides the struct to hold configuration parameters from config file
type ConfigurationParameters struct {
	Title string
	//serviceDiscovery captures configuration parameters needed for service discovery registration with Consul
	ServiceDiscovery ConsulConf
	//metadataReporting captures which metadata to be registered with service into consul for use during discovery
	MetadataReporting MetaDataConf
	//awsTagsToLabels captures the aws tags that should be added to reported metrics as Labels
	AwsTagsToLabels LabelConf
	//collectors captures the list of collectors to use
	Collectors CollectorConf
	//service captures agent related configurations
	Service ServiceConf
}

//ConsulConf captures configuration parameters needed for service discovery registration with Consul
type ConsulConf struct {
	Enabled             bool
	RemoteEndpoint      string
	RemotePort          int
	Datacenter          string
	ServiceID           string
	RegisterServiceName string
}

//MetaDataConf captures which metadata to be registered with service into consul for use during discovery
type MetaDataConf struct {
	Enabled    bool
	AWSRegion  string
	Attributes []TagLabelMap
}

//LabelConf captures the aws tags that should be added to reported metrics as Labels
type LabelConf struct {
	Enabled       bool
	RefreshPeriod int
	TagsToCaptue  []TagLabelMap
}

//TagLabelMap captures a mapping between one or more WMI metrics and the name it should be reported with
type TagLabelMap struct {
	TagName        []string
	LabelName      string
	MergeSeparator string
	MissingLabel   string
}

//CollectorConf captures the list of collectors to use
type CollectorConf struct {
	GoCollectionEnabled       bool
	ExporterCollectionEnabled bool
	WmiCollectionEnabled      bool
	AgentCollectionEnabled    bool
	MetricTimeout             int
	EnabledCollectors         map[string]CollectorSpec
}

// CollectorSpec ...
type CollectorSpec struct {
	Namespace       string
	DefaultDrop     bool
	ExportedMetrics []MetricMap
}

//MetricMap captures a mapping between one or more WMI metrics and the name it should be reported with
type MetricMap struct {
	SourceName     []string
	ExportName     string
	Desc           string
	MetricType     string
	ComputedMetric bool
	ComputeLogic   string
}

//ServiceConf captures agent related configurations
type ServiceConf struct {
	ListenIP           string
	ListenPort         int
	MetricPath         string
	CollectionInterval int
	ServiceName        string
}

// GetAddress returns a fully formatted host-port combination as needed for Consul Endpoint configuration, based on ip/fqdn and port entries made in config file
func (c ConsulConf) GetAddress() string {
	defer trace()()
	return "http://" + c.RemoteEndpoint + ":" + strconv.Itoa(c.RemotePort)
}

// GetAddress returns a fully formatted host-port combination as needed for binding the service startup, based on ip/fqdn and port entries made in config file
func (s ServiceConf) GetAddress() string {
	defer trace()()
	return s.ListenIP + ":" + strconv.Itoa(s.ListenPort)
}

// setAddress allows for setting the IP & Port for service binding based on command line params
func (s ServiceConf) setAddress(addr string) {
	defer trace()()
	a := strings.Split(addr, ":")
	s.ListenIP = a[0]
	p, err := strconv.Atoi(a[1])
	if err != nil {
		log.Fatalf("Cannot parse the address %s into IP=%s and Port=%s; Error=%s", addr, a[0], a[1], err)
	}
	s.ListenPort = p
}

// InitializeFromConfig reads configuration parameters from configuration file and initializes this service
func InitializeFromConfig(configfile string, listenAddress string, metricsPath string, enabledCollectors string) ConfigurationParameters {
	defer trace()()
	//	UCMConfig := ConfigurationParameters{}

	if configfile != "" {
		_, err := toml.DecodeFile(configfile, &UCMConfig)
		if err != nil {
			log.Fatalf("Cannot parse configuration file at %s. Error=%s", configfile, err)
		}
	}

	//allow override of configuration file values with command line params, as long as they are different from defaults or if the conf file does not have a value for those
	if listenAddress != DefaultPlaceholder || len(UCMConfig.Service.GetAddress()) == 0 {
		UCMConfig.Service.setAddress(strings.Replace(listenAddress, DefaultPlaceholder, defaultListenAddress, 1))
	}
	if metricsPath != DefaultPlaceholder || len(UCMConfig.Service.MetricPath) == 0 {
		UCMConfig.Service.MetricPath = strings.Replace(metricsPath, DefaultPlaceholder, defaultMetricsPath, 1)
	}
	if enabledCollectors != DefaultPlaceholder || len(UCMConfig.Collectors.EnabledCollectors) == 0 {
		for _, v := range strings.Split(enabledCollectors, ",") {
			if _, ok := UCMConfig.Collectors.EnabledCollectors[v]; !ok {
				UCMConfig.Collectors.EnabledCollectors[v] = CollectorSpec{Namespace: "wmi", DefaultDrop: false}
			}
		}
		//		UCMConfig.Collectors.EnabledCollectors = strings.Replace(, DefaultPlaceholder, defaultCollectors, -1)
	}

	//at this point, conf is a fully loaded configuration now; now initialize everything from conf
	return UCMConfig
}

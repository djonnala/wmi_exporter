package main

import (
	"strconv"
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
	Enabled   bool
	AWSRegion string
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
	EnabledCollectors         string
	MetricTimeout             int
	MetricRemap               []MetricMap
}

//MetricMap captures a mapping between one or more WMI metrics and the name it should be reported with
type MetricMap struct {
	WmiMetricName  []string
	ExportName     string
	DropMetric     bool
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

func (c ConsulConf) getAddress() string {
	return "http://" + c.RemoteEndpoint + ":" + strconv.Itoa(c.RemotePort)
}

func (s ServiceConf) getAddress() string {
	return s.ListenIP + ":" + strconv.Itoa(s.ListenPort)
}

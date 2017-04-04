package main

import (
	"strconv"
)

//ConfigurationParameters provides the struct to hold configuration parameters from config file
type ConfigurationParameters struct {
	//serviceDiscovery captures configuration parameters needed for service discovery registration with Consul
	serviceDiscovery ConsulConf
	//metadataReporting captures which metadata to be registered with service into consul for use during discovery
	metadataReporting MetaDataConf
	//awsTagsToLabels captures the aws tags that should be added to reported metrics as Labels
	awsTagsToLabels LabelConf
	//collectors captures the list of collectors to use
	collectors CollectorConf
	//service captures agent related configurations
	service ServiceConf
}

//ConsulConf captures configuration parameters needed for service discovery registration with Consul
type ConsulConf struct {
	enabled     bool
	endpoint    string
	port        int
	datacenter  string
	serviceID   string
	serviceName string
}

//MetaDataConf captures which metadata to be registered with service into consul for use during discovery
type MetaDataConf struct {
	enabled   bool
	awsregion string
}

//LabelConf captures the aws tags that should be added to reported metrics as Labels
type LabelConf struct {
	enabled       bool
	refreshPeriod int
}

//CollectorConf captures the list of collectors to use
type CollectorConf struct {
	goCollectionEnabled       bool
	exporterCollectionEnabled bool
	wmiCollectionEnabled      bool
	agentCollectionEnabled    bool
	enabledCollectors         string
	metricNameMapping         []MetricMap
}

//MetricMap captures a mapping between one or more WMI metrics and the name it should be reported with
type MetricMap struct {
	wmiMetricName  []string
	exportName     string
	dropMetric     bool
	computedMetric bool
	computeLogic   string
}

//ServiceConf captures agent related configurations
type ServiceConf struct {
	listenIP           string
	listenPort         int
	metricPath         string
	collectionInterval int
}

func (c ConsulConf) getAddress() string {
	return "http://" + c.endpoint + ":" + strconv.Itoa(c.port)
}

func (s ServiceConf) getAddress() string {
	return s.listenIP + ":" + strconv.Itoa(s.listenPort)
}

package main

import (
	"github.com/prometheus/common/log"

	toml "github.com/BurntSushi/toml"
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

	hostname = "testServer"
	hostip   = "10.0.0.1"

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

// Register the wmi_exporter service from the consul endpoint
func Register() {
	if !ucmconfig.ServiceDiscovery.Enabled {
		return
	}
	var tags []string
	if ucmconfig.MetadataReporting.Enabled {
		//get EC2 metadata
		sess := session.Must(session.NewSession(&aws.Config{
			Region: aws.String(ucmconfig.MetadataReporting.AWSRegion),
		}))
		svc := ec2metadata.New(sess)
		if svc.Available() {
			iddoc, err := svc.GetInstanceIdentityDocument()
			if err != nil {
				log.Error(err)
			}
			tags := make([]string, 4)
			tags[0] = "AvailabilityZone=" + iddoc.AvailabilityZone + ";"
			tags[1] = "Region=" + iddoc.Region + ";"
			tags[2] = "InstanceType=" + iddoc.InstanceType + ";"
			tags[3] = "AccountID=" + iddoc.AccountID + ";"

		}
	}

	//prepare for consul registration
	reg := consul.CatalogRegistration{
		Node:       hostname,
		Address:    hostip,
		Datacenter: ucmconfig.ServiceDiscovery.Datacenter,
		Service: &consul.AgentService{
			ID:      ucmconfig.ServiceDiscovery.ServiceID,
			Service: ucmconfig.ServiceDiscovery.RegisterServiceName,
			Tags:    tags,
			Port:    ucmconfig.Service.ListenPort,
			Address: hostip,
		},
		Check: &consul.AgentCheck{
			Node:      hostname,
			CheckID:   "service:" + ucmconfig.ServiceDiscovery.ServiceID,
			Name:      ucmconfig.ServiceDiscovery.ServiceID + " health check",
			Status:    consul.HealthPassing,
			ServiceID: ucmconfig.ServiceDiscovery.ServiceID,
		},
	}

	//Get the Consul client
	cconfig := consul.DefaultNonPooledConfig()
	cconfig.Address = ucmconfig.ServiceDiscovery.getAddress()
	client, err := consul.NewClient(cconfig)
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

// DeRegister the wmi_exporter service from the consul endpoint
func DeRegister() {
	if !ucmconfig.ServiceDiscovery.Enabled {
		return
	}
	//func (c *Catalog) Deregister(dereg *CatalogDeregistration, q *WriteOptions) (*WriteMeta, error)
	dereg := consul.CatalogDeregistration{
		Node:       hostname,
		Datacenter: ucmconfig.ServiceDiscovery.Datacenter,
		ServiceID:  ucmconfig.ServiceDiscovery.ServiceID,
	}
	//Get the Consul client
	cconfig := consul.DefaultNonPooledConfig()
	cconfig.Address = ucmconfig.ServiceDiscovery.getAddress()
	client, err := consul.NewClient(cconfig)
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

// InitializeFromConfig reads configuration parameters from configuration file and initializes this service
func InitializeFromConfig(configfile string, listenAddress string, metricsPath string, enabledCollectors string) ConfigurationParameters {
	conf := ConfigurationParameters{}

	if configfile == "" {
		return conf
	}

	_, err := toml.DecodeFile(configfile, &conf)
	if err != nil {
		log.Fatalf("Cannot parse configuration file at %s. Error=%s", configfile, err)
	}
	//at this point, conf is a fully loaded configuration now; now initialize everything from conf
	return conf
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

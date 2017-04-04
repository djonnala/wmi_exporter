package main

import (
	"github.com/prometheus/common/log"

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

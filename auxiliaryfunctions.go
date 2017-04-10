package main

import (
	"github.com/prometheus/common/log"

	"os"

	"net"

	toml "github.com/BurntSushi/toml"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	consul "github.com/hashicorp/consul/api"
)

var hostname string
var hostip []string
var tags map[string]string

func init() {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		log.Errorln("Error retrieving the hostname.", err)
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Errorln("Error retrieving the IP.", err)
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && !ipnet.IP.IsInterfaceLocalMulticast() &&
			!ipnet.IP.IsLinkLocalMulticast() && !ipnet.IP.IsLinkLocalUnicast() && !ipnet.IP.IsMulticast() && !ipnet.IP.IsUnspecified() {
			if ipnet.IP.To4() != nil {
				//this gets me the first IP.
				hostip = append(hostip, ipnet.IP.String())
			}
		}
	}
	if len(hostip) > 1 {
		log.Warnf("Multiple IPs (%d) detected: %v. Using only the first one.", len(hostip), hostip)
	}

	if addrs != nil {
		log.Infoln("Just logging")
	}
}

func fetchAWSMetadata() {
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
		tags := make([]string, 5)
		tags[0] = "AvailabilityZone=" + iddoc.AvailabilityZone + ";"
		tags[1] = "Region=" + iddoc.Region + ";"
		tags[2] = "InstanceType=" + iddoc.InstanceType + ";"
		tags[3] = "AccountID=" + iddoc.AccountID + ";"

	}
}

func metadataToTags() []string {
	var ntags []string
	if ucmconfig.MetadataReporting.Enabled {

	}
	return ntags
}

// Register the wmi_exporter service from the consul endpoint
func Register() {
	if !ucmconfig.ServiceDiscovery.Enabled {
		return
	}

	//prepare for consul registration
	reg := consul.CatalogRegistration{
		Node:       hostname,
		Address:    hostip[0],
		Datacenter: ucmconfig.ServiceDiscovery.Datacenter,
		Service: &consul.AgentService{
			ID:      ucmconfig.ServiceDiscovery.ServiceID,
			Service: ucmconfig.ServiceDiscovery.RegisterServiceName,
			Tags:    metadataToTags(),
			Port:    ucmconfig.Service.ListenPort,
			Address: hostip[0],
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
		log.Infof("OK: Consul registration succeeded after %f ns.", w.RequestTime.Nanoseconds())
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
		log.Infof("OK: Consul deregistration succeeded after %f ns.", w.RequestTime.Nanoseconds())
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

	//fetch AWS Metadata, used for registration
	fetchAWSMetadata()

	//at this point, conf is a fully loaded configuration now; now initialize everything from conf
	return conf
}

/*
// tagsAsLabels converst configuration driven  AWS tags to a set of prometheus.Labels.
func tagsAsLabels(md metadata) prometheus.Labels {
	labels := prometheus.Labels{}
	for _, tagmap := range ucmconfig.AwsTagsToLabels.TagsToCaptue {
		if len(tagmap.TagName) > 1 {
			var cVal []string
			for _, eTag := range tagmap.TagName {
				if tagVal, ok := md.tags[eTag]; ok {
					cVal = append(cVal, tagVal)
				} else {
					cVal = append(cVal, tagmap.MissingLabel)
				}
			}
			labels[strings.ToLower(tagmap.LabelName)] = strings.Join(cVal, tagmap.MergeSeparator)
		} else {
			if tagVal, ok := md.tags[tagmap.TagName[0]]; ok {
				labels[strings.ToLower(tagmap.LabelName)] = tagVal
			} else {
				labels[strings.ToLower(tagmap.LabelName)] = tagmap.MissingLabel
			}
		}
	}

	labels["instance"] = hostname
	//TODO: add other metric type based labels if needed
	return labels
}

func refreshMetadata(c *collectdCollector) {
	log.Info("refresh metadata")

	var expectedTags map[string]int

	expectedTags = make(map[string]int)
	expectedTags["Name"] = 1
	expectedTags["Application"] = 1
	expectedTags["Environment"] = 1
	expectedTags["Stack"] = 1
	expectedTags["Role"] = 1

	log.Debugf("expected tags:", expectedTags)

	// retrieve ec2 instance id
	resp, err := http.Get("http://169.254.169.254/latest/meta-data/instance-id")

	if err != nil {
		log.Errorf("Failed to call introspection api to retrieve instance id, %v", err)
		return
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)

	c.md.instanceId = string(data)

	log.Infof("instance-id: %v", c.md.instanceId)

	// retrieve ec2 private ip address
	ip_resp, ip_err := http.Get("http://169.254.169.254/latest/meta-data/local-ipv4")

	if ip_err != nil {
		log.Errorf("Failed to call introspection api to retrieve private IP address, %v", ip_err)
		return
	}

	defer ip_resp.Body.Close()

	ip_data, _ := ioutil.ReadAll(ip_resp.Body)

	c.md.privateIpAddress = string(ip_data)

	log.Infof("private-ip: %v", c.md.privateIpAddress)

	// retrieve ec2 AZ
	az_resp, az_err := http.Get("http://169.254.169.254/latest/meta-data/placement/availability-zone/")

	if az_err != nil {
		log.Errorf("Failed to call introspection api to retrieve AZ, %v", az_err)
		return
	}

	defer az_resp.Body.Close()

	az_data, az_err := ioutil.ReadAll(az_resp.Body)

	var az string = string(az_data)

	var region string = az[0 : len(az)-1]

	log.Infof("region: %v", region)

	// ec2 api call
	session, err := session.NewSession()

	if err != nil {
		log.Errorf("failed to create session %v\n", err)
		return
	}

	service := ec2.New(session, &aws.Config{Region: aws.String(region)})

	params := &ec2.DescribeTagsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("resource-id"),
				Values: []*string{
					aws.String(c.md.instanceId),
				},
			},
		},
	}

	describeTagsRes, err := service.DescribeTags(params)

	if err != nil {
		log.Errorf("failed to call ec2.describe_tags %v\n", err)
		return
	}

	for _, tag := range describeTagsRes.Tags {
		if _, ok := expectedTags[*tag.Key]; ok {
			//var s_key string = sanitize(*tag.Key)
			//var s_value string = sanitize(*tag.Value)

			c.md.tags[*tag.Key] = *tag.Value

			log.Infof("tag-key:%v, tag-value: %v", *tag.Key, c.md.tags[*tag.Key])
		}
	}

}
*/

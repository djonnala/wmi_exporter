package utils

import (
	"encoding/json"
	"net"
	"os"
	"strings"

	"github.com/sujitvp/go-tracey"

	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	consul "github.com/hashicorp/consul/api"
	"github.com/prometheus/common/log"
	"github.com/sujitvp/wmi_exporter/conf"
)

var trace = tracey.New(&conf.TraceConfig)
var hostname string
var hostip []string
var labels map[string]string

// TagLabelNames provides cached list of AWS Tags for use with Labels
var TagLabelNames []string

// TagLabelValues provides cached list of AWS Tags for use with Labels
var TagLabelValues []string

func init() {
	defer trace()()
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
}

// Register the wmi_exporter service from the consul endpoint
func Register() {
	defer trace()()
	if !conf.UCMConfig.ServiceDiscovery.Enabled {
		return
	}

	//prepare for consul registration
	reg := consul.CatalogRegistration{
		Node:       hostname,
		Address:    hostip[0],
		Datacenter: conf.UCMConfig.ServiceDiscovery.Datacenter,
		Service: &consul.AgentService{
			ID:      conf.UCMConfig.ServiceDiscovery.ServiceID,
			Service: conf.UCMConfig.ServiceDiscovery.RegisterServiceName,
			Tags:    metadataToTags(),
			Port:    conf.UCMConfig.Service.ListenPort,
			Address: hostip[0],
		},
		Check: &consul.AgentCheck{
			Node:      hostname,
			CheckID:   "service:" + conf.UCMConfig.ServiceDiscovery.ServiceID,
			Name:      conf.UCMConfig.ServiceDiscovery.ServiceID + " health check",
			Status:    consul.HealthPassing,
			ServiceID: conf.UCMConfig.ServiceDiscovery.ServiceID,
		},
	}

	//Get the Consul client
	cconfig := consul.DefaultNonPooledConfig()
	cconfig.Address = conf.UCMConfig.ServiceDiscovery.GetAddress()
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
		log.Infof("OK: Consul registration succeeded after %d ns.", w.RequestTime.Nanoseconds())
	}
}

// DeRegister the wmi_exporter service from the consul endpoint
func DeRegister() {
	defer trace()()
	if !conf.UCMConfig.ServiceDiscovery.Enabled {
		return
	}
	//func (c *Catalog) Deregister(dereg *CatalogDeregistration, q *WriteOptions) (*WriteMeta, error)
	dereg := consul.CatalogDeregistration{
		Node:       hostname,
		Datacenter: conf.UCMConfig.ServiceDiscovery.Datacenter,
		ServiceID:  conf.UCMConfig.ServiceDiscovery.ServiceID,
	}
	//Get the Consul client
	cconfig := consul.DefaultNonPooledConfig()
	cconfig.Address = conf.UCMConfig.ServiceDiscovery.GetAddress()
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

// TagsToLabels converts the currently cached tags into Prometheus Labels. It does not refresh the tags.
func getTagLabels() ([]string, []string) {
	defer trace()()
	var ntags []string
	var vtags []string
	if conf.UCMConfig.AwsTagsToLabels.Enabled {
		tags := processTagLabelMap(labels, conf.UCMConfig.MetadataReporting.Attributes)
		for k, v := range tags {
			ntags = append(ntags, strings.ToLower(k))
			vtags = append(vtags, v)
		}
	}
	return ntags, vtags
}

func getTagLabelValues(ntags []string) []string {
	defer trace()()
	var vtags []string
	if conf.UCMConfig.AwsTagsToLabels.Enabled {
		tags := processTagLabelMap(labels, conf.UCMConfig.MetadataReporting.Attributes)
		for _, k := range ntags {
			vtags = append(vtags, tags[k])
		}
	}
	return vtags
}

func metadataToTags() []string {
	defer trace()()
	var ntags []string
	if conf.UCMConfig.MetadataReporting.Enabled {
		//fetch AWS Metadata and initialize it for registration tagging, if requested
		tags := processTagLabelMap(labels, conf.UCMConfig.MetadataReporting.Attributes)
		for k, v := range tags {
			ntags = append(ntags, k+"="+v+";")
		}
	}
	return ntags
}

func processTagLabelMap(m map[string]string, tagmap []conf.TagLabelMap) map[string]string {
	defer trace()()
	t := make(map[string]string)
	for _, attr := range tagmap {
		if len(attr.TagName) > 1 {
			var cVal []string
			for _, eTag := range attr.TagName {
				if tagVal, ok := m[eTag]; ok {
					cVal = append(cVal, tagVal)
				} else {
					cVal = append(cVal, attr.MissingLabel)
				}
			}
			t[strings.ToLower(attr.LabelName)] = strings.Join(cVal, attr.MergeSeparator)
		} else {
			if tagVal, ok := m[attr.TagName[0]]; ok {
				t[strings.ToLower(attr.LabelName)] = tagVal
			} else {
				t[strings.ToLower(attr.LabelName)] = attr.MissingLabel
			}
		}
	}
	return t
}

// FetchAWSMetadata gets the instance metadata and primes the tag array, so these can be used for consul registration and metric labeling
func FetchAWSMetadata() {
	defer trace()()
	//get EC2 metadata
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(conf.UCMConfig.MetadataReporting.AWSRegion),
	}))
	svc := ec2metadata.New(sess)
	if svc.Available() {
		//		iddoc, err := svc.GetInstanceIdentityDocument()
		resp, err := svc.GetDynamicData("instance-identity/document")
		if err != nil {
			log.Errorln("EC2Metadata Request Error. Failed to get EC2 Identity document.", err)
			return
		}

		var doc interface{}
		err = json.NewDecoder(strings.NewReader(resp)).Decode(&doc)
		if err != nil {
			log.Errorln("EC2Metadata Request Error. Failed to decode EC2 instance identity document.", err)
			return
		}

		m := doc.(map[string]interface{})
		for k, v := range m {
			switch vv := v.(type) {
			case string:
				labels[k] = v.(string)
			case int:
				labels[k] = strconv.Itoa(v.(int))
			case []interface{}:
				labels[k] = strings.Join(v.([]string), ";")
			default:
				log.Infoln("Found an unknown type", vv)
			}
		}
	}
}

// FetchAWSLabelTags ...
func FetchAWSLabelTags() {
	defer trace()()
	if conf.UCMConfig.AwsTagsToLabels.Enabled && labels == nil {
		//get EC2 metadata
		sess := session.Must(session.NewSession(&aws.Config{
			Region: aws.String(conf.UCMConfig.MetadataReporting.AWSRegion),
		}))
		svc := ec2metadata.New(sess)
		if svc.Available() {
			svc := ec2.New(sess)
			params := &ec2.DescribeTagsInput{
				Filters: []*ec2.Filter{
					{
						Name: aws.String("resource-id"),
						Values: []*string{
							aws.String(labels["instanceId"]),
						},
					},
				},
			}

			describeTagsRes, err := svc.DescribeTags(params)
			if err != nil {
				log.Errorln("AWS Label Tag Request Error. Failed to call ec2.describe_tags.", err)
				return
			}
			for _, tag := range describeTagsRes.Tags {
				//tag.Key, tag.Value
				labels[*tag.Key] = *tag.Value
			}
		}
	}
	// set up the label list here so it does not have to be processed during metric collection
	// create label name & value arrays
	if len(TagLabelNames) == 0 {
		TagLabelNames, TagLabelValues = getTagLabels()
	} else {
		TagLabelValues = getTagLabelValues(TagLabelNames)
	}
}

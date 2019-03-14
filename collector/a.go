package collector

import (
	"github.com/djonnala/go-tracey"
	"github.com/djonnala/wmi_exporter/conf"
)

var trace = tracey.New(&conf.TraceConfig)

func init() {
	defer trace()()
}

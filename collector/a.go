package collector

import (
	"github.com/sujitvp/go-tracey"
	"github.com/sujitvp/wmi_exporter/conf"
)

var trace = tracey.New(&conf.TraceConfig)

func init() {
	defer trace()()
}

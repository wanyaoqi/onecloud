package host_health

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	"yunion.io/x/onecloud/pkg/cloudcommon/options"
)

type SHostHealthOptions struct {
	options.BaseOptions
	etcd.SEtcdOptions

	ServersPath     string `help:"Path for virtual server configuration files" default:"/opt/cloud/workspace/servers"`
	ShotdownServers bool   `help:"shotdown servers on disconenct with controller" default:"false"`

	HostId string `help:"Id of current host"`
}

var HostHealthOptions SHostHealthOptions

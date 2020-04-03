package host_health

import (
	"context"
	"os"

	"yunion.io/x/log"

	app_common "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/service"
	"yunion.io/x/onecloud/pkg/util/sysutils"
)

func NewEtcdOptions() *etcd.SEtcdOptions {
	return &etcd.SEtcdOptions{}
}

type SHostHealthManager struct {
	cli   *etcd.SEtcdClient
	state string
}

type SHostHealthService struct {
	*service.SServiceBase
}

func (s *SHostHealthService) InitService() {
	common_options.ParseOptions(&HostHealthOptions, os.Args, "host_health.conf", "host_health")
	err := etcd.InitDefaultEtcdClient(NewEtcdOptions())
	if err != nil {
		log.Fatalf("failed new etcd client: %s", err)
	}

	isRoot := sysutils.IsRootPermission()
	if !isRoot {
		log.Fatalf("host service must running with root permissions")
	}
}

func (s *SHostHealthService) OnExitService() {}

func (s *SHostHealthService) RunService() {
	s.InitHostHealthMonitor()
	app := app_common.InitApp(&HostHealthOptions.BaseOptions, false)
	app_common.ServeForeverWithCleanup(app, &HostHealthOptions.BaseOptions, nil)
}

func (s *SHostHealthService) InitHostHealthMonitor() {
	go s.initHostHealthMonitor()
}

func (s *SHostHealthService) initHostHealthMonitor() {
	for {
		cli := etcd.Default()
		err := cli.PutSession(context.Background(), HostHealthOptions.HostId, "online")
		if err != nil {
			s.OnEtcdDisconnected()
		} else {

		}
	}
}

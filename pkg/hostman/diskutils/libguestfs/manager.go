package libguestfs

import (
	"sync"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/pkg/errors"
)

var GuestFSWorkManager *appsrv.SWorkerManager

func Init(count int) error {
	if GuestFSWorkManager != nil {
		return errors.Errorf("repeat init")
	}
	GuestFSWorkManager = appsrv.NewWorkerManager(
		"libguestfs-worker", count, appsrv.DEFAULT_BACKLOG, false)
	return nil
}

type GuestfsManager struct {
	initGF sync.Once
}

// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package diskutils

import (
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/hostman/diskutils/libguestfs"
	"yunion.io/x/onecloud/pkg/hostman/diskutils/nbd"
	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/consts"
)

//const MAX_TRIES = 3

type SKVMGuestDisk struct {
	//imagePath string
	//nbdDev      string
	//partitions []*guestfs.SKVMGuestDiskPartition
	//lvms        []*nbd.SKVMGuestLVMPartition
	//acquiredLvm bool

	//imageRootBackFilePath string

	deployer IDeployer
}

func NewKVMGuestDisk(imagePath, driver string) *SKVMGuestDisk {
	return &SKVMGuestDisk{
		deployer: newDeployer(imagePath, driver),
	}
}

func newDeployer(imagePath, driver string) IDeployer {
	switch driver {
	case consts.DEPLOY_DRIVER_NBD:
		return nbd.NewNBDDriver(imagePath)
	case consts.DEPLOY_DRIVER_LIBGUESTFS:
		return libguestfs.NewLibguestfsDriver()
	default:
		return nbd.NewNBDDriver(imagePath)
	}

}

func (d *SKVMGuestDisk) IsLVMPartition() bool {
	return d.deployer.IsLVMPartition()
}

func (d *SKVMGuestDisk) Connect() error {
	return d.deployer.Connect()
}

func (d *SKVMGuestDisk) Disconnect() error {
	return d.deployer.Disconnect()
}

func (d *SKVMGuestDisk) DetectIsUEFISupport(rootfs fsdriver.IRootFsDriver) bool {
	partitions := d.deployer.GetPartitions()
	for i := 0; i < len(partitions); i++ {
		if partitions[i].IsMounted() {
			if rootfs.DetectIsUEFISupport(partitions[i]) {
				return true
			}
		} else {
			if partitions[i].Mount() {
				support := rootfs.DetectIsUEFISupport(partitions[i])
				partitions[i].Umount()
				if support {
					return true
				}
			}
		}
	}
	return false
}

func (d *SKVMGuestDisk) MountRootfs() fsdriver.IRootFsDriver {
	return d.MountKvmRootfs()
}

func (d *SKVMGuestDisk) MountKvmRootfs() fsdriver.IRootFsDriver {
	return d.mountKvmRootfs(false)
}
func (d *SKVMGuestDisk) mountKvmRootfs(readonly bool) fsdriver.IRootFsDriver {
	partitions := d.deployer.GetPartitions()
	for i := 0; i < len(partitions); i++ {
		mountFunc := partitions[i].Mount
		if readonly {
			mountFunc = partitions[i].MountPartReadOnly
		}
		if mountFunc() {
			if fs := guestfs.DetectRootFs(partitions[i]); fs != nil {
				log.Infof("Use rootfs %s, partition %s",
					fs, partitions[i].GetPartDev())
				return fs
			} else {
				partitions[i].Umount()
			}
		}
	}
	return nil
}

func (d *SKVMGuestDisk) MountKvmRootfsReadOnly() fsdriver.IRootFsDriver {
	return d.mountKvmRootfs(true)
}

func (d *SKVMGuestDisk) UmountKvmRootfs(fd fsdriver.IRootFsDriver) {
	if part := fd.GetPartition(); part != nil {
		part.Umount()
	}
}

func (d *SKVMGuestDisk) UmountRootfs(fd fsdriver.IRootFsDriver) {
	if fd == nil {
		return
	}
	d.UmountKvmRootfs(fd)
}

func (d *SKVMGuestDisk) MakePartition(fs string) error {
	return d.deployer.MakePartition(fs)
}

func (d *SKVMGuestDisk) FormatPartition(fs, uuid string) error {
	return d.deployer.FormatPartition(fs, uuid)
}

func (d *SKVMGuestDisk) ResizePartition() error {
	return d.deployer.ResizePartition()
}

func (d *SKVMGuestDisk) Zerofree() {
	d.deployer.Zerofree()
}

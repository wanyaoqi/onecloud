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

	"yunion.io/x/onecloud/pkg/hostman/guestfs"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
)

//const MAX_TRIES = 3

type SKVMGuestDisk struct {
	imagePath string
	//nbdDev      string
	partitions []*guestfs.SKVMGuestDiskPartition
	//lvms        []*nbd.SKVMGuestLVMPartition
	//acquiredLvm bool

	//imageRootBackFilePath string

	deployer IDeployer
}

func NewKVMGuestDisk(imagePath string) *SKVMGuestDisk {
	var ret = new(SKVMGuestDisk)
	ret.imagePath = imagePath
	ret.partitions = make([]*guestfs.SKVMGuestDiskPartition, 0)
	return ret
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
	for i := 0; i < len(d.partitions); i++ {
		if d.partitions[i].IsMounted() {
			if rootfs.DetectIsUEFISupport(d.partitions[i]) {
				return true
			}
		} else {
			if d.partitions[i].Mount() {
				support := rootfs.DetectIsUEFISupport(d.partitions[i])
				d.partitions[i].Umount()
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
	for i := 0; i < len(d.partitions); i++ {
		mountFunc := d.partitions[i].Mount
		if readonly {
			mountFunc = d.partitions[i].MountPartReadOnly
		}
		if mountFunc() {
			if fs := guestfs.DetectRootFs(d.partitions[i]); fs != nil {
				log.Infof("Use rootfs %s, partition %s",
					fs, d.partitions[i].GetPartDev())
				return fs
			} else {
				d.partitions[i].Umount()
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
	return d.deployer.Mkpartition(fs)
}

func (d *SKVMGuestDisk) FormatPartition(fs, uuid string) error {
	return d.deployer.FormatPartition(fs, uuid)
}

func (d *SKVMGuestDisk) ResizePartition() error {
	return d.deployer.ResizeDiskFs()
}

func (d *SKVMGuestDisk) Zerofree() {
	d.deployer.Zerofree()
}

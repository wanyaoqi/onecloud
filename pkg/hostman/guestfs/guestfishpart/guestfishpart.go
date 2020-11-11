package guestfishpart

import (
	"strings"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/hostman/diskutils/libguestfs/guestfish"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/kvmpart"
)

type SGuestfishDiskPartition struct {
	*kvmpart.SLocalGuestFS

	fish *guestfish.Guestfish

	// device name in guest fish
	dev string
	// one of part in guest fish filesystems
	partDev string
	// guest fish detected filesystem type
	fs string

	// is partition mounted on host filesystem
	mounted bool

	// mount as readonly
	readonly bool
}

var _ fsdriver.IDiskPartition = &SGuestfishDiskPartition{}

func NewGuestfishDiskPartition() *SGuestfishDiskPartition {
	return &SGuestfishDiskPartition{}
}

func (d *SGuestfishDiskPartition) GetPartDev() string {
	return d.partDev
}

func (d *SGuestfishDiskPartition) IsMounted() bool {
	return d.mounted
}

func (d *SGuestfishDiskPartition) Mount() bool {

}

func (d *SGuestfishDiskPartition) MountPartReadOnly() bool {

}

func (d *SGuestfishDiskPartition) Umount() bool {

}

func (d *SGuestfishDiskPartition) IsReadonly() bool {
	return d.readonly
}

func (d *SGuestfishDiskPartition) GetPhysicalPartitionType() string {
	ret, err := d.fish.SfdiskL(d.dev)
	if err != nil {
		log.Errorf("failed sfdisk-l %s: %s", d.dev, err)
		return ""
	}
	var partType string
	for i := 0; i < len(ret); i++ {
		if idx := strings.Index(ret[i], "Disk label type:"); idx > 0 {
			partType = strings.TrimSpace(string(ret[i])[idx+len("Disk label type:"):])
		}
	}
	if partType == "dos" {
		return "mbr"
	} else {
		return partType
	}
}

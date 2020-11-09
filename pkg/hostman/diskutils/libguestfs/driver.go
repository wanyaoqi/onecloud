package libguestfs

import (
	"math/rand"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/sortedmap"

	"yunion.io/x/onecloud/pkg/hostman/diskutils/nbd"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
)

const (
	DISK_LABEL_LENGTH = 4
	letterBytes       = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

type LibguestfsDriver struct {
	nbddev    string
	diskLabel string
	fsmap     sortedmap.SSortedMap
	fish      *guestfish
}

func NewLibguestfsDriver() *LibguestfsDriver {
	return &LibguestfsDriver{}
}

func (d *LibguestfsDriver) Connect() error {
	fish, err := guestfsManager.AcquireFish()
	if err != nil {
		return err
	}
	d.fish = fish

	d.nbddev = nbd.GetNBDManager().AcquireNbddev()
	if err != nil {
		return errors.Errorf("Cannot get nbd device")
	}

	lable := RandStringBytes(DISK_LABEL_LENGTH)
	err = fish.addDrive(d.nbddev, lable)
	if err != nil {
		return err
	}
	d.diskLabel = lable

	if err = fish.lvmClearFilter(); err != nil {
		return err
	}
	fsmap, err := fish.listFilesystems()
	if err != nil {
		return err
	}
	d.fsmap = fsmap

	return nil
}

func (d *LibguestfsDriver) Disconnect() error {
	if len(d.diskLabel) > 0 {
		if err := d.fish.removeDrive(); err != nil {
			return err
		}
		d.diskLabel = ""
		d.fish = nil
	}
	if len(d.nbddev) > 0 {
		nbd.GetNBDManager().ReleaseNbddev(d.nbddev)
	}
	return nil
}

func (d *LibguestfsDriver) GetPartitions() []fsdriver.IDiskPartition {
	return nil
}

func (d *LibguestfsDriver) IsLVMPartition() bool {
	return false
}

func (d *LibguestfsDriver) Zerofree() {

}

func (d *LibguestfsDriver) ResizePartition() error {
	return nil
}

func (d *LibguestfsDriver) FormatPartition(fs, uuid string) error {
	return nil
}

func (d *LibguestfsDriver) MakePartition(fs string) error {
	return nil
}

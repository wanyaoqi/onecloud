package libguestfs

import (
	"math/rand"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/sortedmap"

	"yunion.io/x/onecloud/pkg/hostman/diskutils/libguestfs/guestfish"
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

type SLibguestfsDriver struct {
	nbddev    string
	diskLabel string
	fsmap     sortedmap.SSortedMap
	fish      *guestfish.Guestfish
}

func NewLibguestfsDriver() *SLibguestfsDriver {
	return &SLibguestfsDriver{}
}

func (d *SLibguestfsDriver) Connect() error {
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
	err = fish.AddDrive(d.nbddev, lable)
	if err != nil {
		return err
	}
	d.diskLabel = lable

	if err = fish.LvmClearFilter(); err != nil {
		return err
	}
	fsmap, err := fish.ListFilesystems()
	if err != nil {
		return err
	}
	d.fsmap = fsmap

	devs := d.fsmap.Keys()
	for i := 0; i < len(devs); i++ {
		//dev := devs[i]
		//ifs, _ := d.fsmap.Get(devs[i])
		//fs := ifs.(string)

	}

	return nil
}

func (d *SLibguestfsDriver) Disconnect() error {
	if len(d.diskLabel) > 0 {
		if err := d.fish.RemoveDrive(); err != nil {
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

func (d *SLibguestfsDriver) GetPartitions() []fsdriver.IDiskPartition {
	return nil
}

func (d *SLibguestfsDriver) IsLVMPartition() bool {
	return false
}

func (d *SLibguestfsDriver) Zerofree() {

}

func (d *SLibguestfsDriver) ResizePartition() error {
	return nil
}

func (d *SLibguestfsDriver) FormatPartition(fs, uuid string) error {
	return nil
}

func (d *SLibguestfsDriver) MakePartition(fs string) error {
	return nil
}

package libguestfs

import (
	"fmt"
	"math/rand"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/sortedmap"

	"yunion.io/x/onecloud/pkg/hostman/diskutils/libguestfs/guestfish"
	"yunion.io/x/onecloud/pkg/hostman/diskutils/nbd"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/guestfishpart"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
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
	lvmParts  []string
	fsmap     sortedmap.SSortedMap
	fish      *guestfish.Guestfish

	parts []fsdriver.IDiskPartition
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
	err = fish.AddDrive(d.nbddev, lable, false)
	if err != nil {
		return err
	}
	d.diskLabel = lable

	if err = fish.LvmClearFilter(); err != nil {
		return err
	}

	devices, err := fish.ListDevices()
	if err != nil {
		return err
	}
	if len(devices) == 0 {
		return errors.Errorf("fish list devices no device found")
	}
	device := devices[0]

	fsmap, err := fish.ListFilesystems()
	if err != nil {
		return err
	}
	d.fsmap = fsmap

	lvs, err := fish.Lvs()
	if err != nil {
		return err
	}
	d.lvmParts = lvs

	keys := d.fsmap.Keys()
	for i := 0; i < len(keys); i++ {
		partDev := keys[i]
		ifs, _ := d.fsmap.Get(keys[i])
		fs := ifs.(string)
		part := guestfishpart.NewGuestfishDiskPartition(device, partDev, fs, fish)
		d.parts = append(d.parts, part)
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
	return d.parts
}

func (d *SLibguestfsDriver) IsLVMPartition() bool {
	return len(d.lvmParts) > 0
}

func (d *SLibguestfsDriver) Zerofree() {
	startTime := time.Now()
	for _, part := range d.parts {
		part.Zerofree()
	}
	log.Infof("libguestfs zerofree %d partitions takes %f seconds", len(d.parts), time.Now().Sub(startTime).Seconds())
}

func (d *SLibguestfsDriver) ResizePartition() error {
	return nil
}

func (d *SLibguestfsDriver) FormatPartition(fs, uuid string) error {
	return nil
}

func (d *SLibguestfsDriver) MakePartition(fsFormat string) error {
	var (
		labelType = "gpt"
		diskType  = fileutils2.FsFormatToDiskType(fsFormat)
	)
	if len(diskType) == 0 {
		return fmt.Errorf("Unknown fsFormat %s", fsFormat)
	}
	// TODO
	return nil
}

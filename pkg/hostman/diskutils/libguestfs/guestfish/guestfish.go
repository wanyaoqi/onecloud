package guestfish

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/sortedmap"
)

type Guestfish struct {
	*exec.Cmd

	stdin        *bufio.Writer
	stdinCloser  io.Closer
	stdout       *bufio.Scanner
	stdoutCloser io.Closer
	stderr       *bufio.Scanner
	stderrCloser io.Closer

	lock  sync.Mutex
	label string
}

const GuestFishToken = "><fs>"

func newGuestfish() (*Guestfish, error) {
	gf := &Guestfish{Cmd: exec.Command("Guestfish")}

	stdin, err := gf.StdinPipe()
	if err != nil {
		return nil, errors.Wrap(err, "Guestfish stdin pipe")
	}
	stdout, err := gf.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "Guestfish stdout pipe")
	}
	stderr, err := gf.StderrPipe()
	if err != nil {
		return nil, errors.Wrap(err, "Guestfish stderr pipe")
	}

	gf.stdin = bufio.NewWriter(stdin)
	gf.stdout = bufio.NewScanner(stdout)
	gf.stderr = bufio.NewScanner(stderr)
	gf.stdout.Split(bufio.ScanLines)
	gf.stderr.Split(bufio.ScanLines)

	gf.stdinCloser = stdin
	gf.stdoutCloser = stdout
	gf.stderrCloser = stderr

	if err = gf.Start(); err != nil {
		return nil, errors.Wrap(err, "start Guestfish")
	}

	if err = gf.Run(); err != nil {
		return nil, err
	}

	return gf, nil
}

func (fish *Guestfish) execute(cmd string) ([]string, error) {
	fish.lock.Lock()
	defer fish.lock.Unlock()
	_, err := fish.stdin.WriteString(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "exec cmd %s", cmd)
	}
	return fish.fetch()
}

func (fish *Guestfish) fetch() ([]string, error) {
	var (
		stdout, stderr = make([]string, 0), make([]string, 0)
		err            error
	)

	for fish.stdout.Scan() {
		line := fish.stdout.Text()
		log.Debugf("Guestfish stdout: %s", line)
		if strings.HasPrefix(line, GuestFishToken) {
			break
		}
		stdout = append(stdout, line)
	}

	for fish.stderr.Scan() {
		line := fish.stderr.Text()
		log.Debugf("Guestfish stderr: %s", line)
		stderr = append(stderr, line)
	}

	if len(stderr) > 0 {
		err = errors.Errorf(strings.Join(stderr, "\n"))
	}
	return stdout, err
}

/* Fetch error message from stderr, until got ><fs> from stdout */
func (fish *Guestfish) fetchError() error {
	_, err := fish.fetch()
	return err
}

func (fish *Guestfish) Run() error {
	_, err := fish.execute("run\n")
	return err
}

func (fish *Guestfish) Quit() error {
	_, err := fish.execute("quit\n")
	return err
}

func (fish *Guestfish) AddDrive(path, label string, readonly bool) error {
	cmd := fmt.Sprintf("add-drive %s label:%s\n", path, label)
	if readonly {
		cmd += " readonly:true"
	}
	_, err := fish.execute(cmd)
	if err != nil {
		return err
	}
	fish.label = label
	return nil
}

func (fish *Guestfish) RemoveDrive() error {
	if len(fish.label) == 0 {
		return errors.Errorf("no drive add")
	}
	_, err := fish.execute(fmt.Sprintf("remove-drive %s\n", fish.label))
	if err != nil {
		return err
	}
	fish.label = ""
	return err
}

func (fish *Guestfish) ListFilesystems() (sortedmap.SSortedMap, error) {
	output, err := fish.execute("list-filesystems\n")
	if err != nil {
		return nil, err
	}
	return fish.parseListFilesystemsOutput(output), nil
}

func (fish *Guestfish) parseListFilesystemsOutput(output []string) sortedmap.SSortedMap {
	/* /dev/sda1: xfs
	   /dev/centos/root: xfs
	   /dev/centos/swap: swap */
	res := sortedmap.SSortedMap{}
	for i := 0; i < len(output); i++ {
		line := output[i]
		segs := strings.Split(strings.TrimSpace(line), " ")
		log.Debugf("parse line of list filesystems: %v", segs)
		if len(segs) != 2 {
			log.Warningf("Guestfish: parse list filesystem got unwanted line: %s", line)
		}
		sortedmap.Add(res, segs[0], segs[1])
	}
	return res
}

func (fish *Guestfish) ListDevices() ([]string, error) {
	return fish.execute("list-devices\n")
}

func (fish *Guestfish) Mount(partition string) error {
	_, err := fish.execute(fmt.Sprintf("mount %s /\n", partition))
	return err
}

func (fish *Guestfish) MountLocal(localmountpoint string, readonly bool) error {
	cmd := fmt.Sprintf("mount-local %s\n", localmountpoint)
	if readonly {
		cmd += " readonly:true"
	}
	_, err := fish.execute(cmd)
	return err
}

func (fish *Guestfish) Umount(partition string) error {
	_, err := fish.execute("umount\n")
	return err
}

func (fish *Guestfish) UmountLocal() error {
	_, err := fish.execute("umount-local\n")
	return err
}

/* This should only be called after "mount_local" returns successfully.
 * The call will not return until the filesystem is unmounted. */
func (fish *Guestfish) MountLocalRun() error {
	_, err := fish.execute("umount-local-run\n")
	return err
}

/* Clears the LVM cache and performs a volume group scan. */
func (fish *Guestfish) LvmClearFilter() error {
	_, err := fish.execute("lvm-clear-filter")
	return err
}

func (fish *Guestfish) Lvs() ([]string, error) {
	return fish.execute("lvs")
}

func (fish *Guestfish) SfdiskL(dev string) ([]string, error) {
	return fish.execute(fmt.Sprintf("sfdisk-l %s", dev))
}

func (fish *Guestfish) Fsck(dev, fs string) error {
	out, err := fish.execute(fmt.Sprintf("fsck %s %s", fs, dev))
	log.Infof("FSCK: %v", out)
	return err
}

func (fish *Guestfish) Ntfsfix(dev string) error {
	out, err := fish.execute(fmt.Sprintf("ntfsfix %s", dev))
	log.Infof("NTFSFIX: %v", out)
	return err
}

func (fish *Guestfish) Zerofree(dev string) error {
	_, err := fish.execute(fmt.Sprintf("zerofree %s", dev))
	return err
}

func (fish *Guestfish) ZeroFreeSpace(dir string) error {
	_, err := fish.execute(fmt.Sprintf("zero-free-space %s", dir))
	return err
}

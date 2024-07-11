package snapshot

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"

	losetup "github.com/freddierice/go-losetup"
)

func CreateDeviceMapper(base string, overlayDir string) (*Device, error) {
	id := randomString()

	// create and truncate the overlay file
	baseInfo, err := os.Stat(base)
	if err != nil {
		return nil, fmt.Errorf("couldn't stat file %s: %v", base, err)
	}
	overlayFilename := fmt.Sprintf("%s/image-%s.diff", overlayDir, id)
	overlayFile, err := os.Create(overlayFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to create overlay file %s, %v", overlayFilename, err)
	}
	if err := overlayFile.Truncate(baseInfo.Size() + 500000000); err != nil {
		return nil, fmt.Errorf("failed to allocate overlay file %s: %v", overlayFilename, err)
	}
	defer overlayFile.Close()

	// create the loopback devices
	baseDev, err := losetup.Attach(base, 0, true)
	if err != nil {
		return nil, fmt.Errorf("failed to setup loop device for %q: %v", base, err)
	}
	overlayDev, err := losetup.Attach(overlayFilename, 0, false)
	if err != nil {
		return nil, fmt.Errorf("failed to setup loop device for %q: %v", overlayFilename, err)
	}

	// get block size of each device for dmsetup
	baseSize, err := Size512K(baseDev)
	if err != nil {
		return nil, fmt.Errorf("failed to get device size for %s: %v", baseDev.Path(), err)
	}
	overlaySize, err := Size512K(overlayDev)
	if err != nil {
		return nil, fmt.Errorf("failed to get device size for %s: %v", baseDev.Path(), err)
	}

	// do the device mapper setup
	baseName := fmt.Sprintf("base-%s", id)
	overlayName := fmt.Sprintf("overlay-%s", id)
	dmBaseTable := []byte(fmt.Sprintf("0 %d linear %s 0\n%d %d zero", baseSize, baseDev.Path(), baseSize, overlaySize))
	if err = DmCreate(baseName, dmBaseTable); err != nil {
		baseDev.Detach()
		overlayDev.Detach()
		return nil, err
	}

	basePath := fmt.Sprintf("/dev/mapper/%s", baseName)
	dmTable := []byte(fmt.Sprintf("0 %d snapshot %s %s P 8", overlaySize, basePath, overlayDev.Path()))
	if err = DmCreate(overlayName, dmTable); err != nil {
		baseDev.Detach()
		overlayDev.Detach()
		return nil, err
	}
	return &Device{baseDev, overlayDev, baseName, overlayName, overlayFilename}, nil
}

type Device struct {
	BaseDev         losetup.Device
	OverlayDev      losetup.Device
	BaseName        string // /dev/mapper/$THIS
	OverlayName     string // /dev/mapper/$THIS
	OverlayFilename string
}

func (dev *Device) Cleanup() error {
	err := dev.BaseDev.Detach()
	if err != nil {
		return err
	}
	err = dev.OverlayDev.Detach()
	if err != nil {
		return err
	}
	err = DmRemove(dev.OverlayName)
	if err != nil {
		return err
	}
	err = DmRemove(dev.BaseName)
	if err != nil {
		return err
	}
	err = os.Remove(dev.OverlayFilename)
	if err != nil {
		return err
	}
	return nil
}

func randomString() string {
	b := make([]byte, 12)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatalf("failed to generate uuid, %s", err)
	}

	return fmt.Sprintf("%X", b)
}

// copied from ignite
func Size512K(ld losetup.Device) (uint64, error) {
	data, err := ioutil.ReadFile(path.Join("/sys/class/block", path.Base(ld.Path()), "size"))
	if err != nil {
		return 0, err
	}

	// Remove the trailing newline and parse to uint64
	return strconv.ParseUint(string(data[:len(data)-1]), 10, 64)
}

// copied this helper function from ignite, it seems fine
func DmCreate(name string, table []byte) error {
	cmd := exec.Command(
		"dmsetup", "create",
		"--verifyudev", // if udevd is not running, dmsetup will manage the device node in /dev/mapper
		// julia: i have no idea what the above comment means but let's go with it i guess
		name,
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if _, err := stdin.Write(table); err != nil {
		return err
	}

	if err := stdin.Close(); err != nil {
		return err
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command %q exited with %q: %w", cmd.Args, out, err)
	}

	return nil
}

func DmRemove(name string) error {
	cmd := exec.Command("dmsetup", "remove", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command %q exited with %q: %w", cmd.Args, out, err)
	}

	return nil
}

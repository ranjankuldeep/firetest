package netlink

import (
	"fmt"
	"os/exec"
	"strconv"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

func (ops defaultNetlinkOps) AttachTap(nsPath string, tapName string, mtu int, ownerUID int, ownerGID int) error {
	nsorigin, err := netns.Get()
	if err != nil {
		return err
	}
	defer nsorigin.Close()

	nsHandle, err := netns.GetFromPath(nsPath)
	if err != nil {
		return err
	}
	defer nsHandle.Close()

	bool := nsorigin.Equal(nsHandle)
	if bool {
		return fmt.Errorf("process is not in the host namespace")
	}
	if err := WithNetNS(nsHandle, func() error {
		_, err := createTap(tapName, mtu, ownerUID, ownerGID)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// func (ops defaultNetlinkOps) CreateTap(name string, mtu int, ownerUID int, ownerGID int) (netlink.Link, error) {
// 	tapLinkAttrs := netlink.NewLinkAttrs()
// 	tapLinkAttrs.Name = name
// 	taplink := &netlink.Tuntap{
// 		LinkAttrs: tapLinkAttrs,
// 		Mode:      netlink.TUNTAP_MODE_TAP,
// 		// Queues:    1,
// 		// Flags:     netlink.TUNTAP_ONE_QUEUE | netlink.TUNTAP_VNET_HDR,
// 	}
// 	err := netlink.LinkAdd(taplink)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create tap device: %w", err)
// 	}
// 	cleanup := func(format string, a ...interface{}) (*netlink.Tuntap, error) {
// 		netlink.LinkDel(taplink)
// 		logs.Logger.Errorf(format, a...)
// 		return nil, fmt.Errorf(format, a...)
// 	}
// 	time.Sleep(1 * time.Millisecond)
// 	for _, tapFd := range taplink.Fds {
// 		err = unix.IoctlSetInt(int(tapFd.Fd()), unix.TUNSETOWNER, ownerUID)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to set tap %s owner to uid %d: %w",
// 				name, ownerUID, err)

// 		}
// 		err = unix.IoctlSetInt(int(tapFd.Fd()), unix.TUNSETGROUP, ownerGID)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to set tap %s group to gid %d: %w",
// 				name, ownerGID, err)
// 		}
// 	}
// 	err = netlink.LinkSetMTU(taplink, mtu)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to set tap device MTU to %d: %w", mtu, err)
// 	}
// 	err = netlink.LinkSetUp(taplink)
// 	if err != nil {
// 		return cleanup("failed to set tap up: %w", err)
// 	}
// 	return taplink, nil
// }

func createTap(name string, mtu int, ownerUID int, ownerGID int) (netlink.Link, error) {
	cmd := exec.Command("sudo", "ip", "tuntap", "add", "dev", name, "mode", "tap")
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to create tap device: %w", err)
	}

	cmd = exec.Command("sudo", "ip", "link", "set", "dev", name, "mtu", strconv.Itoa(mtu))
	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to set tap device MTU to %d: %w", mtu, err)
	}

	cmd = exec.Command("sudo", "chown", fmt.Sprintf("%d:%d", ownerUID, ownerGID), "/dev/net/tun")
	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to set tap %s owner to uid %d and gid %d: %w", name, ownerUID, ownerGID, err)
	}

	cmd = exec.Command("sudo", "ip", "link", "set", "dev", name, "up")
	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to bring tap device up: %w", err)
	}

	link, err := netlink.LinkByName(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get link by name: %w", err)
	}
	return link, nil
}

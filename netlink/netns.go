package netlink

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

var ErrLinkNotFound = errors.New("Link not found")

// WithNetNS switches to the given namespace, executes the provided function, and then switches back.
func WithNetNS(ns netns.NsHandle, work func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	oldNs, err := netns.Get()
	if err != nil {
		return fmt.Errorf("failed to get current namespace: %w", err)
	}
	defer oldNs.Close()

	if err := netns.Set(ns); err != nil {
		return fmt.Errorf("failed to set namespace: %w", err)
	}
	defer netns.Set(oldNs)

	if err := work(); err != nil {
		return fmt.Errorf("error executing work function in namespace: %w", err)
	}

	return nil
}

func WithNetNSLink(ns netns.NsHandle, ifName string, work func(link netlink.Link) error) error {
	return WithNetNS(ns, func() error {
		link, err := netlink.LinkByName(ifName)
		if err != nil {
			if err.Error() == errors.New("Link not found").Error() {
				return ErrLinkNotFound
			}
			return err
		}
		return work(link)
	})
}

func WithNetNSByPath(path string, work func() error) error {
	ns, err := netns.GetFromPath(path)
	if err != nil {
		return err
	}
	return WithNetNS(ns, work)
}

func NSPathByPid(pid int) string {
	return NSPathByPidWithProc("/proc", pid)
}

func NSPathByPidWithProc(procPath string, pid int) string {
	return filepath.Join(procPath, fmt.Sprint(pid), "/ns/net")
}

package netlink

import (
	"fmt"
	"os"
	"syscall"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"github.com/weaveworks/ignite/pkg/logs"
)

var MainInterface = "eth0"

// Netlink Operations
type NetlinkOps interface {
	GetLink(name string) (netlink.Link, error)
	RemoveLink(name string) error
	AttachTap(nsPath string, tapName string, mtu int, ownerUID int, ownerGID int) error
	AddTcRedirect(nsPath string, ethIface string, tuntapIface string) error
}

type defaultNetlinkOps struct {
}

func DefaultNetlinkOps() NetlinkOps {
	return &defaultNetlinkOps{}
}

func (ops *defaultNetlinkOps) AddTcRedirect(nsPath string, ethIface string, tuntapIface string) error {
	ns, err := netns.GetFromPath(nsPath)
	if err != nil {
		return err
	}
	defer ns.Close()

	return WithNetNS(ns, func() error {
		veth, err := ops.GetLink(ethIface)
		if err != nil {
			logs.Logger.Errorf("Couldn't fetch the link")
			return err
		}
		tuntap, err := ops.GetLink(tuntapIface)
		if err != nil {
			logs.Logger.Errorf("Couldn't fetch the link")
			return err
		}
		err = addIngressQdisc(veth)
		if err != nil {
			return err
		}
		err = addIngressQdisc(tuntap)
		if err != nil {
			return err
		}

		err = addRedirectFilter(veth, tuntap)
		if err != nil {
			return err
		}

		err = addRedirectFilter(tuntap, veth)
		if err != nil {
			return err
		}
		return nil
	})
}

func (ops defaultNetlinkOps) GetLink(name string) (netlink.Link, error) {
	link, err := netlink.LinkByName(name)
	if _, ok := err.(netlink.LinkNotFoundError); ok {
		return nil, &LinkNotFoundError{device: name}
	}
	return link, nil
}

func (ops defaultNetlinkOps) RemoveLink(name string) error {
	link, err := ops.GetLink(name)
	if err != nil {
		return err
	}
	err = netlink.LinkDel(link)
	if _, ok := err.(netlink.LinkNotFoundError); ok {
		return &LinkNotFoundError{device: link.Attrs().Name}
	}
	return err
}

// Add Ip addresses to the interface in the process namespace.
func (ops *defaultNetlinkOps) AddLinkIP(iface netlink.Link, ipAddr netlink.Addr) error {
	logs.Logger.Info("Adding Address")
	if err := netlink.AddrAdd(iface, &ipAddr); err != nil {
		logs.Logger.Errorf("Error Adding Ip address: %s to the interface: %s", iface.Attrs().Name, ipAddr.String())
		return err
	}
	return nil
}

// tc qdisc add dev $SRC_IFACE ingress
func addIngressQdisc(link netlink.Link) error {
	qdisc := &netlink.Ingress{
		QdiscAttrs: netlink.QdiscAttrs{
			LinkIndex: link.Attrs().Index,
			Parent:    netlink.HANDLE_INGRESS,
		},
	}

	if err := netlink.QdiscAdd(qdisc); err != nil {
		if !os.IsExist(err) {
			return err
		}
	}
	return nil
}

// tc filter add dev $SRC_IFACE parent ffff:
// protocol all
// u32 match u32 0 0
// action mirred egress mirror dev $DST_IFACE
func addRedirectFilter(linkSrc, linkDest netlink.Link) error {
	filter := &netlink.U32{
		FilterAttrs: netlink.FilterAttrs{
			LinkIndex: linkSrc.Attrs().Index,
			Parent:    netlink.MakeHandle(0xffff, 0),
			Protocol:  syscall.ETH_P_ALL,
		},
		Actions: []netlink.Action{
			&netlink.MirredAction{
				ActionAttrs: netlink.ActionAttrs{
					Action: netlink.TC_ACT_STOLEN,
				},
				MirredAction: netlink.TCA_EGRESS_MIRROR,
				Ifindex:      linkDest.Attrs().Index,
			},
		},
	}
	return netlink.FilterAdd(filter)
}

type LinkNotFoundError struct {
	device string
}

func (e LinkNotFoundError) Error() string {
	return fmt.Sprintf("did not find expected network device with name %q", e.device)
}

// Copyright (c) 2021 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"

	cli "github.com/urfave/cli/v2"
	"golang.org/x/sys/unix"
	"golang.org/x/xerrors"

	"github.com/gitpod-io/gitpod/common-go/log"
	_ "github.com/gitpod-io/gitpod/common-go/nsenter"
	"github.com/google/nftables"
	"github.com/google/nftables/expr"
	"github.com/vishvananda/netlink"
)

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:  "move-mount",
				Usage: "calls move_mount with the pipe-fd to target",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "target",
						Required: true,
					},
					&cli.IntFlag{
						Name:     "pipe-fd",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					return syscallMoveMount(c.Int("pipe-fd"), "", unix.AT_FDCWD, c.String("target"), flagMoveMountFEmptyPath)
				},
			},
			{
				Name:  "open-tree",
				Usage: "opens a and writes the resulting mountfd to the Unix pipe on the pipe-fd",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "target",
						Required: true,
					},
					&cli.IntFlag{
						Name:     "pipe-fd",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					fd, err := syscallOpenTree(unix.AT_FDCWD, c.String("target"), flagOpenTreeClone|flagAtRecursive)
					if err != nil {
						return err
					}

					err = unix.Sendmsg(c.Int("pipe-fd"), nil, unix.UnixRights(int(fd)), nil, 0)
					if err != nil {
						return err
					}

					return nil
				},
			},
			{
				Name:  "make-shared",
				Usage: "makes a mount point shared",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "target",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					return unix.Mount("none", c.String("target"), "", unix.MS_SHARED, "")
				},
			},
			{
				Name:  "mount-fusefs-mark",
				Usage: "mounts a fusefs mark",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "source",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "merged",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "upper",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "work",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "uidmapping",
						Required: false,
					},
					&cli.StringFlag{
						Name:     "gidmapping",
						Required: false,
					},
				},
				Action: func(c *cli.Context) error {
					target := filepath.Clean(c.String("merged"))
					upper := filepath.Clean(c.String("upper"))
					work := filepath.Clean(c.String("work"))
					source := filepath.Clean(c.String("source"))

					args := []string{
						fmt.Sprintf("lowerdir=%s,upperdir=%v,workdir=%v", source, upper, work),
					}

					if len(c.String("uidmapping")) > 0 {
						args = append(args, fmt.Sprintf("uidmapping=%v", c.String("uidmapping")))
					}

					if len(c.String("gidmapping")) > 0 {
						args = append(args, fmt.Sprintf("gidmapping=%v", c.String("gidmapping")))
					}

					cmd := exec.Command(
						fmt.Sprintf("%v/.supervisor/fuse-overlayfs", source),
						"-o",
						strings.Join(args, ","),
						"none",
						target,
					)
					cmd.Dir = source

					out, err := cmd.CombinedOutput()
					if err != nil {
						return xerrors.Errorf("fuse-overlayfs (%v) failed: %q\n%v",
							cmd.Args,
							string(out),
							err,
						)
					}

					return nil
				},
			},
			{
				Name:  "mount-shiftfs-mark",
				Usage: "mounts a shiftfs mark",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "source",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "target",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					return unix.Mount(c.String("source"), c.String("target"), "shiftfs", 0, "mark")
				},
			},
			{
				Name:  "mount-proc",
				Usage: "mounts proc",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "target",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					return unix.Mount("proc", c.String("target"), "proc", 0, "")
				},
			},
			{
				Name:  "mount-sysfs",
				Usage: "mounts sysfs",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "target",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					return unix.Mount("sysfs", c.String("target"), "sysfs", 0, "")
				},
			},
			{
				Name:  "unmount",
				Usage: "unmounts a mountpoint",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "target",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					return unix.Unmount(c.String("target"), 0)
				},
			},
			{
				Name:  "prepare-dev",
				Usage: "prepares a workspaces /dev directory",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:     "uid",
						Required: true,
					},
					&cli.IntFlag{
						Name:     "gid",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					err := ioutil.WriteFile("/dev/kmsg", nil, 0644)
					if err != nil {
						return err
					}

					_ = os.MkdirAll("/dev/net", 0755)
					err = unix.Mknod("/dev/net/tun", 0666|unix.S_IFCHR, int(unix.Mkdev(10, 200)))
					if err != nil {
						return err
					}
					err = os.Chmod("/dev/net/tun", os.FileMode(0666))
					if err != nil {
						return err
					}
					err = os.Chown("/dev/net/tun", c.Int("uid"), c.Int("gid"))
					if err != nil {
						return err
					}

					err = unix.Mknod("/dev/fuse", 0666|unix.S_IFCHR, int(unix.Mkdev(10, 229)))
					if err != nil {
						return err
					}
					err = os.Chmod("/dev/fuse", os.FileMode(0666))
					if err != nil {
						return err
					}
					err = os.Chown("/dev/fuse", c.Int("uid"), c.Int("gid"))
					if err != nil {
						return err
					}

					return nil
				},
			},
			{
				Name:  "setup-pair-veths",
				Usage: "set up a pair of veths",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:     "target-pid",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					containerIf, vethIf, cethIf := "eth0", "veth0", "ceth0"
					mask := net.IPv4Mask(255, 255, 255, 0)
					vethIp := net.IPNet{
						IP:   net.IPv4(10, 0, 5, 1),
						Mask: mask,
					}
					cethIp := net.IPNet{
						IP:   net.IPv4(10, 0, 5, 2),
						Mask: mask,
					}
					masqueradeAddr := net.IPNet{
						IP:   vethIp.IP.Mask(mask),
						Mask: mask,
					}
					targetPid := c.Int("target-pid")

					veth := &netlink.Veth{
						LinkAttrs: netlink.LinkAttrs{
							Name:  vethIf,
							Flags: net.FlagUp,
							MTU:   1500,
						},
						PeerName:      cethIf,
						PeerNamespace: netlink.NsPid(targetPid),
					}
					if err := netlink.LinkAdd(veth); err != nil {
						return xerrors.Errorf("link %q-%q netns failed: %v", vethIf, cethIf, err)
					}

					vethLink, err := netlink.LinkByName(vethIf)
					if err != nil {
						return xerrors.Errorf("cannot found %q netns failed: %v", vethIf, err)
					}
					if err := netlink.AddrAdd(vethLink, &netlink.Addr{IPNet: &vethIp}); err != nil {
						return xerrors.Errorf("failed to add IP address to %q: %v", vethIf, err)
					}
					if err := netlink.LinkSetUp(vethLink); err != nil {
						return xerrors.Errorf("failed to enable %q: %v", vethIf, err)
					}

					nc := &nftables.Conn{}
					nat := nc.AddTable(&nftables.Table{
						Family: nftables.TableFamilyIPv4,
						Name:   "nat",
					})

					postrouting := nc.AddChain(&nftables.Chain{
						Name:     "postrouting",
						Hooknum:  nftables.ChainHookPostrouting,
						Priority: nftables.ChainPriorityNATSource,
						Table:    nat,
						Type:     nftables.ChainTypeNAT,
					})

					// ip saddr 10.0.5.0/24 oifname "eth0" masquerade
					nc.AddRule(&nftables.Rule{
						Table: nat,
						Chain: postrouting,
						Exprs: []expr.Any{
							&expr.Payload{
								DestRegister: 1,
								Base:         expr.PayloadBaseNetworkHeader,
								Offset:       12,
								Len:          net.IPv4len,
							},
							&expr.Bitwise{
								SourceRegister: 1,
								DestRegister:   1,
								Len:            net.IPv4len,
								Mask:           masqueradeAddr.Mask,
								Xor:            net.IPv4Mask(0, 0, 0, 0),
							},
							&expr.Cmp{
								Op:       expr.CmpOpEq,
								Register: 1,
								Data:     masqueradeAddr.IP.To4(),
							},
							&expr.Meta{Key: expr.MetaKeyOIFNAME, Register: 1},
							&expr.Cmp{
								Op:       expr.CmpOpEq,
								Register: 1,
								Data:     []byte(fmt.Sprintf("%s\x00", containerIf)),
							},
							&expr.Masq{},
						},
					})

					prerouting := nc.AddChain(&nftables.Chain{
						Name:     "prerouting",
						Hooknum:  nftables.ChainHookPrerouting,
						Priority: nftables.ChainPriorityNATDest,
						Table:    nat,
						Type:     nftables.ChainTypeNAT,
					})

					// iif $containerIf tcp dport 1-65535 dnat to $cethIp:tcp dport
					nc.AddRule(&nftables.Rule{
						Table: nat,
						Chain: prerouting,
						Exprs: []expr.Any{
							&expr.Meta{Key: expr.MetaKeyIIFNAME, Register: 1},
							&expr.Cmp{
								Op:       expr.CmpOpEq,
								Register: 1,
								Data:     []byte(containerIf + "\x00"),
							},

							&expr.Meta{Key: expr.MetaKeyL4PROTO, Register: 1},
							&expr.Cmp{
								Op:       expr.CmpOpEq,
								Register: 1,
								Data:     []byte{unix.IPPROTO_TCP},
							},
							&expr.Payload{
								DestRegister: 1,
								Base:         expr.PayloadBaseTransportHeader,
								Offset:       2,
								Len:          2,
							},

							&expr.Cmp{
								Op:       expr.CmpOpGte,
								Register: 1,
								Data:     []byte{0x00, 0x01},
							},
							&expr.Cmp{
								Op:       expr.CmpOpLte,
								Register: 1,
								Data:     []byte{0xff, 0xff},
							},

							&expr.Immediate{
								Register: 2,
								Data:     cethIp.IP.To4(),
							},
							&expr.NAT{
								Type:        expr.NATTypeDestNAT,
								Family:      unix.NFPROTO_IPV4,
								RegAddrMin:  2,
								RegProtoMin: 1,
							},
						},
					})

					if err := nc.Flush(); err != nil {
						return xerrors.Errorf("failed to apply nftables: %v", err)
					}

					return nil
				},
			},
			{
				Name:  "setup-peer-veth",
				Usage: "set up a peer veth",
				Action: func(c *cli.Context) error {
					cethIf := "ceth0"
					mask := net.IPv4Mask(255, 255, 255, 0)
					cethIp := net.IPNet{
						IP:   net.IPv4(10, 0, 5, 2),
						Mask: mask,
					}
					vethIp := net.IPNet{
						IP:   net.IPv4(10, 0, 5, 1),
						Mask: mask,
					}

					cethLink, err := netlink.LinkByName(cethIf)
					if err != nil {
						return xerrors.Errorf("cannot found %q netns failed: %v", cethIf, err)
					}
					if err := netlink.AddrAdd(cethLink, &netlink.Addr{IPNet: &cethIp}); err != nil {
						return xerrors.Errorf("failed to add IP address to %q: %v", cethIf, err)
					}
					if err := netlink.LinkSetUp(cethLink); err != nil {
						return xerrors.Errorf("failed to enable %q: %v", cethIf, err)
					}

					lo, err := netlink.LinkByName("lo")
					if err != nil {
						return xerrors.Errorf("cannot found lo: %v", err)
					}
					if err := netlink.LinkSetUp(lo); err != nil {
						return xerrors.Errorf("failed to enable lo: %v", err)
					}

					defaultGw := netlink.Route{
						Scope: netlink.SCOPE_UNIVERSE,
						Gw:    vethIp.IP,
					}
					if err := netlink.RouteReplace(&defaultGw); err != nil {
						return xerrors.Errorf("failed to set up deafult gw: %v", err)
					}

					return nil
				},
			},
			{
				Name:  "enable-ip-forward",
				Usage: "enable IPv4 forwarding",
				Action: func(c *cli.Context) error {
					return os.WriteFile("/proc/sys/net/ipv4/ip_forward", []byte("1"), 0644)
				},
			},
		},
	}

	log.Init("nsinsider", "", true, false)
	err := app.Run(os.Args)
	if err != nil {
		log.WithField("instanceId", os.Getenv("GITPOD_INSTANCE_ID")).WithField("args", os.Args).Fatal(err)
	}
}

func syscallMoveMount(fromDirFD int, fromPath string, toDirFD int, toPath string, flags uintptr) error {
	fromPathP, err := unix.BytePtrFromString(fromPath)
	if err != nil {
		return err
	}
	toPathP, err := unix.BytePtrFromString(toPath)
	if err != nil {
		return err
	}

	_, _, errno := unix.Syscall6(unix.SYS_MOVE_MOUNT, uintptr(fromDirFD), uintptr(unsafe.Pointer(fromPathP)), uintptr(toDirFD), uintptr(unsafe.Pointer(toPathP)), flags, 0)
	if errno != 0 {
		return errno
	}

	return nil
}

const (
	// FlagMoveMountFEmptyPath: empty from path permitted: https://elixir.bootlin.com/linux/latest/source/include/uapi/linux/mount.h#L70
	flagMoveMountFEmptyPath = 0x00000004
)

func syscallOpenTree(dfd int, path string, flags uintptr) (fd uintptr, err error) {
	p1, err := unix.BytePtrFromString(path)
	if err != nil {
		return 0, err
	}
	fd, _, errno := unix.Syscall(unix.SYS_OPEN_TREE, uintptr(dfd), uintptr(unsafe.Pointer(p1)), flags)
	if errno != 0 {
		return 0, errno
	}

	return fd, nil
}

const (
	// FlagOpenTreeClone: https://elixir.bootlin.com/linux/latest/source/include/uapi/linux/mount.h#L62
	flagOpenTreeClone = 1
	// FlagAtRecursive: Apply to the entire subtree: https://elixir.bootlin.com/linux/latest/source/include/uapi/linux/fcntl.h#L112
	flagAtRecursive = 0x8000
)

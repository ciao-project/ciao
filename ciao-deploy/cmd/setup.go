// Copyright © 2017 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/ciao-project/ciao/ciao-deploy/deploy"
	"github.com/spf13/cobra"
	"github.com/vishvananda/netlink"
)

type cnciFlag string

func (f *cnciFlag) String() string {
	return string(*f)
}

func (f *cnciFlag) Type() string {
	return "string"
}

func (f *cnciFlag) Set(val string) error {
	if val != "tiny" && val != "medium" && val != "large" {
		return fmt.Errorf("tiny, medium or large expected")
	}

	*f = cnciFlag(val)

	return nil
}

var clusterConf = &deploy.ClusterConfiguration{}
var force bool
var localLauncher bool
var cnciSize cnciFlag = "large"

func setup() int {
	ctx, cancelFunc := getSignalContext()
	defer cancelFunc()

	clusterConf.CNCISize = cnciSize.String()
	err := deploy.SetupMaster(ctx, force, imageCacheDirectory, clusterConf, localLauncher)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error provisioning system as master: %v\n", err)
		return 1
	}

	if localLauncher {
		err = deploy.SetupLocalLauncher(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error setting up local launcher: %v\n", err)
		}
	}

	deploy.OutputEnvironment(clusterConf)
	return 0
}

// setupCmd represents the master command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Sets up this machine as the master node",
	Long:  "Configures this machine as the master node for a ciao cluster",
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(setup())
	},
}

func validPhysicalLink(link netlink.Link) bool {
	phyDevice := true

	switch link.Type() {
	case "device":
	case "bond":
	case "vlan":
	case "macvlan":
	case "bridge":
		if strings.HasPrefix(link.Attrs().Name, "docker") ||
			strings.HasPrefix(link.Attrs().Name, "virbr") {
			phyDevice = false
		}
	default:
		phyDevice = false
	}

	if (link.Attrs().Flags & net.FlagLoopback) != 0 {
		return false
	}

	return phyDevice
}

func getFirstPhyDevice() (string, string) {
	links, err := netlink.LinkList()
	if err != nil {
		return "", ""
	}

	for _, link := range links {
		if !validPhysicalLink(link) {
			continue
		}

		addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
		if err != nil || len(addrs) == 0 {
			continue
		}

		return addrs[0].IPNet.String(), addrs[0].IP.String()
	}

	return "", ""
}

func init() {
	RootCmd.AddCommand(setupCmd)

	hostNetwork, hostIP := getFirstPhyDevice()

	// For configuration file generation
	setupCmd.Flags().StringVar(&clusterConf.CephID, "ceph-id", "admin", "The ceph id for the storage cluster")
	setupCmd.Flags().StringVar(&clusterConf.HTTPSCaCertPath, "https-ca-cert", "", "Path to CA certificate for HTTP service")
	setupCmd.Flags().StringVar(&clusterConf.HTTPSCertPath, "https-cert", "", "Path to certificate for HTTPS service")
	setupCmd.Flags().StringVar(&clusterConf.AdminSSHKeyPath, "admin-ssh-key", "", "Path to SSH public key for accessing CNCI")
	setupCmd.Flags().StringVar(&clusterConf.ComputeNet, "compute-net", hostNetwork, "Network range for compute network")
	setupCmd.Flags().StringVar(&clusterConf.MgmtNet, "mgmt-net", hostNetwork, "Network range for management network")
	setupCmd.Flags().StringVar(&clusterConf.CNCINet, "cnci-net", "192.168.128.0", "Host start address for CNCI mgmt network - must be at least /18")
	setupCmd.Flags().StringVar(&clusterConf.ServerIP, "server-ip", hostIP, "IP address nodes can reach this host on")
	setupCmd.Flags().StringVar(&clusterConf.ServerHostname, "server-hostname", deploy.HostnameWithFallback(), "Name or FQDN that this host can be reached on")
	setupCmd.Flags().StringVar(&imageCacheDirectory, "image-cache-directory", deploy.DefaultImageCacheDir(), "Directory to use for caching of downloaded images")
	setupCmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files which might break the cluster")
	setupCmd.Flags().BoolVar(&localLauncher, "local-launcher", false, "Enable a local launcher on this node (for testing)")
	setupCmd.Flags().BoolVar(&clusterConf.DisableLimits, "disable-limits", false, "Disable memory limit checking for cluster nodes")
	setupCmd.Flags().Var(&cnciSize, "cnci", "Specifies the resources (mem, cpu) available to CNCIs.  Can be 'tiny', 'medium', 'large'")
}

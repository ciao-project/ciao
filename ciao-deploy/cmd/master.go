// Copyright © 2017 Intel Corporation

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
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/01org/ciao/ciao-deploy/deploy"
	"github.com/spf13/cobra"
)

var clusterConf = &deploy.ClusterConfiguration{}
var force bool
var localLauncher bool

// masterCmd represents the master command
var masterCmd = &cobra.Command{
	Use:   "master",
	Short: "Sets up this machine as the master node",
	Long:  "Configures this machine as the master node for a ciao cluster",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancelFunc := context.WithCancel(context.Background())
		defer cancelFunc()

		sigCh := make(chan os.Signal, 1)
		go func() {
			<-sigCh
			cancelFunc()
		}()
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		err := deploy.SetupMaster(ctx, force, imageCacheDirectory, clusterConf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error provisioning system as master: %v\n", err)
			os.Exit(1)
		}

		if localLauncher {
			err = deploy.SetupLocalLauncher(ctx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error setting up local launcher: %v\n", err)
			}
		}
		os.Exit(0)
	},
}

func init() {
	RootCmd.AddCommand(masterCmd)

	// For configuration file generation
	masterCmd.Flags().StringVar(&clusterConf.CephID, "ceph-id", "ciao", "The ceph id for the storage cluster")
	masterCmd.Flags().StringVar(&clusterConf.HTTPSCaCertPath, "https-ca-cert", "", "Path to CA certificate for HTTP service")
	masterCmd.Flags().StringVar(&clusterConf.HTTPSCertPath, "https-cert", "", "Path to certificate for HTTPS service")
	masterCmd.Flags().StringVar(&clusterConf.KeystoneServiceUser, "keystone-service-user", "", "Username for controller to access keystone (service user)")
	masterCmd.Flags().StringVar(&clusterConf.KeystoneServicePassword, "keystone-service-password", "", "Password for controller to access keystone (service user)")
	masterCmd.Flags().StringVar(&clusterConf.KeystoneURL, "keystone-url", "", "URL for keystone server")
	masterCmd.Flags().StringVar(&clusterConf.AdminSSHKeyFile, "admin-ssh-key", "", "Path to SSH public key for accessing CNCI")
	masterCmd.Flags().StringVar(&clusterConf.AdminSSHPassword, "admin-password", "", "Password for accessing CNCI")
	masterCmd.Flags().StringVar(&clusterConf.ComputeNet, "compute-net", "", "Network range for compute network")
	masterCmd.Flags().StringVar(&clusterConf.MgmtNet, "mgmt-net", "", "Network range for management network")
	masterCmd.Flags().StringVar(&clusterConf.ServerIP, "server-ip", "", "IP address nodes can reach this host on")

	masterCmd.Flags().StringVar(&imageCacheDirectory, "image-cache-directory", deploy.DefaultImageCacheDir(), "Directory to use for caching of downloaded images")
	masterCmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files which might break the cluster")
	masterCmd.Flags().BoolVar(&localLauncher, "local-launcher", false, "Enable a local launcher on this node (for testing)")
}

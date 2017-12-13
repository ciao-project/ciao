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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func addIPList(name string, IPs []string) error {
	for _, addr := range IPs {
		// verify it's a good address
		IP := net.ParseIP(addr)
		if IP == nil {
			return fmt.Errorf("Invalid IP addresss: %s", IP.String())
		}
	}
	err := c.AddExternalIPAddresses(name, IPs)
	if err != nil {
		return errors.Wrap(err, "Error adding external IPs")
	}

	return nil
}

func addIPSubnet(name, address string) error {
	// verify it's a good subnet address, if not try parsing as regular IP
	_, network, err := net.ParseCIDR(address)
	if err == nil {
		if ones, bits := network.Mask.Size(); bits-ones < 2 {
			return errors.New("Use address mode to add a single IP address")
		}

		err = c.AddExternalIPSubnet(name, network)
		if err != nil {
			return errors.Wrap(err, "Error adding external IP subnet")
		}
	} else {
		ips := []string{address}
		return addIPList(name, ips)
	}

	return nil
}

var addExternalIPCmd = &cobra.Command{
	Use:   "external-ip POOL SUBNET or IP",
	Short: "Add IP to external IP pool",
	Long:  `Add an external IP address to a pool. This command takes either a subnet in CIDR format of a list of IPs.`,
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 2 {
			return addIPList(args[0], args[1:])
		}

		return addIPSubnet(args[0], args[1])
	},
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add objects to objects in the cluster",
}

func init() {
	addCmd.AddCommand(addExternalIPCmd)
	rootCmd.AddCommand(addCmd)
}

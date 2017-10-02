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

	"github.com/spf13/cobra"
)

var ip string
var subnet string

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete various ciao objects",
	Long: `Delete the objects below from the ciao cluster`,
}

var eventDelCmd = &cobra.Command{
	Use:   "event",
	Long: `Delete all events.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Deleting all events...")
	},
}

var externalipDelCmd = &cobra.Command{
	Use:   "external-ip [POOL NAME]",
	Long: `Delete unmapped external IPs from a pool.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("delete unmapped external ips from: " + args[0])
	},
}

var imageDelCmd = &cobra.Command{
	Use:   "image [UUID]",
	Long: `Delete an image.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("image delete called")
	},
}

var instanceDelCmd = &cobra.Command{
	Use:   "instance [UUID]",
	Long: `Delete a specific instance to the ciao cluster.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("instance delete called")
	},
}

var poolDelCmd = &cobra.Command{
	Use:   "pool [NAME]",
	Long: `Delete an external IP pool.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("pool delete called")
	},
}

var volumeDelCmd = &cobra.Command{
	Use:   "volume [UUID]",
	Long: `Delete a volume.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("volume delete called")
	},
}

var workloadDelCmd = &cobra.Command{
	Use:   "workload [UUID]",
	Long: `Delete a workload.`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("workload delete called")
	},
}

var delCmds = []*cobra.Command{eventDelCmd, externalipDelCmd, imageDelCmd, instanceDelCmd, poolDelCmd, volumeDelCmd, workloadDelCmd}

func init() {
	for _, cmd := range delCmds {
		deleteCmd.AddCommand(cmd)
	}
	RootCmd.AddCommand(deleteCmd)

	externalipDelCmd.Flags().StringVar(&ip, "ip", "", "IPv4 Address")
	externalipDelCmd.Flags(). StringVar(&subnet, "subnet", "", "Subnet in CIDR format")
}

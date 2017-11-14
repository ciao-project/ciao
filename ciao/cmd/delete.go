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
	"os"

	"github.com/spf13/cobra"
)

var ip string
var subnet string

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete various ciao objects",
	Long:  `Delete the objects below from the ciao cluster`,
}

var eventDelCmd = &cobra.Command{
	Use:  "event",
	Long: `Delete all events.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := C.DeleteEvents()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting events: %s\n", err)
			return
		}
		fmt.Printf("Deleted all event logs\n")
	},
}

var imageDelCmd = &cobra.Command{
	Use:  "image [UUID]",
	Long: `Delete an image.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := C.DeleteImage(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting image: %s\n", err)
			return
		}

		fmt.Printf("Deleted image %s\n", args[0])
	},
}

var instanceDelCmd = &cobra.Command{
	Use:  "instance [UUID]",
	Long: `Delete a specific instance to the ciao cluster.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if CommandFlags.All {
			err := C.DeleteAllInstances()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error deleting all instances: %s\n", err)
				return
			}
			fmt.Printf("Deleted all instances\n")
		}
		instance := args[0]
		if instance == "" {
			fmt.Println("Missing required instance UUID parameter")
			return
		}

		err := C.DeleteInstance(instance)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting instance: %s\n", err)
			return
		}

		fmt.Printf("Deleted instance: %s\n", instance)
	},
}

var poolDelCmd = &cobra.Command{
	Use:  "pool [NAME]",
	Long: `Delete an external IP pool.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		if  name == "" {
			fmt.Println("Missing required pool NAME parameter")
			return
		}

		err := C.DeleteExternalIPPool(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting external IP pool:%s\n", err)
			return
		}

		fmt.Printf("Deleted pool: %s\n", name)
	},
}

var volumeDelCmd = &cobra.Command{
	Use:  "volume [UUID]",
	Long: `Delete a volume.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		volume := args[0]
		if volume == "" {
			fmt.Println("Error missing required volume UUID parameter")
			return
		}

		err := C.DeleteVolume(volume)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting volume: %s\n", err)
		}
		fmt.Printf("Deleted volume %s\n", volume)
	},
}

var workloadDelCmd = &cobra.Command{
	Use:  "workload [UUID]",
	Long: `Delete a workload.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		workload := args[0]
		if workload == "" {
			fmt.Println("Missing required workload UUID paramter")
			return
		}

		err := C.DeleteWorkload(workload)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting workload: %s\n", err)
			return
		}
		fmt.Printf("Deleted workload %s\n", workload)
	},
}

var delCmds = []*cobra.Command{eventDelCmd, imageDelCmd, instanceDelCmd, poolDelCmd, volumeDelCmd, workloadDelCmd}

func init() {
	for _, cmd := range delCmds {
		deleteCmd.AddCommand(cmd)
	}
	RootCmd.AddCommand(deleteCmd)
}

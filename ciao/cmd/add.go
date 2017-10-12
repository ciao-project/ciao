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

var config string

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add various ciao objects",
	Long:  `Add the objects below to the ciao cluster`,
	Args:  cobra.MinimumNArgs(2),
}

var imageAddCmd = &cobra.Command{
	Use:  "image [UUID]",
	Long: `Add a specific image to the ciao cluster.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("image called")
	},
}

var instanceAddCmd = &cobra.Command{
	Use:  "instance [UUID]",
	Long: `Add a specific instance to the ciao cluster.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("instance called")
	},
}

var poolAddCmd = &cobra.Command{
	Use:  "pool",
	Long: `Add a pool to the cluster.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("pool called")
	},
}

var volumeAddCmd = &cobra.Command{
	Use:  "volume",
	Long: `Add a volume to a given instance.`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("volume called")
	},
}

var workloadAddCmd = &cobra.Command{
	Use:  "workload",
	Long: `Add a new workload.`,
	Run: func(cmd *cobra.Command, args []string) {
		if config == "" {
			fmt.Println("Please supply a config for the workload.")
		} else {
			fmt.Println("add workload using config: " + config)
		}
	},
}

var addCmds = []*cobra.Command{imageAddCmd, instanceAddCmd, poolAddCmd, volumeAddCmd, workloadAddCmd}

func init() {
	for _, cmd := range addCmds {
		addCmd.AddCommand(cmd)
	}
	RootCmd.AddCommand(addCmd)

	workloadAddCmd.Flags().StringVarP(&config, "config", "c", "", "Filename for yaml file describing the workload")
}

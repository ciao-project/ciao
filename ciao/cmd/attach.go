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

var attachCmd = &cobra.Command{
	Use:   "attach",
	Short: "Attach various ciao objects",
	Long: `Attach the objects below to the ciao cluster.`,
	Args: cobra.MinimumNArgs(2),
}

var attachIpCmd = &cobra.Command{
	Use:   "external-ip [POOL] [INSTANCE UUID]",
	Long: `Attach an external IP from a given pool to an instance.`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Attaching ip from " + args[0] + " to " + args[1])
	},
}

var attachVolCmd = &cobra.Command{
	Use:   "volume [UUID] [INSTANCE UUID]",
	Long: `Attach a volume to an instance.`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Attaching Volume " + args[0] + " to " + args[1])
	},
}

func init() {
	attachCmd.AddCommand(attachIpCmd)
	attachCmd.AddCommand(attachVolCmd)

	RootCmd.AddCommand(attachCmd)

	attachVolCmd.Flags().StringP("mode", "m", "", "Access mode (default \"rw\")")
	attachVolCmd.Flags().StringP("mountpoint", "p", "", "Mount point (default \"/mnt\")")
}

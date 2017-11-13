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

var detachCmd = &cobra.Command{
	Use:   "detach",
	Short: "Detach various ciao objects",
	Long:  `Detach the objects below from their given connections`,
	Args:  cobra.MinimumNArgs(2),
}

var detachIpCmd = &cobra.Command{
	Use:  "external-ip [IP]",
	Long: `Detach an external IP`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Detaching IP: " + args[0])
	},
}

var detachVolCmd = &cobra.Command{
	Use:  "volume [UUID]",
	Long: `Detach a volume from an instance`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Detaching Volume " + args[0])
	},
}

func init() {
	detachCmd.AddCommand(detachIpCmd)
	detachCmd.AddCommand(detachVolCmd)

	RootCmd.AddCommand(detachCmd)
}

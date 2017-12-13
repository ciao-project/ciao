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

	"github.com/ciao-project/ciao/ciao-controller/types"

	"github.com/spf13/cobra"
)

func restoreNode(args []string) int {
	err := c.ChangeNodeStatus(args[0], types.NodeStatusReady)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to restore node: %s\n", err)
		return 1
	}
	return 0
}

var restoreCmd = &cobra.Command{
	Use:   "restore [NODE]",
	Short: "Restore a node",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(restoreNode(args))
	},
}

func init() {
	rootCmd.AddCommand(restoreCmd)
}

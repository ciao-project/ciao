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
	"fmt"
	"os"

	"github.com/ciao-project/ciao/ciao-deploy/deploy"
	"github.com/spf13/cobra"
)

func update() int {
	ctx, cancelFunc := getSignalContext()
	defer cancelFunc()

	err := deploy.UpdateMaster(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error updating master node")
		return 1
	}

	if localLauncher {
		err = deploy.SetupLocalLauncher(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error setting up local launcher")
			return 1
		}
	}

	return 0
}

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the master node on the cluster",
	Long:  `Use on an already setup master node to update the current software on the node`,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(update())
	},
}

func init() {
	RootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVar(&localLauncher, "local-launcher", false, "Enable a local launcher on this node (for testing)")

}

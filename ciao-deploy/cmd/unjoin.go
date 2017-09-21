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
	"os/user"

	"github.com/ciao-project/ciao/ciao-deploy/deploy"
	"github.com/spf13/cobra"
)

func unjoin(args []string) int {
	ctx, cancelFunc := getSignalContext()
	defer cancelFunc()

	hosts := args
	err := deploy.TeardownNodes(ctx, sshUser, hosts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error unprovisioning nodes: %v\n", err)
		return 1
	}
	return 0
}

// unjoinCmd represents the unjoin command
var unjoinCmd = &cobra.Command{
	Use:   "unjoin <hosts>",
	Short: "Remove the specified nodes from the cluster",
	Long: `Remove the nodes from the cluster. Removing certificates and
	 uninstalling software.`,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(unjoin(args))
	},
	Args: cobra.MinimumNArgs(1),
}

func init() {
	RootCmd.AddCommand(unjoinCmd)

	u, err := user.Current()
	currentUser := ""
	if err == nil {
		currentUser = u.Username
	}

	unjoinCmd.Flags().StringVar(&sshUser, "user", currentUser, "User to SSH as")
}

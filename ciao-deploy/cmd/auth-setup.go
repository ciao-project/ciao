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

	"github.com/ciao-project/ciao/ciao-deploy/deploy"
	"github.com/spf13/cobra"
)

func setupAuth(force bool) int {
	ctx, cancelFunc := getSignalContext()
	defer cancelFunc()

	caCertPath, certPath, err := deploy.CreateAdminCert(ctx, force)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating admin certificate: %v\n", err)
		return 1
	}
	fmt.Printf("Authentication certificates created: CA certificate: %s admin certificate: %s\n",
		caCertPath, certPath)
	return 0
}

// authSetupCmd represents the init command
var authSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Create initial authentication certificates",

	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(setupAuth(force))
	},
}

func init() {
	authCmd.AddCommand(authSetupCmd)
	authSetupCmd.Flags().BoolVar(&force, "force", false, "Overwrite existing files which might break the cluster")
}

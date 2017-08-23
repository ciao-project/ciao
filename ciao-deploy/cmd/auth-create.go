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

	"github.com/01org/ciao/ciao-deploy/deploy"
	"github.com/01org/ciao/ssntp/uuid"

	"github.com/spf13/cobra"
)

func createAuth(args []string) int {
	ctx, cancelFunc := getSignalContext()
	defer cancelFunc()

	username := args[0]
	tenants := args[1:]

	if len(tenants) == 0 {
		newTenantID := uuid.Generate()
		tenants = append(tenants, newTenantID.String())
		fmt.Printf("No tenant specified: creating new tenant: %s\n", tenants[0])
	}

	certPath, err := deploy.CreateUserCert(ctx, username, tenants)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating user certificate: %v\n", err)
		return 1
	}
	fmt.Printf("User authentication certificate created: %s\n", certPath)
	return 0
}

// authCreateCmd represents the create command
var authCreateCmd = &cobra.Command{
	Use:   "create <username> [<tenant>, <tenant>...]",
	Short: "Create a new user on the cluster",
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(createAuth(args))
	},
	Args: cobra.MinimumNArgs(1),
}

func init() {
	authCmd.AddCommand(authCreateCmd)
}

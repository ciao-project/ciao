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
	"github.com/spf13/cobra"
)

var allWorkloads bool

// createBatWorkloadsCmd represents the create-bat-workloads command
var createBatWorkloadsCmd = &cobra.Command{
	Use:   "create-bat-workloads",
	Short: "Create workloads necessary for BAT",
	Long: `Downloads the images and creates the workloads necessary for running
	 the Basic Acceptance Tests (BAT)`,
	Run: func(cmd *cobra.Command, args []string) {
		err := deploy.CreateBatWorkloads(allWorkloads)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating BAT workloads: %s\n", err)
		}
	},
}

func init() {
	RootCmd.AddCommand(createBatWorkloadsCmd)

	createBatWorkloadsCmd.Flags().BoolVar(&allWorkloads, "all-workloads", false, "Create extra workloads not required for BAT")
}

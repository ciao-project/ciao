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
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/01org/ciao/ciao-deploy/deploy"
	"github.com/spf13/cobra"
)

var allWorkloads bool
var sshPublicKeyPath string
var password string

// createBatWorkloadsCmd represents the create-bat-workloads command
var createBatWorkloadsCmd = &cobra.Command{
	Use:   "create-bat-workloads",
	Short: "Create workloads necessary for BAT",
	Long: `Downloads the images and creates the workloads necessary for running
	 the Basic Acceptance Tests (BAT)`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancelFunc := context.WithCancel(context.Background())
		defer cancelFunc()

		sigCh := make(chan os.Signal, 1)
		go func() {
			<-sigCh
			cancelFunc()
		}()
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		var sshPublicKey []byte
		var err error
		if sshPublicKeyPath != "" {
			sshPublicKey, err = ioutil.ReadFile(sshPublicKeyPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading from SSH key file: %s\n", err)
				os.Exit(1)
			}
		}

		err = deploy.CreateBatWorkloads(ctx, allWorkloads, string(sshPublicKey), password)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating BAT workloads: %s\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	},
}

func init() {
	RootCmd.AddCommand(createBatWorkloadsCmd)

	createBatWorkloadsCmd.Flags().BoolVar(&allWorkloads, "all-workloads", false, "Create extra workloads not required for BAT")
	createBatWorkloadsCmd.Flags().StringVar(&sshPublicKeyPath, "ssh-public-key-file", "", "SSH public key to be injected into workloads (demouser)")
	createBatWorkloadsCmd.Flags().StringVar(&password, "password", "", "Password to be injected into workloads (demouser)")
}

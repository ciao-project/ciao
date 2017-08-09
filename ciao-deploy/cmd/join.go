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
	"os"
	"os/signal"
	"os/user"
	"syscall"

	"github.com/01org/ciao/ciao-deploy/deploy"
	"github.com/spf13/cobra"
)

var networkNode bool
var sshUser string

func join(args []string) int {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	sigCh := make(chan os.Signal, 1)
	go func() {
		<-sigCh
		cancelFunc()
	}()
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	hosts := args
	err := deploy.SetupNodes(ctx, sshUser, networkNode, hosts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error provisioning nodes: %v\n", err)
		return 1
	}
	return 0
}

var joinCmd = &cobra.Command{
	Use:   "join <hostname>",
	Short: "Join a node to the Ciao cluster",
	Long:  `Join a node of the desired role to the cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(join(args))
	},
	Args: cobra.MinimumNArgs(1),
}

func init() {
	RootCmd.AddCommand(joinCmd)

	u, err := user.Current()
	currentUser := ""
	if err == nil {
		currentUser = u.Username
	}

	joinCmd.Flags().BoolVar(&networkNode, "network", false, "Designate as network node")
	joinCmd.Flags().StringVar(&sshUser, "user", currentUser, "User to SSH as")
}

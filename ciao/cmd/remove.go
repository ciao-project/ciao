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
	"net"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:  "remove",
	Long: `Remove objects from other objects in the cluster.`,
}

var removeExternalIPCmd = &cobra.Command{
	Use:  "external-ip POOL SUBNET or IP",
	Long: `Remove IP address or subnet from the pool`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		_, network, err := net.ParseCIDR(args[1])
		if err == nil {
			return errors.Wrap(c.RemoveExternalIPSubnet(args[1], network),
				"Error removing external IP subnet")
		}

		return errors.Wrap(c.RemoveExternalIPAddress(name, args[1]),
			"Error removing external IP address")
	},
}

func init() {
	removeCmd.AddCommand(removeExternalIPCmd)

	rootCmd.AddCommand(removeCmd)
}

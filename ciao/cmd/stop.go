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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var stopInstanceCmd = &cobra.Command{
	Use:  "instance ID",
	Long: `Restart an instance.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.Wrap(c.StopInstance(args[0]), "Error stopping instance")
	},
}

var stopCmd = &cobra.Command{
	Use:  "stop",
	Long: "Stop an object in the cluster",
}

func init() {
	stopCmd.AddCommand(stopInstanceCmd)
	rootCmd.AddCommand(stopCmd)
}

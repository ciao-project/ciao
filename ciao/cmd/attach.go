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

var volAttachFlags = struct {
	mode       string
	mountpoint string
}{}

var attachCmd = &cobra.Command{
	Use:   "attach",
	Short: "Attach objects to other objects in the cluster.",
}

var attachIPCmd = &cobra.Command{
	Use:   "external-ip POOL INSTANCE",
	Short: "Attach external IP to instance",
	Long:  `Attach an external IP from a given pool to an instance.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.Wrap(c.MapExternalIP(args[0], args[1]), "Error mapping external IP")
	},
}

var attachVolCmd = &cobra.Command{
	Use:   "volume VOLUME INSTANCE",
	Short: `Attach a volume to an instance`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.Wrap(c.AttachVolume(args[0], args[1], volAttachFlags.mountpoint, volAttachFlags.mode),
			"Error attaching volume")
	},
}

func init() {
	attachCmd.AddCommand(attachIPCmd)
	attachCmd.AddCommand(attachVolCmd)

	rootCmd.AddCommand(attachCmd)

	attachVolCmd.Flags().StringVar(&volAttachFlags.mode, "mode", "rw", "Access mode")
	attachVolCmd.Flags().StringVar(&volAttachFlags.mountpoint, "mountpoint", "/mnt", "Mount point ")
}

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

var deleteCmd = &cobra.Command{
	Use:  "delete",
	Long: `Delete the objects from the cluster`,
}

var eventsDelCmd = &cobra.Command{
	Use:  "events",
	Long: `Delete all events.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.Wrap(c.DeleteEvents(), "Error deleting events")
	},
}

var imageDelCmd = &cobra.Command{
	Use:  "image ID",
	Long: `Delete an image.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.Wrap(c.DeleteImage(args[0]), "Error deleting image")
	},
}

var deleteInstanceFlags = struct {
	all bool
}{}

var instanceDelCmd = &cobra.Command{
	Use:  "instance ID",
	Long: `Delete instance from cluster.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if deleteInstanceFlags.all {
			return errors.Wrap(c.DeleteAllInstances(), "Error deleting all instances")
		}

		if len(args) < 1 {
			return errors.New("Instance ID required")
		}

		return errors.Wrap(c.DeleteInstance(args[0]), "Error deleting instance")
	},
}

var poolDelCmd = &cobra.Command{
	Use:  "pool NAME",
	Long: `Delete an external IP pool.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.Wrap(c.DeleteExternalIPPool(args[0]), "Error deleting external IP pool")
	},
}

var volumeDelCmd = &cobra.Command{
	Use:  "volume ID",
	Long: `Delete a volume.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.Wrap(c.DeleteVolume(args[0]), "Error deleting volume")
	},
}

var tenantDelCmd = &cobra.Command{
	Use:  "tenant ID",
	Long: `Delete a tenant`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.Wrap(c.DeleteTenant(args[0]), "Error deleting tenant")
	},
}

var workloadDelCmd = &cobra.Command{
	Use:  "workload ID",
	Long: `Delete a workload.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.Wrap(c.DeleteWorkload(args[0]), "Error deleting workload")
	},
}

var delCmds = []*cobra.Command{eventsDelCmd, imageDelCmd, instanceDelCmd, poolDelCmd, volumeDelCmd, workloadDelCmd, tenantDelCmd}

func init() {
	for _, cmd := range delCmds {
		deleteCmd.AddCommand(cmd)
	}

	instanceDelCmd.Flags().BoolVar(&deleteInstanceFlags.all, "all", false, "Delete all instances")

	rootCmd.AddCommand(deleteCmd)
}

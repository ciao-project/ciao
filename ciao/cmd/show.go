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
	"strings"

	"github.com/ciao-project/ciao/ciao-sdk"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show information about various ciao objects",
	Long:  `Show outputs a list and/or details for available commands`,
}

var eventShowCmd = &cobra.Command{
	Use:  "event [TENANT ID]",
	Long: `When called with no args, it will print all events.`,
	Run: func(cmd *cobra.Command, args []string) {
			command := strings.Fields(cmd.Use)
			sdk.Show(command[0], args)
	},
}

var externalipShowCmd = &cobra.Command{
	Use:  "externalip",
	Long: `When called with no args, it will print all externalips.`,
	Run: func(cmd *cobra.Command, args []string) {
			command := strings.Fields(cmd.Use)
			sdk.Show(command[0], args)
	},
}

var imageShowCmd = &cobra.Command{
	Use:  "image <UUID>",
	Long: `When called with no args, it will print all images.`,
	Run: func(cmd *cobra.Command, args []string) {
			command := strings.Fields(cmd.Use)
			sdk.Show(command[0], args)
	},
}

var instanceShowCmd = &cobra.Command{
	Use:  "instance <UUID>",
	Long: `When called with no args, it will print all instances.`,
	Run: func(cmd *cobra.Command, args []string) {
			command := strings.Fields(cmd.Use)
			sdk.Show(command[0], args)
	},
}

var nodeShowCmd = &cobra.Command{
	Use:  "node [NODE-ID]",
	Long: `Show information about a node.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
			command := strings.Fields(cmd.Use)
			sdk.Show(command[0], args)
	},
}

var poolShowCmd = &cobra.Command{
	Use:  "pool [NAME]",
	Long: `Show ciao external IP pool details.`,
	Run: func(cmd *cobra.Command, args []string) {
			command := strings.Fields(cmd.Use)
			sdk.Show(command[0], args)
	},
}

var quotasShowCmd = &cobra.Command{
	Use:  "quotas [TENANT STRING]",
	Long: `When called with no args, it will print all quotas for current tenant.`,
	Run: func(cmd *cobra.Command, args []string) {
			command := strings.Fields(cmd.Use)
			sdk.Show(command[0], args)
	},
}

var tenantShowCmd = &cobra.Command{
	Use:  "tenant [NAME]",
	Long: `When called with no args, it will print all tenants.`,
	Run: func(cmd *cobra.Command, args []string) {
			command := strings.Fields(cmd.Use)
			sdk.Show(command[0], args)
	},
}

var traceShowCmd = &cobra.Command{
	Use:  "trace [LABEL]",
	Long: `When called with no args, it will print all traces.`,
	Run: func(cmd *cobra.Command, args []string) {
			command := strings.Fields(cmd.Use)
			sdk.Show(command[0], args)
	},
}

var volumeShowCmd = &cobra.Command{
	Use:  "volume [UUID]",
	Long: `When called with no args, it will print all volumes.`,
	Run: func(cmd *cobra.Command, args []string) {
			command := strings.Fields(cmd.Use)
			sdk.Show(command[0], args)
	},
}

var workloadShowCmd = &cobra.Command{
	Use:  "workload [UUID]",
	Long: `When called with no args, it will print all workloads.`,
	Run: func(cmd *cobra.Command, args []string) {
			command := strings.Fields(cmd.Use)
			sdk.Show(command[0], args)
	},
}

var showcmds = []*cobra.Command{eventShowCmd, externalipShowCmd, imageShowCmd, instanceShowCmd, nodeShowCmd, poolShowCmd, quotasShowCmd, tenantShowCmd, traceShowCmd, volumeShowCmd, workloadShowCmd}

func init() {
	for _, cmd := range showcmds {
		showCmd.AddCommand(cmd)
	}

	RootCmd.AddCommand(showCmd)

	showCmd.PersistentFlags().StringVarP(&sdk.Template, "template", "t", "", "Template used to format output")

	eventShowCmd.Flags().BoolVar(&sdk.CommandFlags.All, "all", false, "List events for all tenants in a cluster")

	instanceShowCmd.Flags().StringVar(&sdk.CommandFlags.Computenode, "computenode", "", "Compute node to list instances from (defalut to all  nodes when empty)")
	instanceShowCmd.Flags().BoolVar(&sdk.CommandFlags.Detail, "verbose", false, "Print detailed information about each instance")
	instanceShowCmd.Flags().IntVar(&sdk.CommandFlags.Limit, "limit", 1, "Limit listing to <limit> results")
	instanceShowCmd.Flags().StringVar(&sdk.CommandFlags.Marker, "marker", "", "Show instance list starting from the next instance after marker")
	instanceShowCmd.Flags().IntVar(&sdk.CommandFlags.Offset, "offset", 0, "Show instance list starting from instance <offset>")
	instanceShowCmd.Flags().StringVar(&sdk.CommandFlags.TenantID, "tenant", "", "Specify to list instances from a tenant other than -tenant-id")
	instanceShowCmd.Flags().StringVar(&sdk.CommandFlags.Workload, "workload", "", "Workload UUID")
}

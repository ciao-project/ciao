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

	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show detailed information about a ciao object",
}

var cnciShowCmd = &cobra.Command{
	Use:  "cnci ID",
	Long: "Show information about a CNCI.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !c.IsPrivileged() {
			return errors.New("CNCI information is restricted to privileged users")
		}

		cncis, err := c.ListCNCIs()
		if err != nil {
			return errors.Wrap(err, "Error listing CNCIs")
		}

		var cnci *types.CiaoCNCI
		for i := range cncis.CNCIs {
			if cncis.CNCIs[i].ID == args[0] {
				cnci = &cncis.CNCIs[0]
				break
			}
		}

		if cnci == nil {
			return fmt.Errorf("CNCI %s not found", args[0])
		}

		return render(cmd, cnci)
	},
}

var imageShowCmd = &cobra.Command{
	Use:  "image ID",
	Long: "Show information about an image.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		image, err := c.GetImage(args[0])
		if err != nil {
			return errors.Wrap(err, "Error getting image")
		}

		return render(cmd, image)
	},
}

var instanceShowCmd = &cobra.Command{
	Use:  "instance ID",
	Long: "Show information about an instance",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		server, err := c.GetInstance(args[0])
		if err != nil {
			return errors.Wrap(err, "Error getting instance")
		}

		return render(cmd, server.Server)
	},
}

var nodeShowCmd = &cobra.Command{
	Use:  "node ID",
	Long: "Show information about a node.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !c.IsPrivileged() {
			return errors.New("Node information is restricted to privileged users")
		}

		nodes, err := c.ListNodes()
		if err != nil {
			return errors.Wrap(err, "Error listing node")
		}

		var node *types.CiaoNode
		for i := range nodes.Nodes {
			if nodes.Nodes[i].ID == args[0] {
				node = &nodes.Nodes[i]
				break
			}
		}

		if node == nil {
			return fmt.Errorf("Node %s not found", args[0])
		}

		return render(cmd, node)
	},
}

var tenantShowCmd = &cobra.Command{
	Use:  "tenant ID",
	Long: "Show tenant configuration.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !c.IsPrivileged() {
			return errors.New("Tenant configuration is restricted to privileged users")
		}

		tenant, err := c.GetTenantConfig(args[0])
		if err != nil {
			return errors.Wrap(err, "Error getting tenant config")
		}

		return render(cmd, tenant)
	},
}

var traceShowCmd = &cobra.Command{
	Use:  "trace LABEL",
	Long: "Show trace data.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := c.GetTraceData(args[0])
		if err != nil {
			return errors.Wrap(err, "Error gettting trace data")
		}

		return render(cmd, data.Summary)
	},
}

var volumeShowCmd = &cobra.Command{
	Use:  "volume ID",
	Long: "Show volume information.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		volume, err := c.GetVolume(args[0])
		if err != nil {
			return errors.Wrap(err, "Error getting volume")
		}

		return render(cmd, volume)
	},
}

var workloadShowCmd = &cobra.Command{
	Use:  "workload ID",
	Long: "Show workload information.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workload, err := c.GetWorkload(args[0])
		if err != nil {
			return errors.Wrap(err, "Error getting workload")
		}

		return render(cmd, workload)
	},
}

var showCmds = []*cobra.Command{
	cnciShowCmd,
	imageShowCmd,
	instanceShowCmd,
	nodeShowCmd,
	tenantShowCmd,
	traceShowCmd,
	volumeShowCmd,
	workloadShowCmd,
}

func init() {
	for _, cmd := range showCmds {
		showCmd.AddCommand(cmd)
		cmd.Flags().StringVarP(&template, "template", "f", "", "Template used to format output")
	}

	rootCmd.AddCommand(showCmd)
}

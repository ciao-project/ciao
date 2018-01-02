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

	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/intel/tfortools"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show detailed information about an object",
}

var cnciShowCmd = &cobra.Command{
	Use:   "cnci ID",
	Short: "Show information about a CNCI",
	Args:  cobra.ExactArgs(1),
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
	Annotations: map[string]string{
		"template_usage": tfortools.GenerateUsageUndecorated(types.CiaoCNCI{}),
	},
}

var imageShowCmd = &cobra.Command{
	Use:   "image ID",
	Short: "Show information about an image",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		image, err := c.GetImage(args[0])
		if err != nil {
			return errors.Wrap(err, "Error getting image")
		}

		return render(cmd, image)
	},
	Annotations: map[string]string{
		"template_usage": tfortools.GenerateUsageUndecorated(types.Image{}),
	},
}

var instanceShowCmd = &cobra.Command{
	Use:   "instance ID",
	Short: "Show information about an instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		server, err := c.GetInstance(args[0])
		if err != nil {
			return errors.Wrap(err, "Error getting instance")
		}

		return render(cmd, server.Server)
	},
	Annotations: map[string]string{
		"template_usage": tfortools.GenerateUsageUndecorated(api.ServerDetails{}),
	},
}

var nodeShowCmd = &cobra.Command{
	Use:   "node ID",
	Short: "Show information about a node",
	Args:  cobra.ExactArgs(1),
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
	Annotations: map[string]string{
		"template_usage": tfortools.GenerateUsageUndecorated(types.CiaoNode{}),
	},
}

var tenantShowCmd = &cobra.Command{
	Use:   "tenant ID",
	Short: "Show tenant configuration",
	Args:  cobra.ExactArgs(1),
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
	Annotations: map[string]string{
		"default_template": `{{ htable (cols (sliceof .) "ID" "Name") }}`,
		"template_usage":   tfortools.GenerateUsageUndecorated(types.TenantConfig{}),
	},
}

var traceShowCmd = &cobra.Command{
	Use:   "trace LABEL",
	Short: "Show trace data for a label",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := c.GetTraceData(args[0])
		if err != nil {
			return errors.Wrap(err, "Error gettting trace data")
		}

		return render(cmd, data.Summary)
	},
	Annotations: map[string]string{
		"template_usage": tfortools.GenerateUsageUndecorated(types.CiaoBatchFrameStat{}),
	},
}

var volumeShowTemplate = `ID:		{{ .ID }}
Name:		{{ .Name }}
Description:	{{ .Description }}
State:		{{ .State }}
Size:		{{ .Size }}
CreateTime:	{{ .CreateTime }}
`

var volumeShowCmd = &cobra.Command{
	Use:   "volume ID",
	Short: "Show volume information",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		volume, err := c.GetVolume(args[0])
		if err != nil {
			return errors.Wrap(err, "Error getting volume")
		}

		return render(cmd, volume)
	},
	Annotations: map[string]string{
		"default_template": volumeShowTemplate,
		"template_usage":   tfortools.GenerateUsageUndecorated(types.Volume{}),
	},
}

var workloadShowTemplate = `ID:			{{ .ID }}
Description: 		{{ .Description }}
{{ if eq .VMType "qemu" -}}
FWType:			{{ .FWType }}
{{ else -}}
ImageName:		{{ .ImageName }}
{{ end -}}
Visibility:		{{ .Visibility }}
Requirements:
	MemMB:		{{ .Requirements.MemMB }}
	VCPUs:		{{ .Requirements.VCPUs }}
	NodeID:		{{ .Requirements.NodeID }}
	Hostname	{{ .Requirements.Hostname }}
	NetworkNode	{{ .Requirements.NetworkNode }}
	Privileged	{{ .Requirements.Privileged }}
Storage:
{{- range .Storage }}
	ID:		{{ .ID }}
	Size:		{{ .Size }}
	Ephemeral:	{{ .Ephemeral }}
	Bootable:	{{ .Bootable }}
	SourceType:	{{ .SourceType }}
	Source:		{{ .Source }}
{{ end }}`

var workloadShowCmd = &cobra.Command{
	Use:   "workload ID",
	Short: "Show workload information",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workload, err := c.GetWorkload(args[0])
		if err != nil {
			return errors.Wrap(err, "Error getting workload")
		}

		return render(cmd, workload)
	},
	Annotations: map[string]string{
		"default_template": workloadShowTemplate,
		"template_usage":   tfortools.GenerateUsageUndecorated(types.Workload{}),
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
	}

	rootCmd.AddCommand(showCmd)
}

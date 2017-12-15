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
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List objects",
}

var cnciListCmd = &cobra.Command{
	Use:  "cncis",
	Long: `List CNCIs`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !c.IsPrivileged() {
			return errors.New("Listing CNCIs is limited to privileged users")
		}

		cncis, err := c.ListCNCIs()
		if err != nil {
			return errors.Wrap(err, "Error listing CNCIs")
		}

		return render(cmd, cncis.CNCIs)
	},
	Annotations: map[string]string{"default_template": "{{ table .}}"},
}

var eventListCmd = &cobra.Command{
	Use:  "events [TENANT]",
	Long: `List events for the provided tenant. If no tenant is specified and the user is privileged events for all tenants will be returned otherwise returns the current tenants events.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tenantID := ""
		if len(args) == 1 {
			tenantID = args[0]
		}

		if !c.IsPrivileged() {
			if tenantID == "" {
				tenantID = c.TenantID
			}
		}

		events, err := c.ListEvents(tenantID)
		if err != nil {
			return errors.Wrap(err, "Error listing events")
		}

		return render(cmd, events.Events)
	},
	Annotations: map[string]string{"default_template": "{{ table .}}"},
}

var externalipListCmd = &cobra.Command{
	Use:  "external-ips",
	Long: `List external IP addresses.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		IPs, err := c.ListExternalIPs()
		if err != nil {
			return errors.Wrap(err, "Error listing external IPs")
		}

		return render(cmd, IPs)
	},
	Annotations: map[string]string{"default_template": `{{ table (cols . "ExternalIP" "InternalIP" "InstanceID" "PoolName")}}`},
}

var imageListCmd = &cobra.Command{
	Use:  "images",
	Long: `List images.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		images, err := c.ListImages()
		if err != nil {
			return errors.Wrap(err, "Error getting list of images")
		}

		return render(cmd, images)
	},
	Annotations: map[string]string{"default_template": "{{ table .}}"},
}

var instanceListCmd = &cobra.Command{
	Use:  "instances [WORKLOAD]",
	Long: `List instances. If the optional workload ID is provided then only show instances matching that ID.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workloadID := ""
		if len(args) == 1 {
			workloadID = args[0]
		}

		servers, err := c.ListInstancesByWorkload(c.TenantID, workloadID)
		if err != nil {
			return errors.Wrap(err, "Error listing instances")
		}

		return render(cmd, servers.Servers)
	},
	Annotations: map[string]string{"default_template": `{{ table (cols . "Name" "ID" "SSHIP" "SSHPort" "Status") }}`},
}

var nodeListFlags = struct {
	computeNodesOnly bool
	networkNodesOnly bool
}{}

var nodeListCmd = &cobra.Command{
	Use:  "nodes",
	Long: `Lists nodes. Node type can be limited by flags.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !c.IsPrivileged() {
			return errors.New("Listing nodes is limited to privileged users")
		}

		var n types.CiaoNodes
		var err error
		if nodeListFlags.computeNodesOnly {
			n, err = c.ListComputeNodes()
		} else if nodeListFlags.networkNodesOnly {
			n, err = c.ListComputeNodes()
		} else {
			n, err = c.ListNodes()
		}

		if err != nil {
			return errors.Wrap(err, "Error getting nodes")
		}

		return render(cmd, n.Nodes)
	},
	Annotations: map[string]string{"default_template": `{{ table (cols . "ID" "Hostname" "Status")}}`},
}

var poolListCmd = &cobra.Command{
	Use:  "pools",
	Long: `List external IP pools.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := c.ListExternalIPPools()
		if err != nil {
			return errors.Wrap(err, "Error getting external IP pools")
		}

		return render(cmd, p.Pools)
	},
	Annotations: map[string]string{"default_template": `{{ table (cols . "Name" "Free" "TotalIPs")}}`},
}

var quotasListCmd = &cobra.Command{
	Use:  "quotas [TENANT ID]",
	Long: `List quotas for the current tenant or the supplied tenant if the current user is privileged.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tenantID := ""
		if len(args) == 1 {
			if !c.IsPrivileged() {
				return errors.New("Listing quotas for other tenants is for privileged users only")
			}
			tenantID = args[0]
		}

		quotas, err := c.ListQuotas(tenantID)
		if err != nil {
			return errors.Wrap(err, "Error getting quotas")
		}

		return render(cmd, quotas)
	},
	Annotations: map[string]string{"default_template": "{{ table .}}"},
}

var tenantListCmd = &cobra.Command{
	Use:  "tenants",
	Long: `List tenants available to the user or if privileged those on the cluster.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var tenants []types.TenantSummary
		if c.IsPrivileged() {
			t, err := c.ListTenants()
			if err != nil {
				return errors.Wrap(err, "Error listing tenants")
			}
			tenants = t.Tenants
		} else {
			for _, t := range c.Tenants {
				tenants = append(tenants, types.TenantSummary{
					ID: t,
				})
			}
		}

		return render(cmd, tenants)
	},
	Annotations: map[string]string{"default_template": `{{ table (cols . "ID" "Name")}}`},
}

var traceListCmd = &cobra.Command{
	Use:  "traces",
	Long: `List trace labels.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		t, err := c.ListTraceLabels()
		if err != nil {
			return errors.Wrap(err, "Error getting trace labels")
		}

		return render(cmd, t.Summaries)
	},
	Annotations: map[string]string{"default_template": "{{ table .}}"},
}

var volumeListCmd = &cobra.Command{
	Use:  "volumes",
	Long: `List volumes.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		volumes, err := c.ListVolumes()
		if err != nil {
			return errors.Wrap(err, "Error listing volumes")
		}

		return render(cmd, volumes)
	},
	Annotations: map[string]string{"default_template": "{{ table .}}"},
}

type workload struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	CPUs int    `json:"vcpus"`
	Mem  int    `json:"ram"`
}

var workloadListCmd = &cobra.Command{
	Use:  "workloads",
	Long: `List workloads.`,
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		wls, err := c.ListWorkloads()
		if err != nil {
			return errors.Wrap(err, "Error listing workloads")
		}

		var workloads []workload
		for _, wl := range wls {
			workloads = append(workloads, workload{
				Name: wl.Description,
				ID:   wl.ID,
				Mem:  wl.Requirements.MemMB,
				CPUs: wl.Requirements.VCPUs,
			})
		}

		return render(cmd, workloads)
	},
	Annotations: map[string]string{"default_template": "{{ table .}}"},
}

var listCmds = []*cobra.Command{
	cnciListCmd,
	eventListCmd,
	externalipListCmd,
	imageListCmd,
	instanceListCmd,
	nodeListCmd,
	poolListCmd,
	quotasListCmd,
	tenantListCmd,
	traceListCmd,
	volumeListCmd,
	workloadListCmd,
}

func init() {

	for _, cmd := range listCmds {
		listCmd.AddCommand(cmd)
	}

	nodeListCmd.Flags().BoolVar(&nodeListFlags.computeNodesOnly, "compute-nodes", false, "Only show compute nodes")
	nodeListCmd.Flags().BoolVar(&nodeListFlags.networkNodesOnly, "network-nodes", false, "Only show network nodes")

	rootCmd.AddCommand(listCmd)
}

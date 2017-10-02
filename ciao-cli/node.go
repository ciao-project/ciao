//

// Copyright (c) 2016 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"text/template"

	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/intel/tfortools"
)

var nodeCommand = &command{
	SubCommands: map[string]subCommand{
		"list":     new(nodeListCommand),
		"status":   new(nodeStatusCommand),
		"show":     new(nodeShowCommand),
		"evacuate": new(nodeEvacuateCommand),
		"restore":  new(nodeRestoreCommand),
	},
}

type nodeListCommand struct {
	Flag     flag.FlagSet
	compute  bool
	network  bool
	all      bool
	cnci     bool
	template string
}

func (cmd *nodeListCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] node list

List nodes

The list flags are:
`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
The template passed to the -f option operates on one of the following types:

--cnci

%s

--network

%s

--compute

%s

--all

%s`,
		tfortools.GenerateUsageUndecorated([]types.CiaoCNCI{}),
		tfortools.GenerateUsageUndecorated([]types.CiaoNode{}),
		tfortools.GenerateUsageUndecorated([]types.CiaoNode{}),
		tfortools.GenerateUsageUndecorated([]types.CiaoNode{}))
	fmt.Fprintln(os.Stderr, tfortools.TemplateFunctionHelp(nil))
	os.Exit(2)
}

func (cmd *nodeListCommand) parseArgs(args []string) []string {
	cmd.Flag.BoolVar(&cmd.compute, "compute", false, "List all compute nodes")
	cmd.Flag.BoolVar(&cmd.network, "network", false, "List all network nodes")
	cmd.Flag.BoolVar(&cmd.all, "all", false, "List all nodes")
	cmd.Flag.BoolVar(&cmd.cnci, "cnci", false, "List all CNCIs")
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *nodeListCommand) run(args []string) error {
	var t *template.Template
	if cmd.template != "" {
		var err error
		t, err = tfortools.CreateTemplate("node-list", cmd.template, nil)
		if err != nil {
			fatalf(err.Error())
		}
	}

	if cmd.compute {
		listComputeNodes(t)
	} else if cmd.network {
		listNetworkNodes(t)
	} else if cmd.cnci {
		listCNCINodes(t)
	} else if cmd.all {
		listNodes(t)
	} else {
		cmd.usage()
	}
	return nil
}

func dumpNode(node *types.CiaoNode) {
	fmt.Printf("\tHostname: %s\n", node.Hostname)
	fmt.Printf("\tUUID: %s\n", node.ID)
	fmt.Printf("\tStatus: %s\n", node.Status)
	fmt.Printf("\tLoad: %d\n", node.Load)
	fmt.Printf("\tAvailable/Total memory: %d/%d MB\n", node.MemAvailable, node.MemTotal)
	fmt.Printf("\tAvailable/Total disk: %d/%d MB\n", node.DiskAvailable, node.DiskTotal)
	fmt.Printf("\tTotal Instances: %d\n", node.TotalInstances)
	fmt.Printf("\t\tRunning Instances: %d\n", node.TotalRunningInstances)
	fmt.Printf("\t\tPending Instances: %d\n", node.TotalPendingInstances)
	fmt.Printf("\t\tPaused Instances: %d\n", node.TotalPausedInstances)
}

func dumpNodes(headerText string, url string, t *template.Template) {
	var nodes types.CiaoNodes

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &nodes)
	if err != nil {
		fatalf(err.Error())
	}

	if t != nil {
		if err := t.Execute(os.Stdout, &nodes.Nodes); err != nil {
			fatalf(err.Error())
		}
		return
	}

	for i, node := range nodes.Nodes {
		fmt.Printf("%s %d\n", headerText, i+1)
		dumpNode(&node)
	}
}

func listComputeNodes(t *template.Template) {
	url := buildComputeURL("nodes/compute")
	dumpNodes("Compute Node", url, t)
}

func listNetworkNodes(t *template.Template) {
	url := buildComputeURL("nodes/network")
	dumpNodes("Network Node", url, t)
}

func listNodes(t *template.Template) {
	url := buildComputeURL("nodes")
	dumpNodes("Node", url, t)
}

func listCNCINodes(t *template.Template) error {
	var cncis types.CiaoCNCIs

	url := buildComputeURL("cncis")

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &cncis)
	if err != nil {
		fatalf(err.Error())
	}

	if t != nil {
		if err := t.Execute(os.Stdout, &cncis.CNCIs); err != nil {
			fatalf(err.Error())
		}
		return nil
	}

	for i, cnci := range cncis.CNCIs {
		fmt.Printf("CNCI %d\n", i+1)
		fmt.Printf("\tCNCI UUID: %s\n", cnci.ID)
		fmt.Printf("\tTenant UUID: %s\n", cnci.TenantID)
		fmt.Printf("\tIPv4: %s\n", cnci.IPv4)
		fmt.Printf("\tSubnets:\n")
		for _, subnet := range cnci.Subnets {
			fmt.Printf("\t\t%s\n", subnet.Subnet)
		}
	}
	return nil
}

type nodeStatusCommand struct {
	Flag     flag.FlagSet
	template string
}

func (cmd *nodeStatusCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] node status

Show cluster status
`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n%s",
		tfortools.GenerateUsageDecorated("f", types.CiaoClusterStatus{}.Status, nil))
	os.Exit(2)
}

func (cmd *nodeStatusCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *nodeStatusCommand) run(args []string) error {
	var status types.CiaoClusterStatus
	url := buildComputeURL("nodes/summary")

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &status)
	if err != nil {
		fatalf(err.Error())
	}

	if cmd.template != "" {
		return tfortools.OutputToTemplate(os.Stdout, "node-status", cmd.template,
			&status.Status, nil)
	}

	fmt.Printf("Total Nodes %d\n", status.Status.TotalNodes)
	fmt.Printf("\tReady %d\n", status.Status.TotalNodesReady)
	fmt.Printf("\tFull %d\n", status.Status.TotalNodesFull)
	fmt.Printf("\tOffline %d\n", status.Status.TotalNodesOffline)
	fmt.Printf("\tMaintenance %d\n", status.Status.TotalNodesMaintenance)

	return nil
}

type nodeShowCommand struct {
	Flag     flag.FlagSet
	cnci     bool
	nodeID   string
	template string
}

func (cmd *nodeShowCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] node show

Show info about a node

The show flags are:
`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
The template passed to the -f option operates on one of the following types:

--cnci

%s`, tfortools.GenerateUsageUndecorated(types.CiaoCNCI{}))
	fmt.Fprintln(os.Stderr, tfortools.TemplateFunctionHelp(nil))
	os.Exit(2)
}

func (cmd *nodeShowCommand) parseArgs(args []string) []string {
	cmd.Flag.BoolVar(&cmd.cnci, "cnci", false, "Show info about a cnci node")
	cmd.Flag.StringVar(&cmd.nodeID, "node-id", "", "Node ID")
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *nodeShowCommand) run(args []string) error {
	if cmd.cnci {
		return showCNCINode(cmd)
	}
	return showNode(cmd)
}

func showNode(cmd *nodeShowCommand) error {
	if cmd.nodeID == "" {
		fatalf("Missing required -node-id parameter")
	}

	url := buildComputeURL("nodes")

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	var nodes types.CiaoNodes
	err = unmarshalHTTPResponse(resp, &nodes)
	if err != nil {
		fatalf(err.Error())
	}

	var node *types.CiaoNode
	for i := range nodes.Nodes {
		if nodes.Nodes[i].ID == cmd.nodeID {
			node = &nodes.Nodes[i]
			break
		}
	}

	if node == nil {
		fatalf("Node not found: %s", cmd.nodeID)
	}

	if cmd.template != "" {
		return tfortools.OutputToTemplate(os.Stdout, "node-show", cmd.template,
			node, nil)
	}

	dumpNode(node)

	return nil
}

func showCNCINode(cmd *nodeShowCommand) error {
	if cmd.nodeID == "" {
		fatalf("Missing required -node-id parameter")
	}

	var cnci types.CiaoCNCI

	url := buildComputeURL("cncis/%s/detail", cmd.nodeID)

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &cnci)
	if err != nil {
		fatalf(err.Error())
	}

	if cmd.template != "" {
		return tfortools.OutputToTemplate(os.Stdout, "node-show", cmd.template,
			&cnci, nil)
	}

	fmt.Printf("\tCNCI UUID: %s\n", cnci.ID)
	fmt.Printf("\tTenant UUID: %s\n", cnci.TenantID)
	fmt.Printf("\tIPv4: %s\n", cnci.IPv4)
	fmt.Printf("\tSubnets:\n")
	for _, subnet := range cnci.Subnets {
		fmt.Printf("\t\t%s\n", subnet.Subnet)
	}
	return nil
}

func nodeChangeStatus(nodeID string, status types.NodeStatusType) error {
	if !checkPrivilege() {
		fatalf("The evacuation of nodes is restricted to admin users")
	}

	nodeStatus := types.CiaoNodeStatus{Status: status}
	b, err := json.Marshal(&nodeStatus)
	if err != nil {
		fatalf(err.Error())
	}

	url, err := getCiaoResource("node", api.NodeV1)
	if err != nil {
		fatalf(err.Error())
	}

	url = fmt.Sprintf("%s/%s", url, nodeID)

	ver := api.NodeV1
	resp, err := sendCiaoRequest("PUT", url, nil, bytes.NewReader(b), ver)
	if err != nil {
		fatalf(err.Error())
	}

	if resp.StatusCode != http.StatusNoContent {
		fatalf("Node evacuation failed: %s", resp.Status)
	}

	return nil

}

type nodeEvacuateCommand struct {
	Flag   flag.FlagSet
	nodeID string
}

func (cmd *nodeEvacuateCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] node evacuate

Evacuate a node

The evacuate flags are:
`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *nodeEvacuateCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.nodeID, "node-id", "", "Node ID")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *nodeEvacuateCommand) run(args []string) error {
	return nodeChangeStatus(cmd.nodeID, types.NodeStatusMaintenance)
}

type nodeRestoreCommand struct {
	Flag   flag.FlagSet
	nodeID string
}

func (cmd *nodeRestoreCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] node restore

Restore a node.  Once restored instances can be scheduled on the node

The restore flags are:
`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *nodeRestoreCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.nodeID, "node-id", "", "Node ID")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *nodeRestoreCommand) run(args []string) error {
	return nodeChangeStatus(cmd.nodeID, types.NodeStatusReady)
}

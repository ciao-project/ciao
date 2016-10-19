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
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/openstack/compute"
)

const (
	osStart              = "os-start"
	osStop               = "os-stop"
	osDelete             = "os-delete"
	instanceTemplateDesc = `struct {
	HostID   string                               // ID of the host node
	ID       string                               // Instance UUID
	TenantID string                               // Tenant UUID
	Flavor   struct {
		ID string                             // Workload UUID
	}
	Image struct {
		ID string                             // Backing image UUID
	}
	Status    string                              // Instance status
	Addresses struct {
		Private []struct {
			Addr               string     // Instance IP address
			OSEXTIPSMACMacAddr string     // Instance MAC address
		}
	}
	SSHIP   string                                // Instance SSH IP address
	SSHPort int                                   // Instance SSH Port
	OsExtendedVolumesVolumesAttached []string     // list of attached volumes
}`
)

var instanceCommand = &command{
	SubCommands: map[string]subCommand{
		"add":     new(instanceAddCommand),
		"delete":  new(instanceDeleteCommand),
		"list":    new(instanceListCommand),
		"show":    new(instanceShowCommand),
		"restart": new(instanceRestartCommand),
		"stop":    new(instanceStopCommand),
	},
}

type instanceAddCommand struct {
	Flag      flag.FlagSet
	workload  string
	instances int
	label     string
}

func (cmd *instanceAddCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] instance add [flags]

Launches a new instance

The add flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *instanceAddCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.workload, "workload", "", "Workload UUID")
	cmd.Flag.IntVar(&cmd.instances, "instances", 1, "Number of instances to create")
	cmd.Flag.StringVar(&cmd.label, "label", "", "Set a frame label. This will trigger frame tracing")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *instanceAddCommand) run(args []string) error {
	if *tenantID == "" {
		errorf("Missing required -tenant-id parameter")
		cmd.usage()
	}

	if cmd.workload == "" {
		errorf("Missing required -workload parameter")
		cmd.usage()
	}

	var server compute.CreateServerRequest
	var servers compute.Servers

	server.Server.Name = cmd.label
	server.Server.Flavor = cmd.workload
	server.Server.MaxInstances = cmd.instances
	server.Server.MinInstances = 1

	serverBytes, err := json.Marshal(server)
	if err != nil {
		fatalf(err.Error())
	}
	body := bytes.NewReader(serverBytes)

	url := buildComputeURL("%s/servers", *tenantID)

	resp, err := sendHTTPRequest("POST", url, nil, body)
	if err != nil {
		fatalf(err.Error())
	}

	if resp.StatusCode != http.StatusAccepted {
		fatalf("Instance creation failed: %s", resp.Status)
	}

	err = unmarshalHTTPResponse(resp, &servers)
	if err != nil {
		fatalf(err.Error())
	}

	for _, server := range servers.Servers {
		fmt.Printf("Created new instance: %s\n", server.ID)
	}
	return nil
}

type instanceDeleteCommand struct {
	Flag     flag.FlagSet
	instance string
	all      bool
}

func (cmd *instanceDeleteCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] instance delete [flags]

Deltes a given instance

The delete flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *instanceDeleteCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.instance, "instance", "", "Instance UUID")
	cmd.Flag.BoolVar(&cmd.all, "all", false, "Delete all instances for the given tenant")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *instanceDeleteCommand) run(args []string) error {
	if cmd.all {
		return actionAllTenantInstance(*tenantID, osDelete)
	}

	if cmd.instance == "" {
		errorf("Missing required -instance parameter")
		cmd.usage()
	}

	url := buildComputeURL("%s/servers/%s", *tenantID, cmd.instance)

	resp, err := sendHTTPRequest("DELETE", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		fatalf("Instance deletion failed: %s", resp.Status)
	}

	fmt.Printf("Deleted instance: %s\n", cmd.instance)
	return nil
}

type instanceRestartCommand struct {
	Flag     flag.FlagSet
	instance string
}

func (cmd *instanceRestartCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] instance restart [flags]

Restart a stopped Ciao instance

The restart flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *instanceRestartCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.instance, "instance", "", "Instance UUID")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *instanceRestartCommand) run([]string) error {
	err := startStopInstance(cmd.instance, false)
	if err != nil {
		cmd.usage()
	}
	return err
}

type instanceStopCommand struct {
	Flag     flag.FlagSet
	instance string
}

func (cmd *instanceStopCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] instance stop [flags]

Stop a Ciao instance

The stop flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *instanceStopCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.instance, "instance", "", "Instance UUID")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *instanceStopCommand) run([]string) error {
	err := startStopInstance(cmd.instance, true)
	if err != nil {
		cmd.usage()
	}
	return err
}

func startStopInstance(instance string, stop bool) error {
	if *tenantID == "" {
		return errors.New("Missing required -tenant-id parameter")
	}

	if instance == "" {
		return errors.New("Missing required -instance parameter")
	}

	actionBytes := []byte(osStart)
	if stop == true {
		actionBytes = []byte(osStop)
	}

	body := bytes.NewReader(actionBytes)

	url := buildComputeURL("%s/servers/%s/action", *tenantID, instance)

	resp, err := sendHTTPRequest("POST", url, nil, body)
	if err != nil {
		fatalf(err.Error())
	}

	if resp.StatusCode != http.StatusAccepted {
		fatalf("Instance action failed: %s", resp.Status)
	}

	if stop == true {
		fmt.Printf("Instance %s stopped\n", instance)
	} else {
		fmt.Printf("Instance %s restarted\n", instance)
	}
	return nil
}

type instanceListCommand struct {
	Flag     flag.FlagSet
	workload string
	marker   string
	offset   int
	limit    int
	cn       string
	tenant   string
	detail   bool
	template string
}

func (cmd *instanceListCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] instance list [flags]

List instances for a tenant

The list flags are:

`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
The template passed to the -f option operates on a 

[]%s
`, instanceTemplateDesc)

	os.Exit(2)
}

func (cmd *instanceListCommand) parseArgs(args []string) []string {
	cmd.Flag.BoolVar(&cmd.detail, "detail", false, "Print detailed information about each instance")
	cmd.Flag.StringVar(&cmd.workload, "workload", "", "Workload UUID")
	cmd.Flag.StringVar(&cmd.cn, "cn", "", "Computer node to list instances from (default to all nodes when empty)")
	cmd.Flag.StringVar(&cmd.marker, "marker", "", "Show instance list starting from the next instance after marker")
	cmd.Flag.StringVar(&cmd.tenant, "tenant", "", "Specify to list instances from a tenant other than -tenant-id")
	cmd.Flag.IntVar(&cmd.offset, "offset", 0, "Show instance list starting from instance <offset>")
	cmd.Flag.IntVar(&cmd.limit, "limit", 0, "Limit list to <limit> results")
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

type byCreated []compute.ServerDetails

func (ss byCreated) Len() int           { return len(ss) }
func (ss byCreated) Swap(i, j int)      { ss[i], ss[j] = ss[j], ss[i] }
func (ss byCreated) Less(i, j int) bool { return ss[i].Created.Before(ss[j].Created) }

func (cmd *instanceListCommand) run(args []string) error {
	if cmd.tenant == "" {
		cmd.tenant = *tenantID
	}

	if cmd.cn != "" {
		return listNodeInstances(cmd.cn)
	}

	var servers compute.Servers
	var url string

	if cmd.workload != "" {
		url = buildComputeURL("flavors/%s/servers/detail", cmd.workload)
	} else {
		url = buildComputeURL("%s/servers/detail", cmd.tenant)
	}

	var values []queryValue
	if cmd.limit > 0 {
		values = append(values, queryValue{
			name:  "limit",
			value: fmt.Sprintf("%d", cmd.limit),
		})
	}

	if cmd.offset > 0 {
		values = append(values, queryValue{
			name:  "offset",
			value: fmt.Sprintf("%d", cmd.offset),
		})
	}

	if cmd.marker != "" {
		values = append(values, queryValue{
			name:  "marker",
			value: cmd.marker,
		})
	}

	resp, err := sendHTTPRequest("GET", url, values, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &servers)
	if err != nil {
		fatalf(err.Error())
	}

	sortedServers := []compute.ServerDetails{}
	for _, v := range servers.Servers {
		sortedServers = append(sortedServers, v)
	}
	sort.Sort(byCreated(sortedServers))

	if cmd.template != "" {
		return outputToTemplate("instance-list", cmd.template,
			&sortedServers)
	}

	w := new(tabwriter.Writer)
	if !cmd.detail {
		w.Init(os.Stdout, 0, 1, 1, ' ', 0)
		fmt.Fprintln(w, "#\tUUID\tStatus\tPrivate IP\tSSH IP\tSSH PORT")
	}

	for i, server := range sortedServers {
		if !cmd.detail {
			fmt.Fprintf(w, "%d", i+1)
			fmt.Fprintf(w, "\t%s", server.ID)
			fmt.Fprintf(w, "\t%s", server.Status)
			fmt.Fprintf(w, "\t%s", server.Addresses.Private[0].Addr)
			if server.SSHIP != "" {
				fmt.Fprintf(w, "\t%s", server.SSHIP)
				fmt.Fprintf(w, "\t%d\n", server.SSHPort)
			} else {
				fmt.Fprintf(w, "\tN/A")
				fmt.Fprintf(w, "\tN/A\n")
			}
			w.Flush()
		} else {
			fmt.Printf("Instance #%d\n", i+1)
			dumpInstance(&server)
		}
	}
	return nil
}

type instanceShowCommand struct {
	Flag     flag.FlagSet
	instance string
	template string
}

func (cmd *instanceShowCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] instance show [flags]

Print detailed information about an instance

The show flags are:

`)
	cmd.Flag.PrintDefaults()

	fmt.Fprintf(os.Stderr, `
The template passed to the -f option operates on a 

%s
`, instanceTemplateDesc)

	os.Exit(2)
}

func (cmd *instanceShowCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.instance, "instance", "", "Instance UUID")
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *instanceShowCommand) run(args []string) error {
	if cmd.instance == "" {
		errorf("Missing required -instance parameter")
		cmd.usage()
	}

	var server compute.Server
	url := buildComputeURL("%s/servers/%s", *tenantID, cmd.instance)

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}
	err = unmarshalHTTPResponse(resp, &server)
	if err != nil {
		fatalf(err.Error())
	}

	if cmd.template != "" {
		return outputToTemplate("instance-show", cmd.template,
			&server.Server)
	}

	dumpInstance(&server.Server)
	return nil
}

func dumpInstance(server *compute.ServerDetails) {
	fmt.Printf("\tUUID: %s\n", server.ID)
	fmt.Printf("\tStatus: %s\n", server.Status)
	fmt.Printf("\tPrivate IP: %s\n", server.Addresses.Private[0].Addr)
	fmt.Printf("\tMAC Address: %s\n", server.Addresses.Private[0].OSEXTIPSMACMacAddr)
	fmt.Printf("\tCN UUID: %s\n", server.HostID)
	fmt.Printf("\tImage UUID: %s\n", server.Image.ID)
	fmt.Printf("\tTenant UUID: %s\n", server.TenantID)
	if server.SSHIP != "" {
		fmt.Printf("\tSSH IP: %s\n", server.SSHIP)
		fmt.Printf("\tSSH Port: %d\n", server.SSHPort)
	}

	for _, vol := range server.OsExtendedVolumesVolumesAttached {
		fmt.Printf("\tVolume: %s\n", vol)
	}
}

func listNodeInstances(node string) error {
	if node == "" {
		fatalf("Missing required -cn parameter")
	}

	var servers types.CiaoServersStats
	url := buildComputeURL("nodes/%s/servers/detail", node)

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &servers)
	if err != nil {
		fatalf(err.Error())
	}

	for i, server := range servers.Servers {
		fmt.Printf("Instance #%d\n", i+1)
		fmt.Printf("\tUUID: %s\n", server.ID)
		fmt.Printf("\tStatus: %s\n", server.Status)
		fmt.Printf("\tTenant UUID: %s\n", server.TenantID)
		fmt.Printf("\tIPv4: %s\n", server.IPv4)
		fmt.Printf("\tCPUs used: %d\n", server.VCPUUsage)
		fmt.Printf("\tMemory used: %d MB\n", server.MemUsage)
		fmt.Printf("\tDisk used: %d MB\n", server.DiskUsage)
	}

	return nil
}

func actionAllTenantInstance(tenant string, osAction string) error {
	var action types.CiaoServersAction

	url := buildComputeURL("%s/servers/action", tenant)

	action.Action = osAction

	actionBytes, err := json.Marshal(action)
	if err != nil {
		fatalf(err.Error())
	}

	body := bytes.NewReader(actionBytes)

	resp, err := sendHTTPRequest("POST", url, nil, body)
	if err != nil {
		fatalf(err.Error())
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		fatalf("Action %s on all instances failed: %s", osAction, resp.Status)
	}

	fmt.Printf("%s all instances for tenant %s\n", osAction, tenant)
	return nil
}

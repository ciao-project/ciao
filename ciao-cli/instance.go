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
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"text/tabwriter"

	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/intel/tfortools"
	"github.com/pkg/errors"
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
	name      string
	template  string
}

func (cmd *instanceAddCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] instance add [flags]

Launches a new instance

The add flags are:

`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n%s", tfortools.GenerateUsageDecorated("f", []api.ServerDetails{}, nil))
	os.Exit(2)
}

func (cmd *instanceAddCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.workload, "workload", "", "Workload UUID")
	cmd.Flag.IntVar(&cmd.instances, "instances", 1, "Number of instances to create")
	cmd.Flag.StringVar(&cmd.label, "label", "", "Set a frame label. This will trigger frame tracing")
	cmd.Flag.StringVar(&cmd.name, "name", "", "Name for this instance. When multiple instances are requested this is used as a prefix")
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *instanceAddCommand) validateAddCommandArgs() {
	if c.TenantID == "" {
		errorf("Missing required -tenant-id parameter")
		cmd.usage()
	}

	if cmd.workload == "" {
		errorf("Missing required -workload parameter")
		cmd.usage()
	}

	if cmd.instances < 1 {
		errorf("Invalid value for -instances: %d", cmd.instances)
		cmd.usage()
	}

	if cmd.name != "" {
		r := regexp.MustCompile("^[a-z0-9-]{1,64}$")
		if !r.MatchString(cmd.name) {
			errorf("Requested name must be between 1 and 64 lowercase letters, numbers and hyphens")
		}
	}
}

func populateCreateServerRequest(cmd *instanceAddCommand, server *api.CreateServerRequest) {
	if cmd.label != "" {
		server.Server.Metadata = make(map[string]string)
		server.Server.Metadata["label"] = cmd.label
	}

	server.Server.WorkloadID = cmd.workload
	server.Server.MaxInstances = cmd.instances
	server.Server.MinInstances = 1
	server.Server.Name = cmd.name
}

func (cmd *instanceAddCommand) run(args []string) error {
	cmd.validateAddCommandArgs()

	var server api.CreateServerRequest
	var servers api.Servers

	populateCreateServerRequest(cmd, &server)

	servers, err := c.CreateInstances(server)
	if err != nil {
		return errors.Wrap(err, "Error creating instances")
	}

	if cmd.template != "" {
		return tfortools.OutputToTemplate(os.Stdout, "instance-add", cmd.template,
			&servers.Servers, nil)
	}

	if len(servers.Servers) < cmd.instances {
		fmt.Println("Some instances failed to start - check the event log for details.")
	}

	for _, server := range servers.Servers {
		fmt.Printf("Created new (pending) instance: %s\n", server.ID)
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

Deletes a given instance

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
		err := c.DeleteAllInstances()
		if err != nil {
			return errors.Wrap(err, "Error deleting all instances")
		}
		fmt.Printf("Deleted all instances\n")
		return nil
	}

	if cmd.instance == "" {
		errorf("Missing required -instance parameter")
		cmd.usage()
	}

	err := c.DeleteInstance(cmd.instance)
	if err != nil {
		return errors.Wrap(err, "Error deleting instance")
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
	if c.TenantID == "" {
		return errors.New("Missing required -tenant-id parameter")
	}

	if instance == "" {
		return errors.New("Missing required -instance parameter")
	}

	if stop == true {
		err := c.StopInstance(instance)
		if err != nil {
			return errors.Wrap(err, "Error stopping instance")
		}
		fmt.Printf("Instance %s stopped\n", instance)
	} else {
		err := c.StartInstance(instance)
		if err != nil {
			return errors.Wrap(err, "Error starting instance")
		}
		fmt.Printf("Instance %s restarted\n", instance)
	}
	return nil
}

type instanceListCommand struct {
	Flag     flag.FlagSet
	workload string
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
	fmt.Fprintf(os.Stderr, "\n%s", tfortools.GenerateUsageDecorated("f", []api.ServerDetails{}, nil))
	os.Exit(2)
}

func (cmd *instanceListCommand) parseArgs(args []string) []string {
	cmd.Flag.BoolVar(&cmd.detail, "detail", false, "Print detailed information about each instance")
	cmd.Flag.StringVar(&cmd.workload, "workload", "", "Workload UUID")
	cmd.Flag.StringVar(&cmd.cn, "cn", "", "Computer node to list instances from (default to all nodes when empty)")
	cmd.Flag.StringVar(&cmd.tenant, "tenant", "", "Specify to list instances from a tenant other than -tenant-id")
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

type byCreated []api.ServerDetails

func (ss byCreated) Len() int           { return len(ss) }
func (ss byCreated) Swap(i, j int)      { ss[i], ss[j] = ss[j], ss[i] }
func (ss byCreated) Less(i, j int) bool { return ss[i].Created.Before(ss[j].Created) }

func (cmd *instanceListCommand) run(args []string) error {
	if cmd.tenant == "" {
		cmd.tenant = c.TenantID
	}

	if cmd.cn != "" {
		return listNodeInstances(cmd.cn)
	}

	servers, err := c.ListInstancesByWorkload(cmd.tenant, cmd.workload)
	if err != nil {
		return errors.Wrap(err, "Error listing instances")
	}

	sortedServers := []api.ServerDetails{}
	for _, v := range servers.Servers {
		sortedServers = append(sortedServers, v)
	}
	sort.Sort(byCreated(sortedServers))

	if cmd.template != "" {
		return tfortools.OutputToTemplate(os.Stdout, "instance-list", cmd.template,
			&sortedServers, nil)
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
			fmt.Fprintf(w, "\t%s", server.PrivateAddresses[0].Addr)
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
	fmt.Fprintf(os.Stderr, "\n%s", tfortools.GenerateUsageDecorated("f", api.ServerDetails{}, nil))
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

	server, err := c.GetInstance(cmd.instance)
	if err != nil {
		return errors.Wrap(err, "Error getting instance")
	}

	if cmd.template != "" {
		return tfortools.OutputToTemplate(os.Stdout, "instance-show", cmd.template,
			&server.Server, nil)
	}

	dumpInstance(&server.Server)
	return nil
}

func dumpInstance(server *api.ServerDetails) {
	fmt.Printf("\tUUID: %s\n", server.ID)
	fmt.Printf("\tStatus: %s\n", server.Status)
	fmt.Printf("\tPrivate IP: %s\n", server.PrivateAddresses[0].Addr)
	fmt.Printf("\tMAC Address: %s\n", server.PrivateAddresses[0].MacAddr)
	fmt.Printf("\tCN UUID: %s\n", server.NodeID)
	fmt.Printf("\tTenant UUID: %s\n", server.TenantID)
	if server.SSHIP != "" {
		fmt.Printf("\tSSH IP: %s\n", server.SSHIP)
		fmt.Printf("\tSSH Port: %d\n", server.SSHPort)
	}

	for _, vol := range server.Volumes {
		fmt.Printf("\tVolume: %s\n", vol)
	}
}

func listNodeInstances(node string) error {
	if node == "" {
		fatalf("Missing required -cn parameter")
	}

	servers, err := c.ListInstancesByNode(node)
	if err != nil {
		return errors.Wrap(err, "Error getting instances for node")
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

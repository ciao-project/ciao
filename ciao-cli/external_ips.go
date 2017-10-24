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
	"net"
	"os"
	"text/tabwriter"

	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/intel/tfortools"
	"github.com/pkg/errors"
)

var externalIPCommand = &command{
	SubCommands: map[string]subCommand{
		"map":   new(externalIPMapCommand),
		"list":  new(externalIPListCommand),
		"unmap": new(externalIPUnMapCommand),
	},
}

type externalIPMapCommand struct {
	Flag       flag.FlagSet
	instanceID string
	poolName   string
}

func (cmd *externalIPMapCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] external-ip map [flags]

Map an external IP from a given pool to an instance.

The map flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *externalIPMapCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.instanceID, "instance", "", "ID of the instance to map IP to.")
	cmd.Flag.StringVar(&cmd.poolName, "pool", "", "Name of the pool to map from.")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *externalIPMapCommand) run(args []string) error {
	if cmd.instanceID == "" {
		errorf("Missing required -instance parameter")
		cmd.usage()
	}

	err := client.MapExternalIP(cmd.poolName, cmd.instanceID)
	if err != nil {
		return errors.Wrap(err, "Error mapping external IP")
	}

	fmt.Printf("Requested external IP for: %s\n", cmd.instanceID)

	return nil
}

type externalIPListCommand struct {
	Flag     flag.FlagSet
	template string
}

func (cmd *externalIPListCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] external-ip list [flags]

List all mapped external IPs.

The list flags are:

`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n%s", tfortools.GenerateUsageDecorated("f", []types.MappedIP{}, nil))
	os.Exit(2)
}

func (cmd *externalIPListCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *externalIPListCommand) run(args []string) error {
	IPs, err := client.ListExternalIPs()
	if err != nil {
		return errors.Wrap(err, "Error listing external IPs")
	}

	if cmd.template != "" {
		return tfortools.OutputToTemplate(os.Stdout, "external-ip-list", cmd.template,
			&IPs, nil)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 1, 1, ' ', 0)
	fmt.Fprintf(w, "#\tExternalIP\tInternalIP\tInstanceID")
	if client.IsPrivileged() {
		fmt.Fprintf(w, "\tTenantID\tPoolName\n")
	} else {
		fmt.Fprintf(w, "\n")
	}

	for i, IP := range IPs {
		fmt.Fprintf(w, "%d", i+1)
		fmt.Fprintf(w, "\t%s", IP.ExternalIP)
		fmt.Fprintf(w, "\t%s", IP.InternalIP)
		if IP.InstanceID != "" {
			fmt.Fprintf(w, "\t%s", IP.InstanceID)
		}

		if IP.TenantID != "" {
			fmt.Fprintf(w, "\t%s", IP.TenantID)
		}

		if IP.PoolName != "" {
			fmt.Fprintf(w, "\t%s", IP.PoolName)
		}

		fmt.Fprintf(w, "\n")
	}

	w.Flush()

	return nil
}

type externalIPUnMapCommand struct {
	address string
	Flag    flag.FlagSet
}

func (cmd *externalIPUnMapCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] external-ip unmap [flags]

Unmap a given external IP.

The unmap flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *externalIPUnMapCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.address, "address", "", "External IP to unmap.")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *externalIPUnMapCommand) run(args []string) error {
	if cmd.address == "" {
		errorf("Missing required -address parameter")
		cmd.usage()
	}

	err := client.UnmapExternalIP(cmd.address)
	if err != nil {
		return errors.Wrap(err, "Error unmapping external IP")
	}

	fmt.Printf("Requested unmap of: %s\n", cmd.address)

	return nil
}

var poolCommand = &command{
	SubCommands: map[string]subCommand{
		"create": new(poolCreateCommand),
		"list":   new(poolListCommand),
		"show":   new(poolShowCommand),
		"delete": new(poolDeleteCommand),
		"add":    new(poolAddCommand),
		"remove": new(poolRemoveCommand),
	},
}

type poolCreateCommand struct {
	Flag flag.FlagSet
	name string
}

func (cmd *poolCreateCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] pool create [flags]

Creates a new external IP pool.

The create flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

// TBD: add support for specifying a subnet or []ip addresses.
func (cmd *poolCreateCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.name, "name", "", "Name of pool")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *poolCreateCommand) run(args []string) error {
	if cmd.name == "" {
		errorf("Missing required -name parameter")
		cmd.usage()
	}

	err := client.CreateExternalIPPool(cmd.name)
	if err != nil {
		return errors.Wrap(err, "Error creating pool")
	}

	fmt.Printf("Created new pool: %s\n", cmd.name)

	return nil
}

type poolListCommand struct {
	Flag     flag.FlagSet
	template string
}

func (cmd *poolListCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] pool list [flags]

List all ciao external IP pools.

The list flags are:

`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n%s",
		tfortools.GenerateUsageDecorated("f", types.ListPoolsResponse{}.Pools, nil))
	os.Exit(2)
}

func (cmd *poolListCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

// change this command to return different output depending
// on the privilege level of user. Check privilege, then
// if not privileged, build non-privileged URL.
func (cmd *poolListCommand) run(args []string) error {
	pools, err := client.ListExternalIPPools()
	if err != nil {
		return errors.Wrap(err, "Error listing external IP pools")
	}

	if cmd.template != "" {
		return tfortools.OutputToTemplate(os.Stdout, "pool-list", cmd.template,
			&pools.Pools, nil)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 1, 1, ' ', 0)
	fmt.Fprintf(w, "#\tName")
	if client.IsPrivileged() {
		fmt.Fprintf(w, "\tTotalIPs\tFreeIPs\n")
	} else {
		fmt.Fprintf(w, "\n")
	}

	for i, pool := range pools.Pools {
		fmt.Fprintf(w, "%d", i+1)
		fmt.Fprintf(w, "\t%s", pool.Name)

		if pool.TotalIPs != nil {
			fmt.Fprintf(w, "\t%d", *pool.TotalIPs)
		}

		if pool.Free != nil {
			fmt.Fprintf(w, "\t%d", *pool.Free)
		}

		fmt.Fprintf(w, "\n")
	}

	w.Flush()

	return nil
}

type poolShowCommand struct {
	Flag     flag.FlagSet
	name     string
	template string
}

func (cmd *poolShowCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] pool show [flags]

Show ciao external IP pool details.

The show flags are:

`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\n%s", tfortools.GenerateUsageDecorated("f", types.Pool{}, nil))
	os.Exit(2)
}

func (cmd *poolShowCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.name, "name", "", "Name of pool")
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func dumpPool(pool types.Pool) {
	fmt.Printf("\tUUID: %s\n", pool.ID)
	fmt.Printf("\tName: %s\n", pool.Name)
	fmt.Printf("\tFree IPs: %d\n", pool.Free)
	fmt.Printf("\tTotal IPs: %d\n", pool.TotalIPs)

	for _, sub := range pool.Subnets {
		fmt.Printf("\tSubnet: %s\n", sub.CIDR)
	}

	for _, ip := range pool.IPs {
		fmt.Printf("\tIP Address: %s\n", ip.Address)
	}
}

func (cmd *poolShowCommand) run(args []string) error {
	var pool types.Pool

	if cmd.name == "" {
		errorf("Missing required -name parameter")
		cmd.usage()
	}

	pool, err := client.GetExternalIPPool(cmd.name)
	if err != nil {
		return errors.Wrap(err, "Error getting external IP pool")
	}

	if cmd.template != "" {
		return tfortools.OutputToTemplate(os.Stdout, "pool-show", cmd.template,
			&pool, nil)
	}

	dumpPool(pool)

	return nil
}

type poolDeleteCommand struct {
	Flag flag.FlagSet
	name string
}

func (cmd *poolDeleteCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] pool delete [flags]

Delete an unused ciao external IP pool.

The delete flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *poolDeleteCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.name, "name", "", "Name of pool")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *poolDeleteCommand) run(args []string) error {
	if cmd.name == "" {
		errorf("Missing required -name parameter")
		cmd.usage()
	}

	err := client.DeleteExternalIPPool(cmd.name)
	if err != nil {
		return errors.Wrap(err, "Error deleting external IP pool")
	}

	fmt.Printf("Deleted pool: %s\n", cmd.name)

	return nil
}

type poolAddCommand struct {
	Flag   flag.FlagSet
	name   string
	subnet string
}

func (cmd *poolAddCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] pool add [flags] [ip1 ip2...]

Add external IPs to a pool.

The add flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *poolAddCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.name, "name", "", "Name of pool")
	cmd.Flag.StringVar(&cmd.subnet, "subnet", "", "Subnet in CIDR format")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *poolAddCommand) run(args []string) error {
	if cmd.name == "" {
		errorf("Missing required -name parameter")
		cmd.usage()
	}

	if cmd.subnet != "" {
		// verify it's a good address.
		_, network, err := net.ParseCIDR(cmd.subnet)
		if err != nil {
			fatalf(err.Error())
		}

		if ones, bits := network.Mask.Size(); bits-ones < 2 {
			fatalf("Use address mode to add a single IP address")
		}

		err = client.AddExternalIPSubnet(cmd.name, network)
		if err != nil {
			return errors.Wrap(err, "Error adding external IP subnet")
		}
	} else if len(args) < 1 {
		errorf("Missing any addresses to add")
		cmd.usage()
	} else {
		for _, addr := range args {
			// verify it's a good address
			IP := net.ParseIP(addr)
			if IP == nil {
				fatalf("Invalid IP address")
			}
		}
		err := client.AddExternalIPAddresses(cmd.name, args)
		if err != nil {
			return errors.Wrap(err, "Error adding external IP addresses")
		}
	}

	fmt.Printf("Added new address to: %s\n", cmd.name)

	return nil
}

type poolRemoveCommand struct {
	Flag   flag.FlagSet
	name   string
	subnet string
	ip     string
}

func (cmd *poolRemoveCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] pool remove [flags]

Remove unmapped external IPs from a pool.

The remove flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *poolRemoveCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.name, "name", "", "Name of pool")
	cmd.Flag.StringVar(&cmd.subnet, "subnet", "", "Subnet in CIDR format")
	cmd.Flag.StringVar(&cmd.ip, "ip", "", "IPv4 Address")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *poolRemoveCommand) run(args []string) error {
	if cmd.name == "" {
		errorf("Missing required -name parameter")
		cmd.usage()
	}

	if cmd.subnet == "" && cmd.ip == "" {
		errorf("You must specify subnet or ip address to remove")
		cmd.usage()
	}

	if cmd.subnet != "" && cmd.ip != "" {
		errorf("You can only remove one item at a time")
		cmd.usage()
	}

	if cmd.subnet != "" {
		_, network, err := net.ParseCIDR(cmd.subnet)
		if err != nil {
			fatalf(err.Error())
		}

		err = client.RemoveExternalIPSubnet(cmd.name, network)
		if err != nil {
			return errors.Wrap(err, "Error removing external IP subnet")
		}
	}

	if cmd.ip != "" {
		err := client.RemoveExternalIPAddress(cmd.name, cmd.ip)
		if err != nil {
			return errors.Wrap(err, "Error removing external IP address")
		}
	}

	fmt.Printf("Removed address from pool: %s\n", cmd.name)
	return nil
}

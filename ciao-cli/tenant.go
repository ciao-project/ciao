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
	"text/template"

	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/uuid"
	"github.com/intel/tfortools"
	"github.com/pkg/errors"
)

// Project represents a tenant UUID and friendly name.
type Project struct {
	ID   string `mapstructure:"id"`
	Name string `mapstructure:"name"`
}

var tenantCommand = &command{
	SubCommands: map[string]subCommand{
		"list":   new(tenantListCommand),
		"update": new(tenantUpdateCommand),
		"create": new(tenantCreateCommand),
		"delete": new(tenantDeleteCommand),
	},
}

type tenantListCommand struct {
	Flag      flag.FlagSet
	quotas    bool
	resources bool
	config    bool
	all       bool
	tenantID  string
	template  string
}

type tenantUpdateCommand struct {
	Flag                       flag.FlagSet
	name                       string
	cidrPrefixSize             int
	createPrivilegedContainers bool
	tenantID                   string
}

type tenantCreateCommand struct {
	Flag                       flag.FlagSet
	name                       string
	cidrPrefixSize             int
	createPrivilegedContainers bool
	tenantID                   string
	template                   string
}

type tenantDeleteCommand struct {
	Flag     flag.FlagSet
	tenantID string
}

func (cmd *tenantUpdateCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] tenant update [flags]

Updates the configuration for the supplied tenant

The update flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *tenantUpdateCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.tenantID, "for-tenant", "", "Tenant to update")
	cmd.Flag.IntVar(&cmd.cidrPrefixSize, "cidr-prefix-size", 0, "Number of bits in network mask (12-30)")
	cmd.Flag.BoolVar(&cmd.createPrivilegedContainers, "create-privileged-containers", false, "Whether this tenant can create privileged containers")
	cmd.Flag.StringVar(&cmd.name, "name", "", "Tenant name")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *tenantUpdateCommand) run(args []string) error {
	if !c.IsPrivileged() {
		fatalf("Updating tenants is only available for privileged users")
	}

	// we should not require individual parameters?
	if cmd.name == "" && cmd.cidrPrefixSize == 0 {
		errorf("Missing required parameters")
		cmd.usage()
	}

	// subnet bits must be between 12 and 30
	if cmd.cidrPrefixSize != 0 && (cmd.cidrPrefixSize > 30 || cmd.cidrPrefixSize < 12) {
		errorf("cidr-prefix-size must be 12-30")
		cmd.usage()
	}

	config := types.TenantConfig{
		Name:       cmd.name,
		SubnetBits: cmd.cidrPrefixSize,
	}
	config.Permissions.PrivilegedContainers = cmd.createPrivilegedContainers

	return c.UpdateTenantConfig(cmd.tenantID, config)
}

func (cmd *tenantCreateCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] tenant create [flags]

Creates a new tenant with the supplied flags

The create flags are:

`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
The template passed to the -f option operates on the following struct:

%s`, tfortools.GenerateUsageUndecorated(types.TenantSummary{}))
	os.Exit(2)
}

func (cmd *tenantCreateCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.tenantID, "tenant", "", "ID for new tenant")
	cmd.Flag.IntVar(&cmd.cidrPrefixSize, "cidr-prefix-size", 0, "Number of bits in network mask (12-30)")
	cmd.Flag.BoolVar(&cmd.createPrivilegedContainers, "create-privileged-containers", false, "Whether this tenant can create privileged containers")
	cmd.Flag.StringVar(&cmd.name, "name", "", "Tenant name")
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *tenantCreateCommand) run(args []string) error {
	if !c.IsPrivileged() {
		fatalf("Creating tenants is only available for privileged users")
	}

	if cmd.tenantID == "" {
		errorf("Missing required tenantID")
		cmd.usage()
	}

	// CIDR prefix size must be between 12 and 30 bits
	if cmd.cidrPrefixSize != 0 && (cmd.cidrPrefixSize > 30 || cmd.cidrPrefixSize < 12) {
		errorf("cidr-prefix-size must be 12-30")
		cmd.usage()
	}

	var t *template.Template
	if cmd.template != "" {
		var err error
		t, err = tfortools.CreateTemplate("tenant-create", cmd.template, nil)
		if err != nil {
			fatalf(err.Error())
		}
	}

	tuuid, err := uuid.Parse(cmd.tenantID)
	if err != nil {
		fatalf("Tenant ID must be a UUID4")
	}

	config := types.TenantConfig{
		Name:       cmd.name,
		SubnetBits: cmd.cidrPrefixSize,
	}
	config.Permissions.PrivilegedContainers = cmd.createPrivilegedContainers

	summary, err := c.CreateTenantConfig(tuuid.String(), config)
	if err != nil {
		return errors.Wrap(err, "Error creating tenant configuration")
	}

	if t != nil {
		if err := t.Execute(os.Stdout, &summary); err != nil {
			fatalf(err.Error())
		}
		return nil
	}

	fmt.Printf("Tenant [%s]\n", summary.ID)
	fmt.Printf("\tName: %s\n", summary.Name)

	return nil
}

func (cmd *tenantDeleteCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] tenant delete [flags]

Deletes a tenant

The delete flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *tenantDeleteCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.tenantID, "tenant", "", "ID for new tenant")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *tenantDeleteCommand) run(args []string) error {
	if !c.IsPrivileged() {
		fatalf("Creating tenants is only available for privileged users")
	}

	if cmd.tenantID == "" {
		errorf("Missing required tenantID")
		cmd.usage()
	}

	err := c.DeleteTenant(cmd.tenantID)

	return errors.Wrap(err, "Error deleting tenant")
}

func (cmd *tenantListCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] tenant list

List tenants for the current user

The list flags are:

`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
The template passed to the -f option operates on the following structs:

no options:

%s
--quotas:

%s
--resources:

%s
--config:

%s
--all:

%s`,
		tfortools.GenerateUsageUndecorated([]Project{}),
		tfortools.GenerateUsageUndecorated(types.CiaoTenantResources{}),
		tfortools.GenerateUsageUndecorated(types.CiaoUsageHistory{}.Usages),
		tfortools.GenerateUsageUndecorated(types.TenantConfig{}),
		tfortools.GenerateUsageUndecorated(types.TenantsListResponse{}))

	fmt.Fprintln(os.Stderr, tfortools.TemplateFunctionHelp(nil))
	os.Exit(2)
}

func (cmd *tenantListCommand) parseArgs(args []string) []string {
	cmd.Flag.BoolVar(&cmd.quotas, "quotas", false, "List quotas status for a tenant")
	cmd.Flag.BoolVar(&cmd.resources, "resources", false, "List consumed resources for a tenant for the past 15mn")
	cmd.Flag.BoolVar(&cmd.config, "config", false, "List tenant config")
	cmd.Flag.BoolVar(&cmd.all, "all", false, "List all known tenants")
	cmd.Flag.StringVar(&cmd.tenantID, "for-tenant", "", "Tenant to get config for")
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *tenantListCommand) run(args []string) error {
	var t *template.Template
	if cmd.template != "" {
		var err error
		t, err = tfortools.CreateTemplate("tenant-list", cmd.template, nil)
		if err != nil {
			fatalf(err.Error())
		}
	}

	if cmd.quotas {
		return listTenantQuotas(t)
	}
	if cmd.resources {
		return listTenantResources(t)
	}
	if cmd.config {
		if c.IsPrivileged() == false {
			if c.TenantID == "" {
				fatalf("Missing required -tenant-id")
			}
			return listTenantConfig(t, c.TenantID)
		}

		if cmd.tenantID == "" {
			fatalf("Missing required -for-tenant parameter")
		}

		return listTenantConfig(t, cmd.tenantID)
	}
	if cmd.all {
		if c.IsPrivileged() == false {
			fatalf("The all command is for privileged users only")
		}
		return listAllTenants(t)
	}

	return listUserTenants(t)
}

func listUserTenants(t *template.Template) error {
	var projects []Project
	for _, t := range c.Tenants {
		projects = append(projects, Project{ID: t})
	}

	if t != nil {
		if err := t.Execute(os.Stdout, &projects); err != nil {
			fatalf(err.Error())
		}
		return nil
	}

	fmt.Printf("Projects for user\n")
	for _, project := range projects {
		fmt.Printf("\tUUID: %s\n", project.ID)
		fmt.Printf("\tName: %s\n", project.Name)
	}

	return nil
}

func listTenantQuotas(t *template.Template) error {
	if c.TenantID == "" {
		fatalf("Missing required -tenant-id parameter")
	}

	resources, err := c.ListTenantQuotas()
	if err != nil {
		return errors.Wrap(err, "Error listing tenant quotas")
	}

	if t != nil {
		if err := t.Execute(os.Stdout, &resources); err != nil {
			fatalf(err.Error())
		}
		fmt.Println("")
		return nil
	}

	fmt.Printf("Quotas for tenant %s:\n", resources.ID)
	fmt.Printf("\tInstances: %d | %s\n", resources.InstanceUsage, limitToString(resources.InstanceLimit))
	fmt.Printf("\tCPUs:      %d | %s\n", resources.VCPUUsage, limitToString(resources.VCPULimit))
	fmt.Printf("\tMemory:    %d | %s\n", resources.MemUsage, limitToString(resources.MemLimit))
	fmt.Printf("\tDisk:      %d | %s\n", resources.DiskUsage, limitToString(resources.DiskLimit))

	return nil
}

func listTenantResources(t *template.Template) error {
	if c.TenantID == "" {
		fatalf("Missing required -tenant-id parameter")
	}

	usage, err := c.ListTenantResources()
	if err != nil {
		return errors.Wrap(err, "Error listing tenant resources")
	}

	if t != nil {
		if err := t.Execute(os.Stdout, &usage.Usages); err != nil {
			fatalf(err.Error())
		}
		return nil
	}

	if len(usage.Usages) == 0 {
		fmt.Printf("No usage history for %s\n", c.TenantID)
		return nil
	}

	fmt.Printf("Usage for tenant %s:\n", c.TenantID)
	for _, u := range usage.Usages {
		fmt.Printf("\t%v: [%d CPUs] [%d MB memory] [%d MB disk]\n", u.Timestamp, u.VCPU, u.Memory, u.Disk)
	}

	return nil
}

func listTenantConfig(t *template.Template, tenantID string) error {
	config, err := c.GetTenantConfig(tenantID)
	if err != nil {
		fatalf(err.Error())
	}

	if t != nil {
		if err := t.Execute(os.Stdout, &config); err != nil {
			fatalf(err.Error())
		}
		return nil
	}

	fmt.Printf("Tenant [%s]\n", tenantID)
	fmt.Printf("\tName: %s\n", config.Name)
	fmt.Printf("\tCIDR Prefix Size: %d\n", config.SubnetBits)
	fmt.Printf("\tCan create privileged containers: %v\n", config.Permissions.PrivilegedContainers)

	return nil
}

func listAllTenants(t *template.Template) error {
	tenants, err := c.ListTenants()
	if err != nil {
		return errors.Wrap(err, "Error listing tenants")
	}

	if t != nil {
		if err := t.Execute(os.Stdout, &tenants); err != nil {
			fatalf(err.Error())
		}
		return nil
	}

	for i, tenant := range tenants.Tenants {
		fmt.Printf("Tenant [%d]\n", i+1)
		fmt.Printf("\tUUID: %s\n", tenant.ID)
		fmt.Printf("\tName: %s\n", tenant.Name)
	}

	return nil
}

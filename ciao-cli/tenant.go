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
	"os"
	"text/template"
	"time"

	"github.com/01org/ciao/ciao-controller/api"
	"github.com/01org/ciao/ciao-controller/types"
	"github.com/intel/tfortools"
)

func getCiaoTenantsResource() (string, error) {
	url, err := getCiaoResource("tenants", api.TenantsV1)
	return url, err
}

func getCiaoTenantRef(ID string) (string, error) {
	var tenants types.TenantsListResponse

	query := queryValue{
		name:  "id",
		value: ID,
	}

	url, err := getCiaoTenantsResource()
	if err != nil {
		return "", err
	}

	if !checkPrivilege() {
		return url, err
	}

	resp, err := sendCiaoRequest("GET", url, []queryValue{query}, nil, api.TenantsV1)
	if err != nil {
		return "", err
	}

	err = unmarshalHTTPResponse(resp, &tenants)
	if err != nil {
		return "", err
	}

	if len(tenants.Tenants) != 1 {
		return "", errors.New("No tenant by that ID found")
	}

	links := tenants.Tenants[0].Links
	url = getRef("self", links)
	if url == "" {
		return url, errors.New("invalid link returned from controller")
	}

	return url, nil
}

func getCiaoTenantConfig(ID string) (types.TenantConfig, error) {
	var config types.TenantConfig

	url, err := getCiaoTenantRef(ID)
	if err != nil {
		return config, err
	}

	resp, err := sendCiaoRequest("GET", url, nil, nil, api.TenantsV1)
	if err != nil {
		return config, err
	}

	err = unmarshalHTTPResponse(resp, &config)

	return config, err
}

func putCiaoTenantConfig(ID string, name string, bits int) error {
	var config types.TenantConfig

	url, err := getCiaoTenantRef(ID)
	if err != nil {
		return err
	}

	resp, err := sendCiaoRequest("GET", url, nil, nil, api.TenantsV1)
	if err != nil {
		return err
	}

	err = unmarshalHTTPResponse(resp, &config)

	if name != "" {
		config.Name = name
	}

	if bits != 0 {
		config.SubnetBits = bits
	}

	b, err := json.Marshal(config)
	if err != nil {
		fatalf(err.Error())
	}

	body := bytes.NewReader(b)

	_, err = sendCiaoRequest("PUT", url, nil, body, api.TenantsV1)

	return err
}

// Project represents a tenant UUID and friendly name.
type Project struct {
	ID   string `mapstructure:"id"`
	Name string `mapstructure:"name"`
}

var tenantCommand = &command{
	SubCommands: map[string]subCommand{
		"list":   new(tenantListCommand),
		"update": new(tenantUpdateCommand),
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
	Flag       flag.FlagSet
	name       string
	subnetBits int
	tenantID   string
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
	cmd.Flag.IntVar(&cmd.subnetBits, "subnet-bits", 0, "Number of bits in subnet mask")
	cmd.Flag.StringVar(&cmd.name, "name", "", "Tenant name")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *tenantUpdateCommand) run(args []string) error {
	if !checkPrivilege() {
		fatalf("Updating tenants is only available for privileged users")
	}

	// we should not require individual parameters?
	if cmd.name == "" && cmd.subnetBits == 0 {
		errorf("Missing required parameters")
		cmd.usage()
	}

	// subnet bits must be between 4 and 30
	if cmd.subnetBits != 0 && (cmd.subnetBits > 30 || cmd.subnetBits < 4) {
		errorf("subnet_bits must be 4-30")
		cmd.usage()
	}

	return putCiaoTenantConfig(cmd.tenantID, cmd.name, cmd.subnetBits)
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
		if checkPrivilege() == false {
			if *tenantID == "" {
				fatalf("Missing required -tenant-id")
			}
			return listTenantConfig(t, *tenantID)
		}

		if cmd.tenantID == "" {
			fatalf("Missing required -for-tenant parameter")
		}

		return listTenantConfig(t, cmd.tenantID)
	}
	if cmd.all {
		if checkPrivilege() == false {
			fatalf("The all command is for privileged users only")
		}
		return listAllTenants(t)
	}

	return listUserTenants(t)
}

func listUserTenants(t *template.Template) error {
	var projects []Project
	for _, t := range tenants {
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
	if *tenantID == "" {
		fatalf("Missing required -tenant-id parameter")
	}

	var resources types.CiaoTenantResources
	url := buildComputeURL("%s/quotas", *tenantID)

	resp, err := sendHTTPRequest("GET", url, nil, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &resources)
	if err != nil {
		fatalf(err.Error())
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
	if *tenantID == "" {
		fatalf("Missing required -tenant-id parameter")
	}

	var usage types.CiaoUsageHistory
	url := buildComputeURL("%s/resources", *tenantID)

	now := time.Now()
	values := []queryValue{
		{
			name:  "start_date",
			value: now.Add(-15 * time.Minute).Format(time.RFC3339),
		},
		{
			name:  "end_date",
			value: now.Format(time.RFC3339),
		},
	}

	resp, err := sendHTTPRequest("GET", url, values, nil)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &usage)
	if err != nil {
		fatalf(err.Error())
	}

	if t != nil {
		if err := t.Execute(os.Stdout, &usage.Usages); err != nil {
			fatalf(err.Error())
		}
		return nil
	}

	if len(usage.Usages) == 0 {
		fmt.Printf("No usage history for %s\n", *tenantID)
		return nil
	}

	fmt.Printf("Usage for tenant %s:\n", *tenantID)
	for _, u := range usage.Usages {
		fmt.Printf("\t%v: [%d CPUs] [%d MB memory] [%d MB disk]\n", u.Timestamp, u.VCPU, u.Memory, u.Disk)
	}

	return nil
}

func listTenantConfig(t *template.Template, tenantID string) error {
	config, err := getCiaoTenantConfig(tenantID)
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
	fmt.Printf("\tSubnetBits: %d\n", config.SubnetBits)

	return nil
}

func listAllTenants(t *template.Template) error {
	var tenants types.TenantsListResponse

	url, err := getCiaoTenantsResource()
	if err != nil {
		fatalf(err.Error())
	}

	resp, err := sendCiaoRequest("GET", url, nil, nil, api.TenantsV1)
	if err != nil {
		fatalf(err.Error())
	}

	err = unmarshalHTTPResponse(resp, &tenants)
	if err != nil {
		fatalf(err.Error())
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

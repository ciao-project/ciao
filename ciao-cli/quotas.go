//
// Copyright (c) 2017 Intel Corporation
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
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/intel/tfortools"
	"github.com/pkg/errors"
)

var quotasCommand = &command{
	SubCommands: map[string]subCommand{
		"update": new(quotasUpdateCommand),
		"list":   new(quotasListCommand),
	},
}

type quotasUpdateCommand struct {
	Flag     flag.FlagSet
	name     string
	value    string
	tenantID string
}

func (cmd *quotasUpdateCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] quotas update [flags]

Updates the quota entry for the supplied tenant

The update flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *quotasUpdateCommand) parseArgs(args []string) []string {
	cmd.Flag.StringVar(&cmd.name, "name", "", "Name of quota or limit")
	cmd.Flag.StringVar(&cmd.value, "value", "", "Value of quota or limit")
	cmd.Flag.StringVar(&cmd.tenantID, "for-tenant", "", "Tenant to update quota for")
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *quotasUpdateCommand) run(args []string) error {
	if !client.IsPrivileged() {
		fatalf("Updating quotas is only available for privileged users")
	}

	if cmd.name == "" {
		errorf("Missing required -name parameter")
		cmd.usage()
	}

	if cmd.value == "" {
		errorf("Missing required -value parameter")
		cmd.usage()
	}

	if cmd.tenantID == "" {
		errorf("Missing required -for-tenant parameter")
		cmd.usage()
	}
	var v int
	if cmd.value == "unlimited" {
		v = -1
	} else {
		var err error
		v, err = strconv.Atoi(cmd.value)
		if err != nil {
			fatalf(err.Error())
		}
	}

	quotas := []types.QuotaDetails{{
		Name:  cmd.name,
		Value: v,
	}}

	err := client.UpdateQuotas(cmd.tenantID, quotas)
	if err != nil {
		return errors.Wrap(err, "Error updating quotas")
	}

	fmt.Printf("Update quotas succeeded\n")

	return nil
}

type quotasListCommand struct {
	Flag     flag.FlagSet
	template string
	tenantID string
}

func (cmd *quotasListCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `usage: ciao-cli [options] quotas list [flags]

Show all quotas for current tenant or supplied tenant if admin

The list flags are:

`)
	cmd.Flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
The template passed to the -f option operates on a 

%s`,
		tfortools.GenerateUsageUndecorated([]types.QuotaDetails{}))
	fmt.Fprintln(os.Stderr, tfortools.TemplateFunctionHelp(nil))

	os.Exit(2)
}

func (cmd *quotasListCommand) parseArgs(args []string) []string {
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.StringVar(&cmd.tenantID, "for-tenant", "", "Tenant to get quotas for")
	cmd.Flag.StringVar(&cmd.template, "f", "", "Template used to format output")
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

// change this command to return different output depending
// on the privilege level of user. Check privilege, then
// if not privileged, build non-privileged URL.
func (cmd *quotasListCommand) run(args []string) error {
	if cmd.tenantID != "" {
		if !client.IsPrivileged() {
			fatalf("Listing quotas for other tenants is for privileged users only")
		}

	} else {
		if client.IsPrivileged() {
			fatalf("Admin user must specify the tenant with -for-tenant")
		}
	}

	results, err := client.ListQuotas(cmd.tenantID)
	if err != nil {
		return errors.Wrap(err, "Error listing quotas")
	}

	if cmd.template != "" {
		return tfortools.OutputToTemplate(os.Stdout, "quotas-list", cmd.template,
			results.Quotas, nil)
	}

	fmt.Printf("Quotas for tenant: %s\n", cmd.tenantID)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	for _, qd := range results.Quotas {
		fmt.Fprintf(w, "%s:\t", qd.Name)
		if strings.Contains(qd.Name, "quota") {
			fmt.Fprintf(w, "%d of ", qd.Usage)
		}
		if qd.Value == -1 {
			fmt.Fprint(w, "unlimited\n")
		} else {
			fmt.Fprintf(w, "%d\n", qd.Value)
		}
	}
	w.Flush()
	return nil
}

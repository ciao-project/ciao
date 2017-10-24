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

	"github.com/ciao-project/ciao/client"
	"github.com/golang/glog"
)

// Item serves to represent a group of related commands
type command struct {
	SubCommands map[string]subCommand
}

// This is not used but needed to comply with subCommand interface
func (c *command) parseArgs(args []string) []string {
	return args
}

func (c *command) run(args []string) error {
	cmdName := args[0]
	subCmdName := args[1]
	subCmd := c.SubCommands[subCmdName]
	if subCmd == nil {
		c.usage(cmdName)
	}
	args = subCmd.parseArgs(args[2:])
	prepareForCommand()
	return subCmd.run(args)
}

// usage prints the available commands in an item
func (c *command) usage(name ...string) {
	fmt.Fprintf(os.Stderr, `ciao-cli: Command-line interface for the Cloud Integrated Advanced Orchestrator (CIAO)

Usage:

	ciao-cli [options] `+name[0]+` sub-command [flags]
`)

	var t = template.Must(template.New("commandTemplate").Parse(commandTemplate))
	t.Execute(os.Stderr, c)

	fmt.Fprintf(os.Stderr, `
Use "ciao-cli `+name[0]+` sub-command -help" for more information about that item.
`)
	os.Exit(2)
}

// subCommand is the interface that all cli commands should implement
type subCommand interface {
	usage(...string)
	parseArgs([]string) []string
	run([]string) error
}

var commands = map[string]subCommand{
	"instance":    instanceCommand,
	"workload":    workloadCommand,
	"tenant":      tenantCommand,
	"event":       eventCommand,
	"node":        nodeCommand,
	"trace":       traceCommand,
	"image":       imageCommand,
	"volume":      volumeCommand,
	"pool":        poolCommand,
	"external-ip": externalIPCommand,
	"quotas":      quotasCommand,
}

func infof(format string, args ...interface{}) {
	if glog.V(1) {
		glog.InfoDepth(1, fmt.Sprintf("ciao-cli INFO: "+format, args...))
	}
}

func errorf(format string, args ...interface{}) {
	glog.ErrorDepth(1, fmt.Sprintf("ciao-cli ERROR: "+format, args...))
}

func fatalf(format string, args ...interface{}) {
	glog.FatalDepth(1, fmt.Sprintf("ciao-cli FATAL: "+format, args...))
}

var (
	controllerURLFlag  = flag.String("controller", "", "Controller URL")
	tenantIDFlag       = flag.String("tenant-id", "", "Tenant UUID")
	caCertFileFlag     = flag.String("ca-file", "", "CA Certificate")
	clientCertFileFlag = flag.String("client-cert-file", "", "Path to certificate for authenticating with controller")
)

const (
	ciaoControllerEnv     = "CIAO_CONTROLLER"
	ciaoCACertFileEnv     = "CIAO_CA_CERT_FILE"
	ciaoClientCertFileEnv = "CIAO_CLIENT_CERT_FILE"
)

var c client.Client

func limitToString(limit int) string {
	if limit == -1 {
		return "Unlimited"
	}

	return fmt.Sprintf("%d", limit)
}

func getCiaoEnvVariables() {
	controller := os.Getenv(ciaoControllerEnv)
	ca := os.Getenv(ciaoCACertFileEnv)
	clientCert := os.Getenv(ciaoClientCertFileEnv)

	infof("Ciao environment variables:\n")
	infof("\t%s:%s\n", ciaoControllerEnv, controller)
	infof("\t%s:%s\n", ciaoCACertFileEnv, ca)
	infof("\t%s:%s\n", ciaoClientCertFileEnv, clientCert)

	c.ControllerURL = controller
	c.CACertFile = ca
	c.ClientCertFile = clientCert

	if *controllerURLFlag != "" {
		c.ControllerURL = *controllerURLFlag
	}

	if *caCertFileFlag != "" {
		c.CACertFile = *caCertFileFlag
	}

	if *clientCertFileFlag != "" {
		c.ClientCertFile = *clientCertFileFlag
	}

	if *tenantIDFlag != "" {
		c.TenantID = *tenantIDFlag
	}
}

func prepareForCommand() {
	err := c.Init()
	if err != nil {
		fatalf(err.Error())
	}
}

func main() {
	var err error

	flag.Usage = usage
	flag.Parse()

	getCiaoEnvVariables()

	// Print usage if no arguments are given
	args := flag.Args()
	if len(args) < 1 {
		usage()
	}

	// Find command in cmdline args
	cmdName := args[0]
	cmd := commands[cmdName]
	if cmd == nil {
		usage()
	}
	if len(args) < 2 {
		cmd.usage(cmdName)
	}

	// Execute the command
	err = cmd.run(args)
	if err != nil {
		fatalf(err.Error())
	}
}

const usageTemplate1 = `ciao-cli: Command-line interface for the Cloud Integrated Advanced Orchestrator (CIAO)

Usage:

	ciao-cli [options] command sub-command [flags]

The options are:

`

const usageTemplate2 = `

The commands are:
{{range $command, $subCommand := .}}
	{{$command}}{{end}}

Use "ciao-cli command -help" for more information about that command.
`

const commandTemplate = `
The sub-commands are:
{{range $name, $cmd := .SubCommands}}
	{{$name}}{{end}}
`

func usage() {
	var t = template.Must(template.New("usageTemplate1").Parse(usageTemplate1))
	t.Execute(os.Stderr, nil)
	flag.PrintDefaults()
	t = template.Must(template.New("usageTemplate2").Parse(usageTemplate2))
	t.Execute(os.Stderr, commands)
	os.Exit(2)
}

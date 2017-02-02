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
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/01org/ciao/ciao-controller/api"
	"github.com/01org/ciao/ciao-controller/types"
)

var configCommand = &command{
	SubCommands: map[string]subCommand{
		"update": new(configUpdateCommand),
		"show":   new(configShowCommand),
	},
}

type configUpdateCommand struct {
	Flag    flag.FlagSet
	element string
	value   string
}

func (cmd *configUpdateCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `
Usage:

ciao-cli [options] config update [flags]

update cluster configuration

The update flags are:

`)
	cmd.Flag.PrintDefaults()
	os.Exit(2)
}

func (cmd *configUpdateCommand) parseArgs(args []string) []string {
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.StringVar(&cmd.element, "element", "", "configuration field to change")
	cmd.Flag.StringVar(&cmd.value, "value", "", "new value for element to change")
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *configUpdateCommand) run(args []string) error {
	var req types.ConfigRequest
	var response interface{}
	if cmd.element == "" {
		fmt.Fprintf(os.Stderr, "Missing required -element parameter\n")
		cmd.usage()
	}
	if cmd.value == "" {
		fmt.Fprintf(os.Stderr, "Missing required -value parameter\n")
		cmd.usage()
	}
	if validConfigElement(cmd.element) == false {
		fmt.Fprintf(os.Stderr, "'%s' is not a valid element of the cluster configuration\n", cmd.element)
		os.Exit(2)
	}
	if validConfigValue(cmd.value, cmd.element) == false {
		fmt.Fprintf(os.Stderr, "'%s' is not a valid value for %s\n", cmd.value, cmd.element)
		os.Exit(2)
	}

	url := buildCiaoURL("configuration")
	req = types.ConfigRequest{
		Element: cmd.element,
		Value:   cmd.value,
	}
	b, err := json.Marshal(req)
	if err != nil {
		fatalf(err.Error())
	}
	body := bytes.NewReader(b)

	ver := api.ClusterConfigV1

	resp, err := sendCiaoRequest("POST", url, nil, body, &ver)
	if err != nil {
		return err
	}

	err = unmarshalHTTPResponse(resp, &response)
	if err != nil {
		return err
	}
	fmt.Println(response)

	return nil
}

// validConfigElement checks if the element to be modified matches
// the the elements that are allowed to be changed
func validConfigElement(s string) bool {
	switch s {
	case "scheduler.storage_uri":
		fallthrough
	case "controller.compute_port":
		fallthrough
	case "launcher.disk_limit":
		fallthrough
	case "launcher.mem_limit":
		fallthrough
	case "identity_service.type":
		fallthrough
	case "identity_service.url":
		return true

	}
	return false
}

func validConfigValue(s string, t string) bool {
	switch t {
	case "scheduler.storage_uri":
		return validConfigURI(s, "file")
	case "identity_service.url":
		return validConfigURI(s, "https")
	case "controller.compute_port":
		return validConfigNumber(s)
	case "launcher.disk_limit":
		fallthrough
	case "launcher.mem_limit":
		return validConfigBoolean(s)
	case "identity_service.type":
		return "keystone" == s
	}
	return false
}

// validConfigNumber checks that current input is a sane integer number
func validConfigNumber(s string) bool {
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}
	return false
}

func sanatizeBoolean(s string) string {
	s = strings.ToLower(s)
	switch s {
	case "1":
		fallthrough
	case "t":
		fallthrough
	case "true":
		return "true"
	case "0":
		fallthrough
	case "f":
		fallthrough
	case "false":
		return "false"
	}
	return ""
}

// validConfigBoolean returns true if the input (s) is correct value
// for a boolean and its representation for the configuration yaml payload
func validConfigBoolean(s string) bool {
	s = strings.ToLower(s)
	switch s {
	case "1":
		fallthrough
	case "t":
		fallthrough
	case "true":
		fallthrough
	case "0":
		fallthrough
	case "f":
		fallthrough
	case "false":
		return true
	}
	return false
}

// validConfigURI check correctness of the URI given, evaluating
// if the string to be analized matches with the expected scheme
// and meets the URI elements needed for scheme
func validConfigURI(s string, scheme string) bool {
	uri, err := url.Parse(s)
	if err != nil {
		return false
	}
	if scheme != uri.Scheme {
		return false
	}
	switch scheme {
	case "file":
		return uri.Path != ""
	case "https":
		return uri.Host != ""
	}
	return false
}

type configShowCommand struct {
	Flag flag.FlagSet
}

func (cmd *configShowCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `
Usage:

ciao-cli [options] config show

Show current cluster configuration
`)
	os.Exit(2)
}

func (cmd *configShowCommand) parseArgs(args []string) []string {
	cmd.Flag.Usage = func() { cmd.usage() }
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *configShowCommand) run(args []string) error {
	var response interface{}

	ver := api.ClusterConfigV1
	url := buildCiaoURL("configuration")
	resp, err := sendCiaoRequest("GET", url, nil, nil, &ver)
	if err != nil {
		return err
	}

	err = unmarshalHTTPResponse(resp, &response)
	if err != nil {
		return err
	}
	fmt.Println(response)

	return nil
}

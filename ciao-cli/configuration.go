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

type confElement string

type configUpdateCommand struct {
	Flag    flag.FlagSet
	element confElement
	value   string
}

func (e *confElement) String() string {
	return string(*e)
}

func (e *confElement) Set(s string) error {
	switch string(s) {
	case "scheduler.storage_uri",
		"controller.compute_port",
		"launcher.disk_limit",
		"launcher.mem_limit",
		"identity_service.type",
		"identity_service.url":

		*e = confElement(s)
		return nil
	}
	return fmt.Errorf("invalid element of the cluster configuration")

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
	cmd.Flag.Var(&cmd.element, "element", "configuration field to change")
	cmd.Flag.StringVar(&cmd.value, "value", "", "new value for element to change")
	cmd.Flag.Parse(args)
	return cmd.Flag.Args()
}

func (cmd *configUpdateCommand) run(args []string) error {
	var req types.ConfigRequest
	var response types.ConfigUpdateResponse
	if cmd.element == "" {
		fmt.Fprintf(os.Stderr, "Missing required -element parameter\n")
		cmd.usage()
	}

	if cmd.value == "" {
		fmt.Fprintf(os.Stderr, "Missing required -value parameter\n")
		cmd.usage()
	}
	if validConfigValue(cmd.value, string(cmd.element)) == false {
		return fmt.Errorf("'%s' is not a valid value for %s", cmd.value, cmd.element)
	}

	if cmd.element == "launcher.mem_limit" || cmd.element == "launcher.disk_limit" {
		cmd.value = sanatizeBoolean(cmd.value)
	}

	url := buildCiaoURL("configuration")
	req = types.ConfigRequest{
		Element: string(cmd.element),
		Value:   cmd.value,
	}
	b, err := json.Marshal(req)
	if err != nil {
		fatalf(err.Error())
	}
	body := bytes.NewReader(b)
	ver := api.ClusterConfigV1

	resp, err := sendCiaoRequest("PUT", url, nil, body, &ver)
	if err != nil {
		return err
	}

	err = unmarshalHTTPResponse(resp, &response)
	if err != nil {
		return err
	}
	fmt.Println(response.Response)
	return nil
}

func validConfigValue(s string, t string) bool {
	switch t {
	case "scheduler.storage_uri":
		return validConfigURI(s, "file")
	case "identity_service.url":
		return validConfigURI(s, "https")
	case "controller.compute_port":
		return validConfigNumber(s)
	case "launcher.disk_limit", "launcher.mem_limit":
		return validConfigBoolean(s)
	case "identity_service.type":
		return s == "keystone"
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
	case "1", "t", "true":
		return "true"
	case "0", "f", "false":
		return "false"
	}
	return ""
}

// validConfigBoolean returns true if the input (s) is correct value
// for a boolean and its representation for the configuration yaml payload
func validConfigBoolean(s string) bool {
	s = strings.ToLower(s)
	switch s {
	case "1", "t", "true", "0", "f", "false":
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
		// check hostname is explicit (e.g: "https://:35357" is invalid)
		return (uri.Host != "") && (strings.HasPrefix(uri.Host, ":") == false)
	}
	return false
}

type configShowCommand struct {
	Flag flag.FlagSet
}

func (cmd *configShowCommand) usage(...string) {
	fmt.Fprintf(os.Stderr, `
Usage:

ciao-cli config show

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
	var response types.ConfigShowResponse

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
	fmt.Print(response.Configuration)

	return nil
}

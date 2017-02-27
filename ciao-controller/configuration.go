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

package main

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/payloads"
	"gopkg.in/yaml.v2"
)

func (c *controller) UpdateClusterConfig(newConf types.ConfigRequest) error {
	var pyld payloads.Configure
	if newConf.Element == "" {
		return errors.New("empty Element")
	}
	if newConf.Value == "" {
		return errors.New("empty Value")
	}
	if validConfigElement(newConf.Element) == false {
		return errors.New("invalid Value")
	}
	if validConfigValue(newConf.Value, newConf.Element) == false {
		return errors.New("invalid Element")
	}
	pyld, err := c.client.ssntpClient().ClusterConfiguration()
	if err != nil {
		return err
	}
	err = updateConfPayload(newConf, &pyld)
	if err != nil {
		return err
	}
	newConfPyld, err := yaml.Marshal(pyld)
	if err != nil {
		return err
	}
	c.client.UpdateConfig(newConfPyld)
	return nil
}

func (c *controller) ShowClusterConfig() (string, error) {
	pyld, err := c.client.ssntpClient().ClusterConfiguration()
	if err != nil {
		return "", err
	}
	// no need to share sensitive information
	pyld.Configure.Controller.IdentityPassword = ""
	pyld.Configure.Controller.AdminPassword = ""
	pyld.Configure.Controller.AdminSSHKey = ""
	s, err := yaml.Marshal(pyld)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s", string(s)), nil
}

func updateConfPayload(confReq types.ConfigRequest, pyld *payloads.Configure) error {
	badValueErr := fmt.Errorf("Unable to set '%s' value for '%s'", confReq.Value, confReq.Element)
	switch confReq.Element {
	case "scheduler.storage_uri":
		pyld.Configure.Scheduler.ConfigStorageURI = confReq.Value
	case "controller.compute_port":
		newVal, err := strconv.Atoi(confReq.Value)
		if err != nil {
			return badValueErr
		}
		pyld.Configure.Controller.ComputePort = newVal
	case "launcher.disk_limit":
		newVal, err := strconv.ParseBool(confReq.Value)
		if err != nil {
			return badValueErr
		}
		pyld.Configure.Launcher.DiskLimit = newVal
	case "launcher.mem_limit":
		newVal, err := strconv.ParseBool(confReq.Value)
		if err != nil {
			return badValueErr
		}
		pyld.Configure.Launcher.MemoryLimit = newVal
	case "identity_service.type":
		if confReq.Element != "keystone" {
			return badValueErr
		}
	case "identity_service.url":
		pyld.Configure.IdentityService.URL = confReq.Value
	}
	return nil
}

// validConfigElement checks if the element to be modified matches
// the the elements that are allowed to be changed
func validConfigElement(s string) bool {
	switch s {
	case "scheduler.storage_uri", "controller.compute_port",
		"launcher.disk_limit", "launcher.mem_limit",
		"identity_service.type", "identity_service.url":
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

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
	"github.com/01org/ciao/payloads"
	"github.com/golang/glog"
)

//Will implement a simple database to persist state across
//restarts of the host/VM

func dbProcessCommand(client *agentClient, cmd *cmdWrapper) {

	switch netCmd := cmd.cmd.(type) {

	case *payloads.EventTenantAdded:

		c := &netCmd.TenantAdded
		glog.Infof("Tenant Added %v", c)

	case *payloads.EventTenantRemoved:

		c := &netCmd.TenantRemoved
		glog.Infof("Tenant Removed %v", c)

	case *payloads.CommandAssignPublicIP:

		c := &netCmd.AssignIP
		glog.Infof("Assign IP %v", c)

	case *payloads.CommandReleasePublicIP:

		c := &netCmd.ReleaseIP
		glog.Infof("Release IP %v", c)

	default:
		glog.Errorf("Processing unknown command %v", netCmd)

	}
}

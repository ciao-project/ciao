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

package libsnnet

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/vishvananda/netlink"
)

const (
	prefixBridge   = "sbr"
	prefixVnic     = "svn"
	prefixVnicCont = "svp"
	prefixVnicHost = "svn"
	prefixCnciVnic = "svc"
	prefixGretap   = "sgt"
)

const ifaceRetryLimit = 10

var (
	ifaceRseed rand.Source
	ifaceRsrc  *rand.Rand
)

func init() {
	ifaceRseed = rand.NewSource(time.Now().UnixNano())
	ifaceRsrc = rand.New(ifaceRseed)
}

func validSnPrefix(s string) bool {
	switch {
	case strings.HasPrefix(s, prefixBridge):
	case strings.HasPrefix(s, prefixVnic):
	case strings.HasPrefix(s, prefixVnicCont):
	case strings.HasPrefix(s, prefixVnicHost):
	case strings.HasPrefix(s, prefixCnciVnic):
	case strings.HasPrefix(s, prefixGretap):
	default:
		return false
	}

	return true
}

// GenIface generates locally unique interface names based on the
// type of device passed in. It will additionally check if the
// interface name exists on the localhost based on unique
// When uniqueness is specified error will be returned
// if it is not possible to generate a locally unique name within
// a finite number of retries
func GenIface(device interface{}, unique bool) (string, error) {
	var prefix string

	switch d := device.(type) {
	case *Bridge:
		prefix = prefixBridge
	case *Vnic:
		switch d.Role {
		case TenantVM:
			prefix = prefixVnic
		case TenantContainer:
			prefix = prefixVnicHost
		}
	case *GreTunEP:
		prefix = prefixGretap
	case *CnciVnic:
		prefix = prefixCnciVnic
	default:
		return "", fmt.Errorf("invalid device type %T %v", device, device)
	}

	if !unique {
		iface := fmt.Sprintf("%s_%x", prefix, ifaceRsrc.Uint32())
		return iface, nil
	}

	for i := 0; i < ifaceRetryLimit; i++ {
		iface := fmt.Sprintf("%s_%x", prefix, ifaceRsrc.Uint32())
		if _, err := netlink.LinkByName(iface); err != nil {
			return iface, nil
		}
	}

	// The chances of the failure are remote
	return "", fmt.Errorf("unable to create unique interface name")
}

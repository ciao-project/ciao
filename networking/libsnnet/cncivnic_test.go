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
	"net"
	"strings"
	"testing"

	"github.com/vishvananda/netlink"
)

//Just pick the first physical interface with an IP
func getFirstPhyDevice() (int, error) {

	links, err := netlink.LinkList()
	if err != nil {
		return 0, err
	}

	for _, link := range links {

		if link.Type() != "device" {
			continue
		}

		if link.Attrs().Name == "lo" {
			continue
		}

		addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
		if err != nil || len(addrs) == 0 {
			continue
		}

		return link.Attrs().Index, nil
	}

	return 0, fmt.Errorf("Unable to obtain physical device")

}

//Test CNCI VNIC primitives
//
//Tests all the primitives used to create a CNCI instance
//compatible vnic including create, enable, disable, destroy
//
//Test is expected to pass
func TestCnciVnic_Basic(t *testing.T) {

	cnciVnic, _ := newCnciVnic("testcnciVnic")

	pIndex, err := getFirstPhyDevice()

	if err != nil {
		t.Errorf("CnciVnic creation failed: %v", err)
	}

	cnciVnic.Link.ParentIndex = pIndex
	cnciVnic.Link.HardwareAddr, _ = net.ParseMAC("DE:AD:BE:EF:01:02")

	if err := cnciVnic.create(); err != nil {
		t.Errorf("CnciVnic creation failed: %v", err)
	}

	if err := cnciVnic.enable(); err != nil {
		t.Errorf("CnciVnic enable failed: %v", err)
	}

	if err := cnciVnic.disable(); err != nil {
		t.Errorf("CnciVnic enable failed: %v", err)
	}

	if err := cnciVnic.destroy(); err != nil {
		t.Errorf("CnciVnic deletion failed: %v", err)
	}
	if err := cnciVnic.destroy(); err == nil {
		t.Errorf("CnciVnic deletion should have failed")
	}

}

//Test duplicate creation
//
//Tests the creation of a duplicate interface is handled
//gracefully
//
//Test is expected to pass
func TestCnciVnic_Dup(t *testing.T) {
	cnciVnic, _ := newCnciVnic("testcnciVnic")

	pIndex, err := getFirstPhyDevice()
	if err != nil {
		t.Errorf("CnciVnic creation failed: %v", err)
	}
	cnciVnic.Link.ParentIndex = pIndex

	if err := cnciVnic.create(); err != nil {
		t.Errorf("CnciVnic creation failed: %v", err)
	}
	if err := cnciVnic.create(); err == nil {
		t.Errorf("CnciVnic creation should have failed")
	}

	defer func() { _ = cnciVnic.destroy() }()

	cnciVnic1, _ := newCnciVnic("testcnciVnic")
	cnciVnic1.Link.ParentIndex = pIndex

	if err := cnciVnic1.create(); err == nil {
		t.Errorf("Duplicate CnciVnic creation: %v", err)
	}

}

//Negative test cases
//
//Tests for graceful handling of various Negative
//primitive invocation scenarios
//
//Test is expected to pass
func TestCnciVnic_Invalid(t *testing.T) {
	cnciVnic, err := newCnciVnic("testcnciVnic")

	if err = cnciVnic.getDevice(); err == nil {
		t.Errorf("Non existent device: %v", cnciVnic)
	}
	if !strings.HasPrefix(err.Error(), "cncivnic error") {
		t.Errorf("Invalid error format %v", err)
	}

	if err = cnciVnic.enable(); err == nil {
		t.Errorf("Non existent device: %v", cnciVnic)
	}
	if !strings.HasPrefix(err.Error(), "cncivnic error") {
		t.Errorf("Invalid error format %v", err)
	}

	if err = cnciVnic.disable(); err == nil {
		t.Errorf("Non existent device: %v", cnciVnic)
	}
	if !strings.HasPrefix(err.Error(), "cncivnic error") {
		t.Errorf("Invalid error format %v", err)
	}

	if err = cnciVnic.destroy(); err == nil {
		t.Errorf("Non existent device: %v", cnciVnic)
	}
	if !strings.HasPrefix(err.Error(), "cncivnic error") {
		t.Errorf("Invalid error format %v", err)
	}

}

//Test ability to attach
//
//Tests that you can attach to an existing CNCI VNIC and
//perform all CNCI VNIC operations on the attached VNIC
//
//Test is expected to pass
func TestCnciVnic_GetDevice(t *testing.T) {
	cnciVnic1, _ := newCnciVnic("testcnciVnic")

	pIndex, err := getFirstPhyDevice()
	if err != nil {
		t.Errorf("CnciVnic creation failed: %v", err)
	}
	cnciVnic1.Link.ParentIndex = pIndex

	if err := cnciVnic1.create(); err != nil {
		t.Errorf("CnciVnic creation failed: %v", err)
	}

	cnciVnic, _ := newCnciVnic("testcnciVnic")

	if err := cnciVnic.getDevice(); err != nil {
		t.Errorf("CnciVnic Get Device failed: %v", err)
	}

	if err := cnciVnic.enable(); err != nil {
		t.Errorf("CnciVnic enable failed: %v", err)
	}

	if err := cnciVnic.disable(); err != nil {
		t.Errorf("CnciVnic enable failed: %v", err)
	}

	if err := cnciVnic.destroy(); err != nil {
		t.Errorf("CnciVnic deletion failed: %v", err)
	}
}

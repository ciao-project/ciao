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

package libsnnet_test

import (
	"strings"
	"testing"

	"github.com/01org/ciao/networking/libsnnet"
)

//Tests all the basic VNIC primitives
//
//Tests for creation, get, enable, disable and destroy
//primitives. If any of these fail, it may be a issue
//with the underlying netlink or kernel dependencies
//
//Test is expected to pass
func TestVnic_Basic(t *testing.T) {

	vnic, _ := libsnnet.NewVnic("testvnic")

	if err := vnic.Create(); err != nil {
		t.Errorf("Vnic creation failed: %v", err)
	}

	vnic1, _ := libsnnet.NewVnic("testvnic")

	if err := vnic1.GetDevice(); err != nil {
		t.Errorf("Vnic Get Device failed: %v", err)
	}

	if err := vnic.Enable(); err != nil {
		t.Errorf("Vnic enable failed: %v", err)
	}

	if err := vnic.Disable(); err != nil {
		t.Errorf("Vnic enable failed: %v", err)
	}

	if err := vnic.Destroy(); err != nil {
		t.Errorf("Vnic deletion failed: %v", err)
	}

}

//Tests all the basic Container VNIC primitives
//
//Tests for creation, get, enable, disable and destroy
//primitives. If any of these fail, it may be a issue
//with the underlying netlink or kernel dependencies
//
//Test is expected to pass
func TestVnicContainer_Basic(t *testing.T) {

	vnic, _ := libsnnet.NewContainerVnic("testvnic")

	if err := vnic.Create(); err != nil {
		t.Errorf("Vnic creation failed: %v", err)
	}

	vnic1, _ := libsnnet.NewContainerVnic("testvnic")

	if err := vnic1.GetDevice(); err != nil {
		t.Errorf("Vnic Get Device failed: %v", err)
	}

	if err := vnic.Enable(); err != nil {
		t.Errorf("Vnic enable failed: %v", err)
	}

	if err := vnic.Disable(); err != nil {
		t.Errorf("Vnic enable failed: %v", err)
	}

	if err := vnic.Destroy(); err != nil {
		t.Errorf("Vnic deletion failed: %v", err)
	}

}

//Duplicate VNIC creation detection
//
//Checks if the VNIC create primitive fails gracefully
//on a duplicate VNIC creation
//
//Test is expected to pass
func TestVnic_Dup(t *testing.T) {
	vnic, _ := libsnnet.NewVnic("testvnic")

	if err := vnic.Create(); err != nil {
		t.Errorf("Vnic creation failed: %v", err)
	}

	defer vnic.Destroy()

	vnic1, _ := libsnnet.NewVnic("testvnic")

	if err := vnic1.Create(); err == nil {
		t.Errorf("Duplicate Vnic creation: %v", err)
	}

}

//Duplicate Container VNIC creation detection
//
//Checks if the VNIC create primitive fails gracefully
//on a duplicate VNIC creation
//
//Test is expected to pass
func TestVnicContainer_Dup(t *testing.T) {
	vnic, _ := libsnnet.NewVnic("testconvnic")

	if err := vnic.Create(); err != nil {
		t.Errorf("Vnic creation failed: %v", err)
	}

	defer vnic.Destroy()

	vnic1, _ := libsnnet.NewVnic("testconvnic")

	if err := vnic1.Create(); err == nil {
		t.Errorf("Duplicate Vnic creation: %v", err)
	}

}

//Negative test case for VNIC primitives
//
//Simulates various error scenarios and ensures that
//they are handled gracefully
//
//Test is expected to pass
func TestVnic_Invalid(t *testing.T) {
	vnic, err := libsnnet.NewVnic("testvnic")

	if err = vnic.GetDevice(); err == nil {
		t.Errorf("Non existent device: %v", vnic)
	}
	if !strings.HasPrefix(err.Error(), "vnic error") {
		t.Errorf("Invalid error format %v", err)
	}

	if err = vnic.Enable(); err == nil {
		t.Errorf("Non existent device: %v", vnic)
	}
	if !strings.HasPrefix(err.Error(), "vnic error") {
		t.Errorf("Invalid error format %v", err)
	}

	if err = vnic.Disable(); err == nil {
		t.Errorf("Non existent device: %v", vnic)
	}
	if !strings.HasPrefix(err.Error(), "vnic error") {
		t.Errorf("Invalid error format %v", err)
	}

	if err = vnic.Destroy(); err == nil {
		t.Errorf("Non existent device: %v", vnic)
	}
	if !strings.HasPrefix(err.Error(), "vnic error") {
		t.Errorf("Invalid error format %v", err)
	}

}

//Negative test case for Container VNIC primitives
//
//Simulates various error scenarios and ensures that
//they are handled gracefully
//
//Test is expected to pass
func TestVnicContainer_Invalid(t *testing.T) {
	vnic, err := libsnnet.NewContainerVnic("testcvnic")

	if err = vnic.GetDevice(); err == nil {
		t.Errorf("Non existent device: %v", vnic)
	}
	if !strings.HasPrefix(err.Error(), "vnic error") {
		t.Errorf("Invalid error format %v", err)
	}

	if err = vnic.Enable(); err == nil {
		t.Errorf("Non existent device: %v", vnic)
	}
	if !strings.HasPrefix(err.Error(), "vnic error") {
		t.Errorf("Invalid error format %v", err)
	}

	if err = vnic.Disable(); err == nil {
		t.Errorf("Non existent device: %v", vnic)
	}
	if !strings.HasPrefix(err.Error(), "vnic error") {
		t.Errorf("Invalid error format %v", err)
	}

	if err = vnic.Destroy(); err == nil {
		t.Errorf("Non existent device: %v", vnic)
	}
	if !strings.HasPrefix(err.Error(), "vnic error") {
		t.Errorf("Invalid error format %v", err)
	}

}

//Test ability to attach to an existing Vnic
//
//Tests the the ability to attach to an existing
//vnic and perform all VNIC operations on it
//
//Test is expected to pass
func TestVnic_GetDevice(t *testing.T) {
	vnic1, _ := libsnnet.NewVnic("testvnic")

	if err := vnic1.Create(); err != nil {
		t.Errorf("Vnic creation failed: %v", err)
	}

	vnic, _ := libsnnet.NewVnic("testvnic")

	if err := vnic.GetDevice(); err != nil {
		t.Errorf("Vnic Get Device failed: %v", err)
	}

	if err := vnic.Enable(); err != nil {
		t.Errorf("Vnic enable failed: %v", err)
	}

	if err := vnic.Disable(); err != nil {
		t.Errorf("Vnic enable failed: %v", err)
	}

	if err := vnic.Destroy(); err != nil {
		t.Errorf("Vnic deletion failed: %v", err)
	}
}

//Test ability to attach to an existing Container Vnic
//
//Tests the the ability to attach to an existing
//vnic and perform all VNIC operations on it
//
//Test is expected to pass
func TestVnicContainer_GetDevice(t *testing.T) {
	vnic1, _ := libsnnet.NewContainerVnic("testvnic")

	if err := vnic1.Create(); err != nil {
		t.Errorf("Vnic creation failed: %v", err)
	}

	vnic, _ := libsnnet.NewContainerVnic("testvnic")

	if err := vnic.GetDevice(); err != nil {
		t.Errorf("Vnic Get Device failed: %v", err)
	}

	if err := vnic.Enable(); err != nil {
		t.Errorf("Vnic enable failed: %v", err)
	}

	if err := vnic.Disable(); err != nil {
		t.Errorf("Vnic enable failed: %v", err)
	}

	if err := vnic.Destroy(); err != nil {
		t.Errorf("Vnic deletion failed: %v", err)
	}
}

//Tests VNIC attach to a bridge
//
//Tests all interactions between VNIC and Bridge
//
//Test is expected to pass
func TestVnic_Bridge(t *testing.T) {
	vnic, _ := libsnnet.NewVnic("testvnic")
	bridge, _ := libsnnet.NewBridge("testbridge")

	if err := vnic.Create(); err != nil {
		t.Errorf("Vnic Create failed: %v", err)
	}

	defer vnic.Destroy()

	if err := bridge.Create(); err != nil {
		t.Errorf("Vnic Create failed: %v", err)
	}
	defer bridge.Destroy()

	if err := vnic.Attach(bridge); err != nil {
		t.Errorf("Vnic attach failed: %v", err)
	}

	if err := vnic.Enable(); err != nil {
		t.Errorf("Vnic enable failed: %v", err)
	}

	if err := bridge.Enable(); err != nil {
		t.Errorf("Vnic deletion failed: %v", err)
	}

	if err := vnic.Detach(bridge); err != nil {
		t.Errorf("Vnic detach failed: %v", err)
	}

}

//Tests Container VNIC attach to a bridge
//
//Tests all interactions between VNIC and Bridge
//
//Test is expected to pass
func TestVnicContainer_Bridge(t *testing.T) {
	vnic, _ := libsnnet.NewContainerVnic("testvnic")
	bridge, _ := libsnnet.NewBridge("testbridge")

	if err := vnic.Create(); err != nil {
		t.Errorf("Vnic Create failed: %v", err)
	}

	defer vnic.Destroy()

	if err := bridge.Create(); err != nil {
		t.Errorf("Vnic Create failed: %v", err)
	}
	defer bridge.Destroy()

	if err := vnic.Attach(bridge); err != nil {
		t.Errorf("Vnic attach failed: %v", err)
	}

	if err := vnic.Enable(); err != nil {
		t.Errorf("Vnic enable failed: %v", err)
	}

	if err := bridge.Enable(); err != nil {
		t.Errorf("Vnic deletion failed: %v", err)
	}

	if err := vnic.Detach(bridge); err != nil {
		t.Errorf("Vnic detach failed: %v", err)
	}
}

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
	"testing"

	"github.com/stretchr/testify/assert"
)

func performVnicOps(shouldPass bool, assert *assert.Assertions, vnic *Vnic) {
	a := assert.Nil
	if !shouldPass {
		a = assert.NotNil
	}
	a(vnic.enable())
	a(vnic.disable())
	a(vnic.destroy())
}

//Tests all the basic VNIC primitives
//
//Tests for creation, get, enable, disable and destroy
//primitives. If any of these fail, it may be a issue
//with the underlying netlink or kernel dependencies
//
//Test is expected to pass
func TestVnic_Basic(t *testing.T) {
	assert := assert.New(t)

	vnic, err := newVnic("testvnic")
	assert.Nil(err)
	assert.Nil(vnic.create())

	vnic1, err := newVnic("testvnic")
	assert.Nil(err)

	assert.Nil(vnic1.getDevice())
	assert.NotEqual(vnic.interfaceName(), "")
	assert.Equal(vnic.peerName(), vnic.interfaceName())

	performVnicOps(true, assert, vnic)
}

//Tests all the basic Container VNIC primitives
//
//Tests for creation, get, enable, disable and destroy
//primitives. If any of these fail, it may be a issue
//with the underlying netlink or kernel dependencies
//
//Test is expected to pass
func TestVnicContainer_Basic(t *testing.T) {
	assert := assert.New(t)

	vnic, _ := newContainerVnic("testvnic")
	assert.Nil(vnic.create())

	vnic1, _ := newContainerVnic("testvnic")
	assert.Nil(vnic1.getDevice())

	performVnicOps(true, assert, vnic)
}

//Duplicate VNIC creation detection
//
//Checks if the VNIC create primitive fails gracefully
//on a duplicate VNIC creation
//
//Test is expected to pass
func TestVnic_Dup(t *testing.T) {
	assert := assert.New(t)
	vnic, _ := newVnic("testvnic")
	vnic1, _ := newVnic("testvnic")

	assert.Nil(vnic.create())
	defer func() { _ = vnic.destroy() }()
	assert.NotNil(vnic1.create())
}

//Duplicate Container VNIC creation detection
//
//Checks if the VNIC create primitive fails gracefully
//on a duplicate VNIC creation
//
//Test is expected to pass
func TestVnicContainer_Dup(t *testing.T) {
	assert := assert.New(t)
	vnic, _ := newContainerVnic("testconvnic")
	vnic1, _ := newContainerVnic("testconvnic")

	assert.Nil(vnic.create())
	defer func() { _ = vnic.destroy() }()
	assert.NotNil(vnic1.create())
}

//Negative test case for VNIC primitives
//
//Simulates various error scenarios and ensures that
//they are handled gracefully
//
//Test is expected to pass
func TestVnic_Invalid(t *testing.T) {
	assert := assert.New(t)
	vnic, err := newVnic("testvnic")
	assert.Nil(err)

	assert.NotNil(vnic.getDevice())

	performVnicOps(false, assert, vnic)
}

//Negative test case for Container VNIC primitives
//
//Simulates various error scenarios and ensures that
//they are handled gracefully
//
//Test is expected to pass
func TestVnicContainer_Invalid(t *testing.T) {
	assert := assert.New(t)

	vnic, err := newContainerVnic("testcvnic")
	assert.Nil(err)

	assert.NotNil(vnic.getDevice())

	performVnicOps(false, assert, vnic)
}

//Test ability to attach to an existing VNIC
//
//Tests the ability to attach to an existing
//VNIC and perform all VNIC operations on it
//
//Test is expected to pass
func TestVnic_GetDevice(t *testing.T) {
	assert := assert.New(t)
	vnic1, _ := newVnic("testvnic")

	assert.Nil(vnic1.create())
	vnic, _ := newVnic("testvnic")

	assert.Nil(vnic.getDevice())
	assert.NotEqual(vnic.interfaceName(), "")
	assert.Equal(vnic.interfaceName(), vnic1.interfaceName())
	assert.NotEqual(vnic1.peerName(), "")
	assert.Equal(vnic1.peerName(), vnic.peerName())

	performVnicOps(true, assert, vnic)
}

//Test ability to attach to an existing Container VNIC
//
//Tests the ability to attach to an existing
//VNIC and perform all VNIC operations on it
//
//Test is expected to pass
func TestVnicContainer_GetDevice(t *testing.T) {
	assert := assert.New(t)

	vnic1, err := newContainerVnic("testvnic")
	assert.Nil(err)

	err = vnic1.create()
	assert.Nil(err)

	vnic, err := newContainerVnic("testvnic")
	assert.Nil(err)

	assert.Nil(vnic.getDevice())
	performVnicOps(true, assert, vnic)
}

//Tests VNIC attach to a bridge
//
//Tests all interactions between VNIC and Bridge
//
//Test is expected to pass
func TestVnic_Bridge(t *testing.T) {
	assert := assert.New(t)
	vnic, _ := newVnic("testvnic")
	bridge, _ := NewBridge("testbridge")

	assert.Nil(vnic.create())
	defer func() { _ = vnic.destroy() }()

	assert.Nil(bridge.Create())
	defer func() { _ = bridge.Destroy() }()

	assert.Nil(vnic.attach(bridge))
	assert.Nil(vnic.enable())
	assert.Nil(bridge.Enable())
	assert.Nil(vnic.detach(bridge))

}

//Tests Container VNIC attach to a bridge
//
//Tests all interactions between VNIC and Bridge
//
//Test is expected to pass
func TestVnicContainer_Bridge(t *testing.T) {
	assert := assert.New(t)
	vnic, _ := newContainerVnic("testvnic")
	bridge, _ := NewBridge("testbridge")

	assert.Nil(vnic.create())

	defer func() { _ = vnic.destroy() }()

	assert.Nil(bridge.Create())
	defer func() { _ = bridge.Destroy() }()

	assert.Nil(vnic.attach(bridge))
	assert.Nil(vnic.enable())
	assert.Nil(bridge.Enable())
	assert.Nil(vnic.detach(bridge))
}

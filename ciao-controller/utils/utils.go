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

package utils

import (
	"crypto/rand"
	"net"
)

// NewTenantHardwareAddr will generate a MAC address for a tenant instance.
func NewTenantHardwareAddr(ip net.IP) net.HardwareAddr {
	buf := make([]byte, 6)
	ipBytes := ip.To4()
	buf[0] |= 2
	buf[1] = 0
	copy(buf[2:6], ipBytes)
	return net.HardwareAddr(buf)
}

// NewHardwareAddr will generate a MAC address for a CNCI.
func NewHardwareAddr() (net.HardwareAddr, error) {
	buf := make([]byte, 6)
	_, err := rand.Read(buf)
	if err != nil {
		return nil, err
	}

	// vnic creation seems to require not just the
	// bit 1 to be set, but the entire byte to be
	// set to 2.  Also, ensure that we get no
	// overlap with tenant mac addresses by not allowing
	// byte 1 to ever be zero.
	buf[0] = 2
	if buf[1] == 0 {
		buf[1] = 3
	}

	hw := net.HardwareAddr(buf)

	return hw, nil
}

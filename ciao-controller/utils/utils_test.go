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
	"net"
	"testing"
)

// TestNewTenantHardwareAddr
// Confirm that the mac addresses generated from a given
// IP address is as expected.
func TestNewTenantHardwareAddr(t *testing.T) {
	ip := net.ParseIP("172.16.0.2")
	expectedMAC := "02:00:ac:10:00:02"
	hw := NewTenantHardwareAddr(ip)
	if hw.String() != expectedMAC {
		t.Error("Expected: ", expectedMAC, " Received: ", hw.String())
	}
}

// TestHardwareAddr
// Confirm that the mac addresses generated from a given
// IP address is as expected.
func TestHardwareAddr(t *testing.T) {
	hw, err := NewHardwareAddr()
	if err != nil {
		t.Fatal(err)
	}

	// byte 0 has to be 2, byte 1 can never be zero
	if hw[0] != 2 {
		t.Fatalf("Expected byte zero to be 2, Got %v", hw[0])
	}

	if hw[1] == 0 {
		t.Fatal("Byte 1 may never be zero")
	}
}

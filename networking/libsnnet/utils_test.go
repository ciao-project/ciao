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
	"net"
	"testing"
)

const (
	testCIDRIPv4 = "198.51.100.0/24"
	testCIDRIPv6 = "2001:db8::/32"
)

func TestEqualNetSlice(t *testing.T) {
	netSlice1 := []string{"192.168.0.0/24", "192.168.5.0/24", "192.168.42.0/24"}
	netSlice2 := []string{"192.168.0.0/24", "192.168.5.0/24", "192.168.42.0/24"}

	equalSlices := EqualNetSlice(netSlice1, netSlice2)
	if equalSlices == false {
		t.Fatalf("Expected true, got %v", equalSlices)
	}
}

// TestIPNetDeepCopyIPv4 tests that given an *IPNet (ip1)
// creates a full copy of that struct (ip2), so if we
// modify the copy, original struct should not change
// this test is expected to pass
func TestIPNetDeepCopyIPv4(t *testing.T) {
	_, ip1, _ := net.ParseCIDR(testCIDRIPv4)
	ip2 := IPNetDeepCopy(*ip1)

	// check address of structs are different
	if ip1 == ip2 {
		t.Fatalf("expected a copy of %v, got a reference", ip1)
	}

	// change values of copy to verify original is not changed
	ip2.IP = []byte{42, 42, 42, 42}
	if ip1.String() != testCIDRIPv4 {
		t.Fatalf("expected %v, got %v", testCIDRIPv4, ip1.String())
	}
}

// TestIPNetDeepCopyIPv6 perform a test similar to
// TestIPNetDeepCopyIPv4 but using an IPv6 CIDR as input
// this test is expected to pass
func TestIPNetDeepCopyIPv6(t *testing.T) {
	_, ip1, _ := net.ParseCIDR(testCIDRIPv6)
	ip2 := IPNetDeepCopy(*ip1)

	// check address of structs are different
	if ip1 == ip2 {
		t.Fatalf("expected a copy of %v, got a reference", ip1)
	}

	// change values of copy to verify original is not changed
	// new value "2001:db8:2001:db8:2001:db8:2001:db8"
	ip2.IP = []byte{32, 01, 13, 184, 32, 01, 13, 184, 32, 01, 13, 184, 32, 01, 13, 184}
	if ip1.String() != testCIDRIPv6 {
		t.Fatalf("expected %v, got %v", testCIDRIPv6, ip1.String())
	}
}

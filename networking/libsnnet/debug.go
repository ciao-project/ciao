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

	"github.com/vishvananda/netlink"
)

// Show dumps as much information about the device as possible to stdout
func Show(name string) error {

	link, err := netlink.LinkByAlias(name)

	if err != nil {
		fmt.Println("fetching by name")
		link, err = netlink.LinkByName(name)
		if err != nil {
			return fmt.Errorf("interface name and alias does not exist: %v", name)
		}
	}

	switch t := link.(type) {
	default:
		fmt.Printf("Type: %v\n", t)
		fmt.Printf("Link type: %v\n", link.Type())
		fmt.Println("Attributes :", link.Attrs())
		fmt.Println("Alias :", link.Attrs().Alias)
		fmt.Println("Details : ", link)
	}

	return nil
}

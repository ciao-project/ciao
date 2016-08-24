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

import "github.com/01org/ciao/osprepare"

var launcherDeps = osprepare.PackageRequirements{
	// docker for containers
	// qemu/kvm for VM's
	// xorriso for cloud init config drive

	"clearlinux": {
		{"/usr/bin/docker", "cloud-control"},
		{"/usr/bin/qemu-system-x86_64", "cloud-control"},
		{"/usr/bin/xorriso", "cloud-control"},
	},
	"fedora": {
		{"/usr/bin/docker", "docker-engine"},
		{"/usr/bin/qemu-system-x86_64", "qemu-system-x86"},
		{"/usr/bin/xorriso", "xorriso"},
	},
	"ubuntu": {
		{"/usr/bin/docker", "docker"},
		{"/usr/bin/qemu-system-x86_64", "qemu-system-x86"},
		{"/usr/bin/xorriso", "xorriso"},
	},
}

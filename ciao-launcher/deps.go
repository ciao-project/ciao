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

// common launcher node needs are:
//
// qemu/kvm for VM's
// xorriso for cloud init config drive
// fuser for qemu instance pid

var launcherClearLinuxCommonDeps = []osprepare.PackageRequirement{
	{"/usr/bin/qemu-system-x86_64", "cloud-control"},
	{"/usr/bin/xorriso", "cloud-control"},
	{"/usr/sbin/fuser", "cloud-control"},
}

var launcherFedoraCommonDeps = []osprepare.PackageRequirement{
	{"/usr/bin/qemu-system-x86_64", "qemu-system-x86"},
	{"/usr/bin/xorriso", "xorriso"},
	{"/usr/sbin/fuser", "psmisc"},
}

var launcherUbuntuCommonDeps = []osprepare.PackageRequirement{
	{"/usr/bin/qemu-system-x86_64", "qemu-system-x86"},
	{"/usr/bin/xorriso", "xorriso"},
	{"/bin/fuser", "psmisc"},
}

var launcherNetNodeDeps = map[string][]osprepare.PackageRequirement{
	// network nodes have a unique additional need for:
	//
	// 	none currently

	"clearlinux": launcherClearLinuxCommonDeps,
	"fedora":     launcherFedoraCommonDeps,
	"ubuntu":     launcherUbuntuCommonDeps,
}

var launcherComputeNodeDeps = map[string][]osprepare.PackageRequirement{
	// compute nodes have a unique additional need for:
	//
	// docker for containers

	"clearlinux": append(launcherClearLinuxCommonDeps,
		osprepare.PackageRequirement{BinaryName: "/usr/bin/docker", PackageName: "cloud-control"}),
	"fedora": append(launcherFedoraCommonDeps,
		osprepare.PackageRequirement{BinaryName: "/usr/bin/docker", PackageName: "docker-engine"}),
	"ubuntu": append(launcherUbuntuCommonDeps,
		osprepare.PackageRequirement{BinaryName: "/usr/bin/docker", PackageName: "docker"}),
}

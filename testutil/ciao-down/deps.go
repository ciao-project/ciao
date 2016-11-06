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

var ciaoDevClearDeps = []osprepare.PackageRequirement{
	{BinaryName: "/usr/bin/qemu-system-x86_64", PackageName: "cloud-control"},
	{BinaryName: "/usr/bin/xorriso", PackageName: "cloud-control"},
}

var ciaoDevFedoraDeps = []osprepare.PackageRequirement{
	{BinaryName: "/usr/bin/qemu-system-x86_64", PackageName: "qemu-system-x86"},
	{BinaryName: "/usr/bin/xorriso", PackageName: "xorriso"},
}

var ciaoDevUbuntuDeps = []osprepare.PackageRequirement{
	{BinaryName: "/usr/bin/qemu-system-x86_64", PackageName: "qemu-system-x86"},
	{BinaryName: "/usr/bin/xorriso", PackageName: "xorriso"},
}

var ciaoDevDeps = map[string][]osprepare.PackageRequirement{
	"clearlinux": ciaoDevClearDeps,
	"fedora":     ciaoDevFedoraDeps,
	"ubuntu":     ciaoDevUbuntuDeps,
}

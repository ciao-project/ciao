//
// Copyright © 2016 Intel Corporation
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

package osprepare

import (
	"fmt"
	"os"
	"strings"
)

// Minimal versions supported by ciao
const (
	MinDockerVersion = "1.11.0"
	MinQemuVersion   = "2.5.0"
)

// PackageRequirement contains the BinaryName expected to
// exist on the filesystem once PackageName is installed
// (e.g: { '/usr/bin/qemu-system-x86_64', 'qemu'})
type PackageRequirement struct {
	BinaryName  string
	PackageName string
}

// PackageRequirements type allows to create complex
// mapping to group a set of PackageRequirement to a single
// key.
// (e.g:
//
//	"ubuntu": {
//		{"/usr/bin/docker", "docker"},
//	},
//	"clearlinux": {
//		{"/usr/bin/docker", "containers-basic"},
//	},
// )
type PackageRequirements map[string][]*PackageRequirement

// Required for absolutely core functionality across all Ciao components
var BootstrapRequirements = PackageRequirements{
	"ubuntu": {
		{"/usr/bin/cephfs", "ceph-fs-common"},
		{"/usr/bin/ceph", "ceph-common"},
	},
	"clearlinux": {
		{"/usr/bin/ceph", "storage-cluster"},
	},
}

// CollectPackages returns a list of non-installed packages from
// the PackageRequirements received
func collectPackages(dist distro, reqs *PackageRequirements) []string {
	// For now just support keys like "ubuntu" vs "ubuntu:16.04"
	var pkgsMissing []string
	if reqs == nil {
		return nil
	}

	id := dist.getID()
	if pkgs, success := (*reqs)[id]; success {
		for _, pkg := range pkgs {
			// Have the path existing, skip.
			if pathExists(pkg.BinaryName) {
				continue
			}
			// Mark the package for installation
			pkgsMissing = append(pkgsMissing, pkg.PackageName)
		}
		return pkgsMissing
	}
	return nil
}

// PrepareOsDeps installs all the dependencies defined in
// PackageRequirements in order to run the ciao component
func PrepareOsDeps(reqs *PackageRequirements) {
	distro := getDistro()

	if distro == nil {
		fmt.Fprintf(os.Stderr, "Running on an unsupported distro\n")
		if rel := GetOsRelease(); rel != nil {
			fmt.Fprintf(os.Stderr, "Unsupported distro: %s %s\n", rel.Name, rel.Version)
		} else {
			fmt.Fprintln(os.Stderr, "No os-release found on this host")
		}
		return
	}
	// Nothing requested to install
	if reqs == nil {
		return
	}
	if reqPkgs := collectPackages(distro, reqs); reqPkgs != nil {
		if distro.InstallPackages(reqPkgs) == false {
			fmt.Fprintf(os.Stderr, "Failed to install: %s\n", strings.Join(reqPkgs, ", "))
		}
	}
}

// Bootstrap installs all the core dependencies required to bootstrap the core
// configuration of all Ciao components
func Bootstrap() {
	PrepareOsDeps(&BootstrapRequirements)
}

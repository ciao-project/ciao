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
	"strings"

	"github.com/golang/glog"
)

// OSPLog is a logging interface used by the osprepare package to log various
// interesting pieces of information.  Rather than introduce a dependency
// on a given logging package, osprepare presents this interface that allows
// clients to provide their own logging type or reuse the OSPGlogLogger..
type OSPLog interface {
	// V returns true if the given argument is less than or equal
	// to the implementation's defined verbosity level.
	V(int32) bool

	// Infof writes informational output to the log.  A newline will be
	// added to the output if one is not provided.
	Infof(string, ...interface{})

	// Warningf writes warning output to the log.  A newline will be
	// added to the output if one is not provided.
	Warningf(string, ...interface{})

	// Errorf writes error output to the log.  A newline will be
	// added to the output if one is not provided.
	Errorf(string, ...interface{})
}

type ospNullLogger struct{}

func (l ospNullLogger) V(level int32) bool {
	return false
}

func (l ospNullLogger) Infof(format string, v ...interface{}) {
}

func (l ospNullLogger) Warningf(format string, v ...interface{}) {
}

func (l ospNullLogger) Errorf(format string, v ...interface{}) {
}

// OSPGlogLogger is a type that makes use of glog for the OSPLog interface
type OSPGlogLogger struct{}

// V returns true if the given argument is less than or equal
// to glog's verbosity level.
func (l OSPGlogLogger) V(level int32) bool {
	return bool(glog.V(glog.Level(level)))
}

// Infof writes informational output to glog.
func (l OSPGlogLogger) Infof(format string, v ...interface{}) {
	glog.InfoDepth(2, fmt.Sprintf(format, v...))
}

// Warningf writes warning output to glog.
func (l OSPGlogLogger) Warningf(format string, v ...interface{}) {
	glog.WarningDepth(2, fmt.Sprintf(format, v...))
}

// Errorf writes error output to glog.
func (l OSPGlogLogger) Errorf(format string, v ...interface{}) {
	glog.ErrorDepth(2, fmt.Sprintf(format, v...))
}

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
type PackageRequirements map[string][]PackageRequirement

// BootstrapRequirements lists required dependencies for absolutely core
// functionality across all Ciao components
var BootstrapRequirements = PackageRequirements{
	"ubuntu": {
		{"/usr/bin/cephfs", "ceph-fs-common"},
		{"/usr/bin/ceph", "ceph-common"},
	},
	"fedora": {
		{"/usr/bin/cephfs", "ceph"},
		{"/usr/bin/ceph", "ceph-common"},
	},
	"clearlinux": {
		{"/usr/bin/ceph", "storage-cluster"},
	},
}

// CollectPackages returns a list of non-installed packages from
// the PackageRequirements received
func collectPackages(dist distro, reqs PackageRequirements) []string {
	// For now just support keys like "ubuntu" vs "ubuntu:16.04"
	var pkgsMissing []string
	if reqs == nil {
		return nil
	}

	id := dist.getID()
	if pkgs, success := reqs[id]; success {
		for _, pkg := range pkgs {
			// skip empties
			if pkg.BinaryName == "" || pkg.PackageName == "" {
				continue
			}

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

// InstallDeps installs all the dependencies defined in a component
// specific PackageRequirements in order to enable running the component
func InstallDeps(reqs PackageRequirements, logger OSPLog) {
	if logger == nil {
		logger = ospNullLogger{}
	}
	distro := getDistro()

	if distro == nil {
		logger.Errorf("Running on an unsupported distro")
		if rel := getOSRelease(); rel != nil {
			logger.Errorf("Unsupported distro: %s %s", rel.Name, rel.Version)
		} else {
			logger.Errorf("No os-release found on this host")
		}
		return
	}
	logger.Infof("OS Detected: %s", distro.getID())

	// Nothing requested to install
	if reqs == nil {
		return
	}
	if reqPkgs := collectPackages(distro, reqs); reqPkgs != nil {
		logger.Infof("Missing packages detected: %v", reqPkgs)
		if distro.InstallPackages(reqPkgs, logger) == false {
			logger.Errorf("Failed to install: %s", strings.Join(reqPkgs, ", "))
			return
		}
		logger.Infof("Missing packages installed.")
	}
}

// Bootstrap installs all the core dependencies required to bootstrap the core
// configuration of all Ciao components
func Bootstrap(logger OSPLog) {
	InstallDeps(BootstrapRequirements, logger)
}

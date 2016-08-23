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
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	aptSourcesList  = "/etc/apt/sources.list"
	aptSourcesListD = "/etc/apt/sources.list.d"
)

// aptSourcesFile abstracts a source.list file into
// a list of aptSource structs and the file where
// those definitions are located.
// Each apt source list may have multiple sources
type aptSourcesFile struct {
	Sources []*aptSource
	Path    string
}

// aptSource struct contains all the fields
// in an apt line in /etc/apt/sources.list
// e.g:
// deb|deb-src $origin $distribution $component1 $component2
type aptSource struct {
	DebType      string
	Origin       string
	Distribution string
	Components   []string
}

// readAptSourcesFile reads a sources.list style file
// and return an aptSourcesFile
func readAptSourcesFile(path string) *aptSourcesFile {
	fi, err := os.Open(path)

	if err != nil {
		return nil
	}

	ret := aptSourcesFile{Path: path}

	defer fi.Close()
	sc := bufio.NewScanner(fi)
	for sc.Scan() {
		line := sc.Text()

		line = strings.TrimSpace(line)

		// Skip blanks..
		if len(line) < 1 {
			continue
		}

		// Skip comment lines
		if line[0] == '#' {
			continue
		}

		if rs := newAptSource(line); rs != nil {
			ret.Sources = append(ret.Sources, rs)
		}
		// Could warn here, but that's overkill.
	}
	return &ret
}

// loadAptSources reads the apt source files
// defined by aptSourcesList and aptSourcesListD
// returning a list of aptSourcesFile containing
// all sources defined in the distro
func loadAptSources() []*aptSourcesFile {
	var sources []*aptSourcesFile

	// Closure is only relevant to this function
	addAptSources := func(path string) {
		if r := readAptSourcesFile(path); r != nil {
			sources = append(sources, r)
		}
	}

	if pathExists(aptSourcesList) {
		addAptSources(aptSourcesList)
	}

	// Glob the *.list files now
	if pathExists(aptSourcesListD) {
		tpath := fmt.Sprintf("%s/*.list", aptSourcesListD)
		if files, err := filepath.Glob(tpath); err == nil {
			for _, file := range files {
				addAptSources(file)
			}
		}
	}

	return sources
}

// newAptSource constructs a new apt source from
// the given deb style line
func newAptSource(debLine string) *aptSource {
	fields := strings.Fields(debLine)
	var asource aptSource

	if len(fields) < 3 {
		return nil
	}

	dtype := fields[0]
	if dtype != "deb" && dtype != "deb-src" {
		return nil
	}

	asource.DebType = dtype
	asource.Origin = fields[1]
	asource.Distribution = fields[2]
	if len(fields) > 3 {
		asource.Components = fields[3:]
	}

	return &asource
}

// isUbuntuDockerRepoEnabled iterates through the sources
// to find out if the docker repo is enabled or not.
func isUbuntuDockerRepoEnabled() bool {
	sources := loadAptSources()
	if sources == nil {
		return false
	}
	for _, sourceFile := range sources {
		for _, source := range sourceFile.Sources {
			if strings.Contains(source.Origin, "apt.dockerproject.org") {
				return true
			}
		}
	}
	return false
}

// pathExists is a helper function which handles the
// error and simply return true or false if the given
// path exists
func pathExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

type distro interface {
	// InstallPackages should implement the installation
	// of packages using distro specific methods for
	// the given target list of items to install
	InstallPackages(packages []string) bool

	// getID should return a string specifying
	// the distribution ID (e.g: "clearlinux")
	getID() string
}

// getDistro will return a distro based on what
// is read from GetOsRelease
func getDistro() distro {
	osRelease := getOSRelease()

	if osRelease == nil {
		return nil
	}

	if strings.HasPrefix(osRelease.ID, "clear-linux") {
		return &clearLinuxDistro{}
	} else if strings.Contains(osRelease.ID, "ubuntu") {
		// Store the Ubuntu codename, i.e. "xenial'
		return &ubuntuDistro{CodeName: osRelease.GetValue("UBUNTU_CODENAME")}
	} else if strings.Contains(osRelease.ID, "fedora") {
		return &fedoraDistro{}
	}
	return nil
}

// os-release clear-linux*
type clearLinuxDistro struct {
}

func (d *clearLinuxDistro) getID() string {
	return "clearlinux"
}

// Correctly split and format the command, using sudo if appropriate, as a
// common mechanism for the various package install functions.
func sudoFormatCommand(command string, packages []string) bool {
	toInstall := strings.Join(packages[0:], " ")

	var executable string
	var args string
	splits := strings.Split(command, " ")

	if syscall.Geteuid() == 0 {
		executable = splits[0]
		args = fmt.Sprintf(strings.Join(splits[1:], " "), toInstall)
	} else {
		executable = "sudo"
		args = fmt.Sprintf(command, toInstall)
	}

	c := exec.Command(executable, strings.Split(args, " ")...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to run command: %s", err)
		return false
	}
	return true
}

func (d *clearLinuxDistro) InstallPackages(packages []string) bool {
	return sudoFormatCommand("swupd bundle-add %s", packages)
}

// os-release *ubuntu*
type ubuntuDistro struct {
	CodeName string
}

func (d *ubuntuDistro) getID() string {
	return "ubuntu"
}

func (d *ubuntuDistro) InstallPackages(packages []string) bool {
	return sudoFormatCommand("apt-get --yes --force-yes install %s", packages)
}

// Fedora
type fedoraDistro struct {
}

func (d *fedoraDistro) getID() string {
	return "fedora"
}

// Use dnf to install on Fedora
func (d *fedoraDistro) InstallPackages(packages []string) bool {
	return sudoFormatCommand("dnf install -y %s", packages)
}

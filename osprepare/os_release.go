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
	"os"
	"strings"
)

type OsRelease struct {
	Name       string
	ID         string
	PrettyName string
	Version    string
	VersionID  string
	mapping    map[string]string
}

// Parse the given path and attempt to return a valid
// OsRelease for it
func ParseReleaseFile(path string) *OsRelease {
	fi, err := os.Open(path)
	var os_rel OsRelease
	os_rel.mapping = make(map[string]string)

	if err != nil {
		return nil
	}
	defer fi.Close()
	sc := bufio.NewScanner(fi)
	for sc.Scan() {
		line := sc.Text()

		spl := strings.Split(line, "=")
		if len(spl) < 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(spl[0]))
		value := strings.TrimSpace(strings.Join(spl[1:], "="))

		value = strings.Replace(value, "\"", "", -1)
		value = strings.Replace(value, "'", "", -1)

		if key == "name" {
			os_rel.Name = value
		} else if key == "id" {
			os_rel.ID = value
		} else if key == "pretty_name" {
			os_rel.PrettyName = value
		} else if key == "version" {
			os_rel.Version = value
		} else if key == "version_id" {
			os_rel.VersionID = value
		}

		// Store it for use by Distro
		os_rel.mapping[key] = value
	}
	return &os_rel
}

// Try all known paths to get the right OsRelease instance
func GetOsRelease() *OsRelease {
	paths := []string{
		"/etc/os-release",
		"/usr/lib/os-release",
		"/usr/lib64/os-release",
	}

	for _, item := range paths {
		if os_rel := ParseReleaseFile(item); os_rel != nil {
			return os_rel
		}
	}
	return nil
}

func (o *OsRelease) GetValue(key string) string {
	if val, succ := o.mapping[strings.ToLower(key)]; succ {
		return val
	}
	return ""
}

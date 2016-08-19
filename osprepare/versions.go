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
	"os/exec"
	"strconv"
	"strings"
)

func get_command_output(command string) string {
	splits := strings.Split(command, " ")
	c := exec.Command(splits[0], splits[1:]...)
	c.Env = os.Environ()
	// Force C locale
	c.Env = append(c.Env, "LC_ALL=C")
	c.Env = append(c.Env, "LANG=C")
	c.Stderr = os.Stderr

	if out, err := c.Output(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run %s: %s\n", splits[0], err)
		return ""
	} else {
		return string(out)
	}
}

func GetDockerVersion() string {
	ret := get_command_output("docker --version")
	var version string
	if n, _ := fmt.Sscanf(ret, "Docker version %s, build", &version); n != 1 {
		return ""
	} else {
		if strings.HasSuffix(version, ",") {
			return string(version[0 : len(version)-1])
		}
		return version
	}
}

func GetQemuVersion() string {
	ret := get_command_output("qemu-system-x86_64 --version")
	var version string
	if n, _ := fmt.Sscanf(ret, "QEMU emulator version %s, Copyright (c)", &version); n != 1 {
		return ""
	} else {
		if strings.HasSuffix(version, ",") {
			return string(version[0 : len(version)-1])
		}
		return version
	}
}

// Determine if the given current version is less than the test version
// Note: Can only compare equal version schemas (i.e. same level of dots)
func VersionLessThan(current_version string, test_version string) bool {
	cur_splits := strings.Split(current_version, ".")
	test_splits := strings.Split(test_version, ".")

	max_range := len(cur_splits)
	if l2 := len(test_splits); l2 < max_range {
		max_range = l2
	}

	cur_isplits := make([]int, max_range)
	cur_tsplits := make([]int, max_range)

	for i := 0; i < max_range; i++ {
		cur_isplits[i], _ = strconv.Atoi(cur_splits[i])
		cur_tsplits[i], _ = strconv.Atoi(test_splits[i])
	}

	for i := 0; i < max_range; i++ {
		if i == 0 {
			if cur_isplits[i] < cur_tsplits[i] {
				return true
			}
		} else {
			match := true
			for j := 0; j < i; j++ {
				if cur_isplits[j] != cur_tsplits[j] {
					match = false
					break
				}
			}
			if match && cur_isplits[i] < cur_tsplits[i] {
				return true
			}
		}
	}
	return false
}

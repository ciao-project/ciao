//
// Copyright (c) 2017 Intel Corporation
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

import (
	"io/ioutil"
	"path"
	"testing"

	"os"

	"github.com/stretchr/testify/assert"
)

const xenialWorkloadSpecNoVM = `
base_image_url: ` + guestDownloadURL + `
base_image_name: ` + guestImageFriendlyName + `
`

const sampleVMSpec = `
mem_gib: 3
cpus: 2
ports:
- host: 10022
  guest: 22
mounts: []
`

const xenialWorkloadSpec = `
base_image_url: ` + guestDownloadURL + `
base_image_name: ` + guestImageFriendlyName + `
vm:
  mem_gib: 3
  cpus: 2
  ports:
  - host: 10022
    guest: 22
  mounts: []
`

var mockVMSpec = VMSpec{
	MemGiB:       3,
	CPUs:         2,
	PortMappings: []portMapping{{Host: 10022, Guest: 22}},
	Mounts:       []mount{},
}

const sampleCloudInit = `
`

const sampleWorkload3Docs = "---\n" + xenialWorkloadSpecNoVM + "...\n---\n" + sampleVMSpec + "...\n---\n" + sampleCloudInit + "...\n"
const sampleWorkload = "---\n" + xenialWorkloadSpec + "...\n---\n" + sampleCloudInit + "...\n"

func createMockWorkSpaceWithWorkload(t *testing.T, workload string) *workspace {
	ciaoDir, err := ioutil.TempDir("", "ciao-down-tests-")
	assert.Nil(t, err)

	instanceDir := path.Join(ciaoDir, "foo")
	err = os.Mkdir(instanceDir, 0750)
	assert.Nil(t, err)

	ws := &workspace{
		ciaoDir:     ciaoDir,
		instanceDir: instanceDir,
	}

	workloadFile := path.Join(ws.instanceDir, "state.yaml")
	err = ioutil.WriteFile(workloadFile, []byte(workload), 0640)
	assert.Nil(t, err)

	return ws
}

func createMockWorkspace(t *testing.T) *workspace {
	return createMockWorkSpaceWithWorkload(t, sampleWorkload)
}

func cleanupMockWorkspace(t *testing.T, ws *workspace) {
	err := os.RemoveAll(ws.ciaoDir)
	assert.Nil(t, err)
}

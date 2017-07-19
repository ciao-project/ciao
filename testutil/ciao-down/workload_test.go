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
	"testing"

	"github.com/stretchr/testify/assert"
)

const document1 = `# Just a simple document
foo: "a string"
bar: True
`

const document2 = `# A list
- foo
- bar
`

const twoDocuments = "---\n" + document1 + "...\n----\n" + document2 + "...\n"

func TestSplitYaml(t *testing.T) {
	tests := []struct {
		content   string
		documents [][]byte
	}{
		{"", [][]byte{}},
		{document1, [][]byte{[]byte(document1)}},
		{twoDocuments, [][]byte{[]byte(document1), []byte(document2)}},
	}

	for i := range tests {
		test := &tests[i]

		documents := splitYaml([]byte(test.content))
		assert.Equal(t, test.documents, documents)
	}
}

func TestRestoreWorkload(t *testing.T) {
	tests := []struct {
		checkSpec bool // Should we check the workload.spec content?
		workload  string
	}{
		// 1 document: per-VM data (legacy)
		{workload: sampleVMSpec},
		// 3 documents: spec, per-VM data, cloud init file
		{workload: sampleWorkload, checkSpec: true},
	}

	for i := range tests {
		test := &tests[i]

		ws := createMockWorkSpaceWithWorkload(t, test.workload)

		workload, err := restoreWorkload(ws)
		assert.Nil(t, err)
		assert.Equal(t, mockVMSpec, workload.spec.VM)
		if test.checkSpec {
			assert.Equal(t, guestDownloadURL, workload.spec.BaseImageURL)
			assert.Equal(t, guestImageFriendlyName, workload.spec.BaseImageName)
			assert.Equal(t, defaultHostname, workload.spec.Hostname)
		}

		cleanupMockWorkspace(t, ws)
	}
}

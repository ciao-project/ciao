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
	"bufio"
	"bytes"
	"fmt"
	"go/build"
	"io/ioutil"
	"path/filepath"
	"regexp"
)

const ciaoDownPkg = "github.com/01org/ciao/testutil/ciao-down"

var indentedRegexp *regexp.Regexp

func init() {
	indentedRegexp = regexp.MustCompile("\\s+.*")
}

type workload struct {
	insSpec  instanceSpec
	insData  instance
	userData string
}

func loadWorkload(ws *workspace, vmType string) ([]byte, error) {
	localPath := filepath.Join(ws.Home, ".ciao-down", "workloads",
		fmt.Sprintf("%s.yaml", vmType))
	wkld, err := ioutil.ReadFile(localPath)
	if err == nil {
		return wkld, nil
	}

	p, err := build.Default.Import(ciaoDownPkg, "", build.FindOnly)
	if err != nil {
		return nil, fmt.Errorf("Unable to locate ciao-down workload directory: %v", err)
	}
	workloadPath := filepath.Join(p.Dir, "workloads", fmt.Sprintf("%s.yaml", vmType))
	wkld, err = ioutil.ReadFile(workloadPath)
	if err != nil {
		return nil, fmt.Errorf("Unable to load workload %s", workloadPath)
	}

	return wkld, nil
}

func createWorkload(ws *workspace, vmType string) (*workload, error) {
	data, err := loadWorkload(ws, vmType)
	if err != nil {
		return nil, err
	}

	var wkld workload
	var insSpec, insData, userData string
	docs := splitYaml(data)
	if len(docs) == 1 {
		userData = string(docs[0])
	} else if len(docs) >= 3 {
		insSpec = string(docs[0])
		insData = string(docs[1])
		userData = string(docs[2])
	} else {
		return nil, fmt.Errorf("Invalid workload")
	}

	err = wkld.insSpec.unmarshallWithTemplate(ws, insSpec)
	if err != nil {
		return nil, err
	}

	err = wkld.insData.unmarshallWithTemplate(ws, insData)
	if err != nil {
		return nil, err
	}

	wkld.userData = userData

	return &wkld, nil
}

func findDocument(lines [][]byte) ([]byte, int) {
	var realStart int
	var realEnd int
	docStartFound := false
	docEndFound := false

	start := len(lines) - 1
	line := lines[start]
	if bytes.HasPrefix(line, []byte("...")) {
		docEndFound = true
		realEnd = start
		start--
	}

	for ; start >= 0; start-- {
		line := lines[start]
		if bytes.HasPrefix(line, []byte("---")) {
			docStartFound = true
			break
		}
		if bytes.HasPrefix(line, []byte("...")) {
			start++
			break
		}
	}

	if docStartFound {
		realStart = start + 1
		for start = start - 1; start >= 0; start-- {
			line := lines[start]
			if !bytes.HasPrefix(line, []byte{'%'}) {
				break
			}
		}
		start++
	} else {
		if start < 0 {
			start = 0
		}
		realStart = start
	}

	if !docEndFound {
		realEnd = len(lines)
	}

	var buf bytes.Buffer
	for _, line := range lines[realStart:realEnd] {
		_, _ = buf.Write(line)
		_ = buf.WriteByte('\n')
	}

	return buf.Bytes(), start
}

func splitYaml(data []byte) [][]byte {
	lines := make([][]byte, 0, 256)
	docs := make([][]byte, 0, 3)

	reader := bytes.NewReader(data)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Bytes()
		lineC := make([]byte, len(line))
		_ = copy(lineC, line)
		lines = append(lines, lineC)
	}

	endOfNextDoc := len(lines)
	for endOfNextDoc > 0 {
		var doc []byte
		doc, endOfNextDoc = findDocument(lines[:endOfNextDoc])
		docs = append([][]byte{doc}, docs...)
	}

	return docs
}

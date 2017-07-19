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
	"path"
	"path/filepath"
	"regexp"

	yaml "gopkg.in/yaml.v2"
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

func (wkld *workload) save(ws *workspace) error {
	var buf bytes.Buffer

	_, _ = buf.WriteString("---\n")
	data, err := yaml.Marshal(wkld.insSpec)
	if err != nil {
		return fmt.Errorf("Unable to marshal instance specification : %v", err)
	}
	_, _ = buf.Write(data)
	_, _ = buf.WriteString("...\n")

	_, _ = buf.WriteString("---\n")
	data, err = yaml.Marshal(wkld.insData)
	if err != nil {
		return fmt.Errorf("Unable to marshal instance specification : %v", err)
	}
	_, _ = buf.Write(data)
	_, _ = buf.WriteString("...\n")

	_, _ = buf.WriteString("---\n")
	_, _ = buf.WriteString(wkld.userData)
	_, _ = buf.WriteString("...\n")

	err = ioutil.WriteFile(path.Join(ws.instanceDir, "state.yaml"),
		buf.Bytes(), 0600)
	if err != nil {
		return fmt.Errorf("Unable to write instance state : %v", err)
	}

	return nil
}

func loadWorkloadData(ws *workspace, workloadName string) ([]byte, error) {
	localPath := filepath.Join(ws.Home, ".ciao-down", "workloads",
		fmt.Sprintf("%s.yaml", workloadName))
	wkld, err := ioutil.ReadFile(localPath)
	if err == nil {
		return wkld, nil
	}

	p, err := build.Default.Import(ciaoDownPkg, "", build.FindOnly)
	if err != nil {
		return nil, fmt.Errorf("Unable to locate ciao-down workload directory: %v", err)
	}
	workloadPath := filepath.Join(p.Dir, "workloads", fmt.Sprintf("%s.yaml", workloadName))
	wkld, err = ioutil.ReadFile(workloadPath)
	if err != nil {
		return nil, fmt.Errorf("Unable to load workload %s", workloadPath)
	}

	return wkld, nil
}

func unmarshalWorkload(ws *workspace, wkld *workload, insSpec, insData,
	userData string) error {
	err := wkld.insSpec.unmarshalWithTemplate(ws, insSpec)
	if err != nil {
		return err
	}

	err = wkld.insData.unmarshalWithTemplate(ws, insData)
	if err != nil {
		return err
	}

	wkld.userData = userData

	return nil
}

func createWorkload(ws *workspace, workloadName string) (*workload, error) {
	data, err := loadWorkloadData(ws, workloadName)
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

	err = unmarshalWorkload(ws, &wkld, insSpec, insData, userData)
	if err != nil {
		return nil, err
	}
	if wkld.insSpec.WorkloadName == "" {
		wkld.insSpec.WorkloadName = workloadName
	}
	return &wkld, nil
}

func restoreWorkload(ws *workspace) (*workload, error) {
	var wkld workload
	data, err := ioutil.ReadFile(path.Join(ws.instanceDir, "state.yaml"))
	if err != nil {
		if err = wkld.insData.loadLegacyInstance(ws); err != nil {
			return nil, err
		}
		return &wkld, nil
	}

	docs := splitYaml(data)
	if len(docs) == 1 {
		// Older versions of ciao-down just stored the instance
		// data and not the entire workload.
		if err = wkld.insData.unmarshalWithTemplate(ws, string(docs[0])); err != nil {
			return nil, err
		}
		return &wkld, nil
	} else if len(docs) < 3 {
		return nil, fmt.Errorf("Invalid workload")
	}

	err = unmarshalWorkload(ws, &wkld, string(docs[0]), string(docs[1]), string(docs[2]))
	return &wkld, err
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

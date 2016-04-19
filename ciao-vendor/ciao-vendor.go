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

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
)

type repoInfo struct {
	URL     string
	version string
}

var repos = map[string]repoInfo{
	"github.com/docker/distribution": {"https://github.com/docker/distribution.git", "v2.4.0"},
}

var vendorTmpPath = "/tmp/ciao-vendor"

func isStandardPackage(name string) bool {
	cmd := exec.Command("go", "list", "-f", "{{.Standard}}", name)
	output, err := cmd.Output()
	if err != nil {
		return true
	}
	return bytes.HasPrefix(output, []byte{'t', 'r', 'u', 'e'})
}

func calcDeps() ([]string, error) {
	deps := make(map[string]struct{})

	listTemplate := `
{{- range .Deps -}}
{{.}}
{{end -}}
`
	var output bytes.Buffer
	cmd := exec.Command("go", "list", "-f", listTemplate, "./...")
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("go list failed: %v\n", err)
	}

	scanner := bufio.NewScanner(&output)
	for scanner.Scan() {
		deps[scanner.Text()] = struct{}{}
	}

	ch := make(chan string)
	for pkg := range deps {
		go func(pkg string) {
			if !isStandardPackage(pkg) {
				ch <- pkg
			} else {
				ch <- ""
			}
		}(pkg)
	}

	depsAr := make([]string, 0, len(deps))
	for i := 0; i < cap(depsAr); i++ {
		pkg := <-ch
		if pkg != "" {
			depsAr = append(depsAr, pkg)
		}
	}

	sort.Strings(depsAr)
	return depsAr, nil
}

func checkWD() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("Unable to determine cwd: %v", err)
	}
	gopath, _ := os.LookupEnv("GOPATH")
	if gopath == "" {
		return "", fmt.Errorf("GOPATH is not set")
	}

	pths := strings.Split(gopath, ":")

	for _, p := range pths {
		if path.Join(p, "src/github.com/01org/ciao") == cwd {
			return cwd, nil
		}
	}

	return "", fmt.Errorf("ciao-vendor must be run from $GOPATH/src/01org/ciao")
}

func cloneRepos() error {
	err := os.MkdirAll(vendorTmpPath, 0755)
	if err != nil {
		return fmt.Errorf("Unable to create %s : %v", vendorTmpPath, err)
	}

	errCh := make(chan error)

	for _, r := range repos {
		go func(URL string) {
			cmd := exec.Command("git", "clone", URL)
			cmd.Dir = vendorTmpPath
			err := cmd.Run()
			if err != nil {
				errCh <- fmt.Errorf("git clone %s failed : %v", URL, err)
			} else {
				errCh <- nil
			}
		}(r.URL)
	}

	for _ = range repos {
		rcvErr := <-errCh
		if err == nil && rcvErr != nil {
			err = rcvErr
		}
	}

	return err
}

func copyRepos(cwd string, subPackages map[string][]string) error {
	errCh := make(chan error)
	for k, r := range repos {
		packages, ok := subPackages[k]
		if !ok {
			fmt.Printf("Warning: No packages found for: %s\n", r.URL)
			continue
		}
		go func(k string, packages []string) {
			args := []string{"archive", repos[k].version}
			args = append(args, packages...)
			cmd1 := exec.Command("git", args...)
			index := strings.LastIndex(k, "/")
			cmd1.Dir = path.Join(vendorTmpPath, k[index+1:])
			fmt.Println(cmd1.Dir)
			fmt.Println(path.Join(cwd, k))
			os.MkdirAll(path.Join(cwd, "vendor", k), 0755)
			cmd2 := exec.Command("tar", "-x", "-C", path.Join(cwd, "vendor", k))
			fmt.Println(args)
			pipe, err := cmd1.StdoutPipe()
			if err != nil {
				errCh <- fmt.Errorf("Unable to retrieve pipe for git command %v: %v", args, err)
				return
			}
			defer func() {
				_ = pipe.Close()
			}()
			cmd2.Stdin = pipe
			err = cmd1.Start()
			if err != nil {
				errCh <- fmt.Errorf("Unable to start git command %v: %v", args, err)
				return
			}
			err = cmd2.Run()
			if err != nil {
				errCh <- fmt.Errorf("Unable to run tar command %v", err)
				return
			}
			errCh <- nil
		}(k, packages)
	}

	var err error
	for _ = range repos {
		rcvErr := <-errCh
		if err == nil && rcvErr != nil {
			err = rcvErr
		}
	}

	return err
}

func vendor(cwd string) error {
	vendorPath := path.Join(cwd, "vendor")

	err := os.RemoveAll(vendorPath)
	if err != nil {
		return err
	}

	err = os.RemoveAll(vendorTmpPath)
	if err != nil {
		return err
	}

	defer func() {
		_ = os.RemoveAll(vendorTmpPath)
	}()

	err = cloneRepos()
	if err != nil {
		return err
	}

	deps, err := calcDeps()
	if err != nil {
		return err
	}

	subPackages := make(map[string][]string)
	for _, d := range deps {
		for k := range repos {
			if !strings.HasPrefix(d, k) {
				continue
			}

			packages := subPackages[k]
			if len(packages) == 1 && packages[0] == "" {
				continue
			}

			subPackage := d[len(k):]
			if subPackage == "" {
				subPackages[k] = []string{""}
			} else if subPackage[0] == '/' {
				subPackages[k] = append(packages, subPackage[1:])
			} else {
				fmt.Printf("Warning: unvendored package: %s\n", d)
			}
		}
	}

	return copyRepos(cwd, subPackages)
}

func main() {

	cwd, err := checkWD()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = vendor(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

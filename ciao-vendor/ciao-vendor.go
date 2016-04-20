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
	"text/tabwriter"
)

type repoInfo struct {
	URL     string
	version string
}

type packageDeps struct {
	p    string
	deps []string
}

var repos = map[string]repoInfo{
	"github.com/docker/distribution":    {"https://github.com/docker/distribution.git", "v2.4.0"},
	"gopkg.in/yaml.v2":                  {"https://gopkg.in/yaml.v2", "a83829b"},
	"github.com/Sirupsen/logrus":        {"https://github.com/Sirupsen/logrus.git", "v0.9.0"},
	"github.com/boltdb/bolt":            {"https://github.com/boltdb/bolt.git", "144418e"},
	"github.com/coreos/go-iptables":     {"https://github.com/coreos/go-iptables.git", "fbb7337"},
	"github.com/davecgh/go-spew":        {"https://github.com/davecgh/go-spew.git", "5215b55"},
	"github.com/docker/docker":          {"https://github.com/docker/docker.git", "v1.10.3"},
	"github.com/docker/engine-api":      {"https://github.com/docker/engine-api.git", "v0.3.3"},
	"github.com/docker/go-connections":  {"https://github.com/docker/go-connections.git", "5b7154b"},
	"github.com/docker/go-units":        {"https://github.com/docker/go-units.git", "651fc22"},
	"github.com/docker/libnetwork":      {"https://github.com/docker/libnetwork.git", "dbb0722"},
	"github.com/golang/glog":            {"https://github.com/golang/glog.git", "23def4e"},
	"github.com/gorilla/context":        {"https://github.com/gorilla/context.git", "1ea2538"},
	"github.com/gorilla/mux":            {"https://github.com/gorilla/mux.git", "0eeaf83"},
	"github.com/mattn/go-sqlite3":       {"https://github.com/mattn/go-sqlite3.git", "467f50b"},
	"github.com/mitchellh/mapstructure": {"https://github.com/mitchellh/mapstructure.git", "d2dd026"},
	"github.com/opencontainers/runc":    {"https://github.com/opencontainers/runc.git", "v0.1.0"},
	"github.com/rackspace/gophercloud":  {"https://github.com/rackspace/gophercloud.git", "c54bbac"},
	"github.com/tylerb/graceful":        {"https://github.com/tylerb/graceful.git", "9a3d423"},
	"github.com/vishvananda/netlink":    {"https://github.com/vishvananda/netlink.git", "a632d6d"},
	"golang.org/x/net":                  {"https://go.googlesource.com/net", "origin/release-branch.go1.6"},
}

var vendorTmpPath = "/tmp/ciao-vendor"
var listTemplate = `
{{- range .Deps -}}
{{.}}
{{end -}}
`

func isStandardPackage(name string) bool {
	cmd := exec.Command("go", "list", "-f", "{{.Standard}}", name)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return bytes.HasPrefix(output, []byte{'t', 'r', 'u', 'e'})
}

func calcDeps(projectRoot string, packages []string) ([]string, error) {
	deps := make(map[string]struct{})
	args := []string{"list", "-f", listTemplate}
	args = append(args, packages...)
	var output bytes.Buffer
	cmd := exec.Command("go", args...)
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
			if !strings.HasPrefix(pkg, projectRoot) && !isStandardPackage(pkg) {
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

func checkWD() (string, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("Unable to determine cwd: %v", err)
	}
	gopath, _ := os.LookupEnv("GOPATH")
	if gopath == "" {
		return "", "", fmt.Errorf("GOPATH is not set")
	}

	pths := strings.Split(gopath, ":")

	for _, p := range pths {
		if path.Join(p, "src/github.com/01org/ciao") == cwd {
			return cwd, gopath, nil
		}
	}

	return "", "", fmt.Errorf("ciao-vendor must be run from $GOPATH/src/01org/ciao")
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

func baseCloneDir(URL string) string {
	index := strings.LastIndex(URL, "/")
	if index == -1 {
		return URL
	}

	dir := URL[index+1:]
	index = strings.LastIndex(dir, ".git")
	if index == -1 {
		return dir
	}
	return dir[:index]
}

func copyRepos(cwd string, subPackages map[string][]string) error {
	errCh := make(chan error)
	for k, r := range repos {
		go func(k string, URL string) {
			packages, ok := subPackages[k]
			if !ok {
				fmt.Printf("Warning: No packages found for: %s\n", URL)
				errCh <- nil
				return
			}

			args := []string{"archive", repos[k].version}
			args = append(args, packages...)
			cmd1 := exec.Command("git", args...)
			cmd1.Dir = path.Join(vendorTmpPath, baseCloneDir(URL))
			os.MkdirAll(path.Join(cwd, "vendor", k), 0755)
			cmd2 := exec.Command("tar", "-x", "-C", path.Join(cwd, "vendor", k))
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
		}(k, r.URL)
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

func vendor(cwd, projectRoot string) error {
	vendorPath := path.Join(cwd, "vendor")

	err := os.RemoveAll(vendorPath)
	if err != nil {
		return err
	}

	err = os.RemoveAll(vendorTmpPath)
	if err != nil {
		return err
	}
	/*
		defer func() {
			_ = os.RemoveAll(vendorTmpPath)
		}()
	*/
	fmt.Println("Cloning Repos")
	err = cloneRepos()
	if err != nil {
		return err
	}

	fmt.Println("Calculating Dependencies")
	deps, err := calcDeps(projectRoot, []string{"./..."})
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

	fmt.Println("Populating vendor folder")

	return copyRepos(cwd, subPackages)
}

func usedBy(name string, packages []string, depsMap map[string][]string) string {
	var users bytes.Buffer

	for _, p := range packages {
		if p == name {
			continue
		}

		deps := depsMap[p]
		for _, d := range deps {
			if d == name {
				users.WriteString(" ")
				users.WriteString(p)
				break
			}
		}
	}

	// BUG(markus): We don't report when a depdenency is used by ciao if
	// it is also used by a depdenency

	if users.Len() == 0 {
		return "ciao"
	}

	return users.String()[1:]
}

func depsByPackage(packages []string) map[string][]string {
	depsMap := make(map[string][]string)
	depsCh := make(chan packageDeps)
	for _, p := range packages {
		go func(p string) {
			var output bytes.Buffer
			cmd := exec.Command("go", "list", "-f", listTemplate, p)
			cmd.Stdout = &output
			err := cmd.Run()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Unable to call get list on %s : %v", p, err)
				depsCh <- packageDeps{p: p}
				return
			}
			scanner := bufio.NewScanner(&output)
			deps := make([]string, 0, 32)
			for scanner.Scan() {
				deps = append(deps, scanner.Text())
			}
			depsCh <- packageDeps{p, deps}
		}(p)
	}

	for _ = range packages {
		pkgDeps := <-depsCh
		depsMap[pkgDeps.p] = pkgDeps.deps
	}

	return depsMap
}

func computeClients(packages []string) map[string]string {
	depsMap := depsByPackage(packages)
	clientMap := make(map[string]string)
	for _, p := range packages {
		clientMap[p] = usedBy(p, packages, depsMap)
	}
	return clientMap
}

func check(cwd, projectRoot string) error {
	var output bytes.Buffer
	cmd := exec.Command("go", "list", "./...")
	cmd.Stdout = &output
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("go list failed: %v\n", err)
	}

	packages := make([]string, 0, 128)
	scanner := bufio.NewScanner(&output)

	vendorDir := path.Join(projectRoot, "vendor")

	for scanner.Scan() {
		pkg := scanner.Text()
		if !strings.HasPrefix(pkg, vendorDir) {
			packages = append(packages, pkg)
		}
	}

	deps, err := calcDeps(projectRoot, packages)
	if err != nil {
		return err
	}

	missing := make([]string, 0, 128)

depLoop:
	for _, d := range deps {
		for k := range repos {
			if strings.HasPrefix(d, k) {
				continue depLoop
			}
		}
		missing = append(missing, d)
	}

	if len(missing) == 0 {
		fmt.Println("All dependencies are vendored")
		return nil
	}

	clientMap := computeClients(deps)

	fmt.Println("NON-VENDORED DEPENDENCIES FOUND\n")
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, '\t', 0)
	fmt.Fprintln(w, "Package\tUsed By")
	for _, d := range missing {
		fmt.Fprintf(w, "%s\t%s\n", d, clientMap[d])
	}
	w.Flush()

	return nil
}

func main() {
	if len(os.Args) != 2 || (os.Args[1] != "vendor" && os.Args[1] != "check") {
		fmt.Fprintln(os.Stderr, "Usage: ciao-vendor vendor|check")
		os.Exit(1)
	}

	cwd, gopath, err := checkWD()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	sourceRoot := path.Join(gopath, "src")
	if len(cwd) < len(sourceRoot)+1 {
		fmt.Fprintln(os.Stderr, "Could not determine project root")
		os.Exit(1)
	}
	projectRoot := cwd[len(sourceRoot)+1:]

	if os.Args[1] == "check" {
		err = check(cwd, projectRoot)
	} else {
		err = vendor(cwd, projectRoot)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

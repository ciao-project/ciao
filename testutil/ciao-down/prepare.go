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
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/01org/ciao/osprepare"
)

type logger struct{}

func (l logger) V(int32) bool {
	return false
}

func (l logger) Infof(s string, args ...interface{}) {
	out := fmt.Sprintf(s, args...)
	fmt.Print(out)
	if !strings.HasSuffix(out, "\n") {
		fmt.Println()
	}
}

func (l logger) Warningf(s string, args ...interface{}) {
	l.Infof(s, args)
}

func (l logger) Errorf(s string, args ...interface{}) {
	l.Infof(s, args)
}

type workspace struct {
	GoPath         string
	Home           string
	HTTPProxy      string
	HTTPSProxy     string
	NoProxy        string
	User           string
	PublicKey      string
	HTTPServerPort int
	GitUserName    string
	GitEmail       string
	ciaoDir        string
	instanceDir    string
}

func installDeps(ctx context.Context) {
	osprepare.InstallDeps(ctx, ciaoDevDeps, logger{})
}

func prepareEnv(ctx context.Context) (*workspace, error) {
	ws := &workspace{HTTPServerPort: 8080}

	ws.GoPath = os.Getenv("GOPATH")
	if ws.GoPath == "" {
		return nil, fmt.Errorf("GOPATH is not defined")
	}
	ws.Home = os.Getenv("HOME")
	if ws.Home == "" {
		return nil, fmt.Errorf("HOME is not defined")
	}
	ws.User = os.Getenv("USER")
	if ws.User == "" {
		return nil, fmt.Errorf("USER is not defined")
	}

	ws.HTTPProxy = os.Getenv("HTTP_PROXY")
	if ws.HTTPProxy == "" {
		ws.HTTPProxy = os.Getenv("http_proxy")
	}

	ws.HTTPSProxy = os.Getenv("HTTPS_PROXY")
	if ws.HTTPSProxy == "" {
		ws.HTTPSProxy = os.Getenv("https_proxy")
	}

	ws.NoProxy = os.Getenv("no_proxy")

	pkPath := path.Join(ws.Home, ".ssh/id_rsa.pub")
	var err error
	publicKey, err := ioutil.ReadFile(pkPath)
	if err != nil {
		return nil, fmt.Errorf("Unable to read public ssh key: %v", err)
	}
	ws.PublicKey = string(publicKey)

	ws.ciaoDir = path.Join(ws.Home, ".ciao-down")
	ws.instanceDir = path.Join(ws.ciaoDir, "instance")

	data, err := exec.Command("git", "config", "--global", "user.name").Output()
	if err == nil {
		ws.GitUserName = strings.TrimSpace(string(data))
	}

	data, err = exec.Command("git", "config", "--global", "user.email").Output()
	if err == nil {
		ws.GitEmail = strings.TrimSpace(string(data))
	}

	return ws, nil
}

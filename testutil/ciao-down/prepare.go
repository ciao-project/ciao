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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"

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

// TODO: Code copied from launcher.  Needs to be moved to qemu

func createCloudInitISO(ctx context.Context, instanceDir string, userData, metaData []byte) error {
	configDrivePath := path.Join(instanceDir, "clr-cloud-init")
	dataDirPath := path.Join(configDrivePath, "openstack", "latest")
	metaDataPath := path.Join(dataDirPath, "meta_data.json")
	userDataPath := path.Join(dataDirPath, "user_data")
	isoPath := path.Join(instanceDir, "config.iso")

	defer func() {
		_ = os.RemoveAll(configDrivePath)
	}()

	err := os.MkdirAll(dataDirPath, 0755)
	if err != nil {
		return fmt.Errorf("Unable to create config drive directory %s", dataDirPath)
	}

	err = ioutil.WriteFile(metaDataPath, metaData, 0644)
	if err != nil {
		return fmt.Errorf("Unable to create %s", metaDataPath)
	}

	err = ioutil.WriteFile(userDataPath, userData, 0644)
	if err != nil {
		return fmt.Errorf("Unable to create %s", userDataPath)
	}

	cmd := exec.CommandContext(ctx, "xorriso", "-as", "mkisofs", "-R", "-V", "config-2",
		"-o", isoPath, configDrivePath)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Unable to create cloudinit iso image %v", err)
	}

	return nil
}

func buildISOImage(ctx context.Context, instanceDir string, ws *workspace, debug bool) error {
	udt := template.Must(template.New("user-data").Parse(userDataTemplate))
	var udBuf bytes.Buffer
	err := udt.Execute(&udBuf, ws)
	if err != nil {
		return fmt.Errorf("Unable to execute user data template : %v", err)
	}

	mdt := template.Must(template.New("meta-data").Parse(metaDataTemplate))

	var mdBuf bytes.Buffer
	err = mdt.Execute(&mdBuf, ws)
	if err != nil {
		return fmt.Errorf("Unable to execute user data template : %v", err)
	}

	if debug {
		fmt.Println(string(udBuf.Bytes()))
		fmt.Println(string(mdBuf.Bytes()))
	}

	return createCloudInitISO(ctx, instanceDir, udBuf.Bytes(), mdBuf.Bytes())
}

// TODO: Code copied from launcher.  Needs to be moved to qemu

func createRootfs(ctx context.Context, backingImage, instanceDir string) error {
	vmImage := path.Join(instanceDir, "image.qcow2")
	if _, err := os.Stat(vmImage); err == nil {
		_ = os.Remove(vmImage)
	}
	params := make([]string, 0, 32)
	params = append(params, "create", "-f", "qcow2", "-o", "backing_file="+backingImage,
		vmImage, "60000M")
	return exec.CommandContext(ctx, "qemu-img", params...).Run()
}

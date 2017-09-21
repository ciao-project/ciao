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
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ciao-project/ciao/osprepare"
	"github.com/ciao-project/ciao/qemu"
	"github.com/ciao-project/ciao/ssntp/uuid"
)

const metaDataTemplate = `
{
  "uuid": "{{.UUID}}",
  "hostname": "{{.Hostname}}"
}
`

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
	Mounts         []mount
	Hostname       string
	UUID           string
	PackageUpgrade string
	ciaoDir        string
	instanceDir    string
	keyPath        string
	publicKeyPath  string
}

func (w *workspace) MountPath(tag string) string {
	for _, m := range w.Mounts {
		if m.Tag == tag {
			return m.Path
		}
	}

	return ""
}

func installDeps(ctx context.Context) {
	osprepare.InstallDeps(ctx, ciaoDevDeps, logger{})
}

func hostSupportsNestedKVMIntel() bool {
	data, err := ioutil.ReadFile("/sys/module/kvm_intel/parameters/nested")
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(data)) == "Y"
}

func hostSupportsNestedKVMAMD() bool {
	data, err := ioutil.ReadFile("/sys/module/kvm_amd/parameters/nested")
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(data)) == "1"
}

func hostSupportsNestedKVM() bool {
	return hostSupportsNestedKVMIntel() || hostSupportsNestedKVMAMD()
}

func prepareSSHKeys(ctx context.Context, ws *workspace) error {
	_, privKeyErr := os.Stat(ws.keyPath)
	_, pubKeyErr := os.Stat(ws.publicKeyPath)

	if pubKeyErr != nil || privKeyErr != nil {
		err := exec.CommandContext(ctx, "ssh-keygen",
			"-f", ws.keyPath, "-t", "rsa", "-N", "").Run()
		if err != nil {
			return fmt.Errorf("Unable to generate SSH key pair : %v", err)
		}
	}

	publicKey, err := ioutil.ReadFile(ws.publicKeyPath)
	if err != nil {
		return fmt.Errorf("Unable to read public ssh key: %v", err)
	}

	ws.PublicKey = string(publicKey)
	return nil
}

func getProxy(upper, lower string) (string, error) {
	proxy := os.Getenv(upper)
	if proxy == "" {
		proxy = os.Getenv(lower)
	}

	if proxy == "" {
		return "", nil
	}

	if proxy[len(proxy)-1] == '/' {
		proxy = proxy[:len(proxy)-1]
	}

	proxyURL, err := url.Parse(proxy)
	if err != nil {
		return "", fmt.Errorf("Failed to parse %s : %v", proxy, err)
	}
	return proxyURL.String(), nil
}

func prepareEnv(ctx context.Context) (*workspace, error) {
	var err error

	ws := &workspace{HTTPServerPort: 8080}
	data, err := exec.Command("go", "env", "GOPATH").Output()
	if err == nil {
		ws.GoPath = filepath.Clean(strings.TrimSpace(string(data)))
	}
	ws.Home = os.Getenv("HOME")
	if ws.Home == "" {
		return nil, fmt.Errorf("HOME is not defined")
	}
	ws.User = os.Getenv("USER")
	if ws.User == "" {
		return nil, fmt.Errorf("USER is not defined")
	}

	ws.HTTPProxy, err = getProxy("HTTP_PROXY", "http_proxy")
	if err != nil {
		return nil, err
	}

	ws.HTTPSProxy, err = getProxy("HTTPS_PROXY", "https_proxy")
	if err != nil {
		return nil, err
	}

	if ws.HTTPSProxy != "" {
		u, _ := url.Parse(ws.HTTPSProxy)
		u.Scheme = "http"
		ws.HTTPSProxy = u.String()
	}

	ws.NoProxy = os.Getenv("no_proxy")
	ws.ciaoDir = path.Join(ws.Home, ".ciao-down")
	ws.instanceDir = path.Join(ws.ciaoDir, "instance")
	ws.keyPath = path.Join(ws.ciaoDir, "id_rsa")
	ws.publicKeyPath = fmt.Sprintf("%s.pub", ws.keyPath)

	data, err = exec.Command("git", "config", "--global", "user.name").Output()
	if err == nil {
		ws.GitUserName = strings.TrimSpace(string(data))
	}

	data, err = exec.Command("git", "config", "--global", "user.email").Output()
	if err == nil {
		ws.GitEmail = strings.TrimSpace(string(data))
	}

	ws.UUID = uuid.Generate().String()

	return ws, nil
}

func createCloudInitISO(ctx context.Context, instanceDir string, userData, metaData []byte) error {
	isoPath := path.Join(instanceDir, "config.iso")
	return qemu.CreateCloudInitISO(ctx, instanceDir, isoPath, userData, metaData)
}

func downloadFN(ws *workspace, URL, location string) string {
	url := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("10.0.2.2:%d", ws.HTTPServerPort),
		Path:   "download",
	}
	q := url.Query()
	q.Set(urlParam, URL)
	url.RawQuery = q.Encode()
	return fmt.Sprintf("wget %s -O %s", url.String(), location)
}

func beginTaskFN(ws *workspace, message string) string {
	const infoStr = `curl -X PUT -d "%s" 10.0.2.2:%d`
	return fmt.Sprintf(infoStr, message, ws.HTTPServerPort)
}

func endTaskCheckFN(ws *workspace) string {
	const checkStr = `if [ $? -eq 0 ] ; then ret="OK" ; else ret="FAIL" ; fi ; ` +
		`curl -X PUT -d $ret 10.0.2.2:%d`
	return fmt.Sprintf(checkStr, ws.HTTPServerPort)
}

func endTaskOkFN(ws *workspace) string {
	const okStr = `curl -X PUT -d "OK" 10.0.2.2:%d`
	return fmt.Sprintf(okStr, ws.HTTPServerPort)
}

func endTaskFailFN(ws *workspace) string {
	const failStr = `curl -X PUT -d "FAIL" 10.0.2.2:%d`
	return fmt.Sprintf(failStr, ws.HTTPServerPort)
}

func finishedFN(ws *workspace) string {
	const finishedStr = `curl -X PUT -d "FINISHED" 10.0.2.2:%d`
	return fmt.Sprintf(finishedStr, ws.HTTPServerPort)
}

func proxyVarsFN(ws *workspace) string {
	var buf bytes.Buffer
	if ws.NoProxy != "" {
		buf.WriteString("no_proxy=")
		buf.WriteString(ws.NoProxy)
		buf.WriteString(" ")
		buf.WriteString("NO_PROXY=")
		buf.WriteString(ws.NoProxy)
		buf.WriteString(" ")
	}
	if ws.HTTPProxy != "" {
		buf.WriteString("http_proxy=")
		buf.WriteString(ws.HTTPProxy)
		buf.WriteString(" HTTP_PROXY=")
		buf.WriteString(ws.HTTPProxy)
		buf.WriteString(" ")
	}
	if ws.HTTPSProxy != "" {
		buf.WriteString("https_proxy=")
		buf.WriteString(ws.HTTPSProxy)
		buf.WriteString(" HTTPS_PROXY=")
		buf.WriteString(ws.HTTPSProxy)
		buf.WriteString(" ")
	}
	return strings.TrimSpace(buf.String())
}

func proxyEnvFN(ws *workspace, indent int) string {
	var buf bytes.Buffer
	spaces := strings.Repeat(" ", indent)
	if ws.NoProxy != "" {
		buf.WriteString(spaces)
		buf.WriteString(`no_proxy="`)
		buf.WriteString(ws.NoProxy)
		buf.WriteString(`"` + "\n")
		buf.WriteString(spaces)
		buf.WriteString(`NO_PROXY="`)
		buf.WriteString(ws.NoProxy)
		buf.WriteString(`"` + "\n")
	}
	if ws.HTTPProxy != "" {
		buf.WriteString(spaces)
		buf.WriteString(`http_proxy="`)
		buf.WriteString(ws.HTTPProxy)
		buf.WriteString(`"` + "\n")
		buf.WriteString(spaces)
		buf.WriteString(`HTTP_PROXY="`)
		buf.WriteString(ws.HTTPProxy)
		buf.WriteString(`"` + "\n")
	}
	if ws.HTTPSProxy != "" {
		buf.WriteString(spaces)
		buf.WriteString(`https_proxy="`)
		buf.WriteString(ws.HTTPSProxy)
		buf.WriteString(`"` + "\n")
		buf.WriteString(spaces)
		buf.WriteString(`HTTPS_PROXY="`)
		buf.WriteString(ws.HTTPSProxy)
		buf.WriteString(`"`)
	}
	return buf.String()
}

func buildISOImage(ctx context.Context, instanceDir, tmpl string, ws *workspace, debug bool) error {
	funcMap := template.FuncMap{
		"proxyVars":    proxyVarsFN,
		"proxyEnv":     proxyEnvFN,
		"download":     downloadFN,
		"beginTask":    beginTaskFN,
		"endTaskCheck": endTaskCheckFN,
		"endTaskOk":    endTaskOkFN,
		"endTaskFail":  endTaskFailFN,
		"finished":     finishedFN,
	}

	udt := template.Must(template.New("user-data").Funcs(funcMap).Parse(tmpl))
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

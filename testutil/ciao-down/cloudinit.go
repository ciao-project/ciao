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
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/01org/ciao/qemu"
	"github.com/golang/glog"
)

const userDataTemplate = `
{{- define "PROXIES" -}}
{{if len .HTTPSProxy }}https_proxy={{.HTTPSProxy}} {{end -}}
{{if len .HTTPProxy }}http_proxy={{.HTTPProxy}} {{end -}}
{{end}}
#cloud-config
mounts:
 - [hostgo, {{.GoPath}}, 9p, "trans=virtio,version=9p2000.L", 0, 0]
write_files:
{{- if len $.HTTPProxy }}
 - content: |
     Acquire::http::Proxy "{{$.HTTPProxy}}";
   path: /etc/apt/apt.conf
 - content: |
     [Service]
     Environment="HTTP_PROXY={{$.HTTPProxy}}"{{if len .HTTPSProxy}} "HTTPS_PROXY={{.HTTPSProxy}}{{end}}"{{if len .NoProxy}} "NO_PROXY={{.NoProxy}},singlevm{{end}}"
   path: /etc/systemd/system/docker.service.d/http-proxy.conf
{{- end}}
 - content: |
     #!/bin/sh
     printf "\n"
     printf "To run Single VM:\n"
     printf "\n"
     printf "cd {{.GoPath}}/src/github.com/01org/ciao/testutil/singlevm\n"
     printf "./setup.sh\n"
   path: /etc/update-motd.d/10-ciao-help-text
   permissions: '0755'
 - content: |
     deb https://apt.dockerproject.org/repo ubuntu-xenial main
   path: /etc/apt/sources.list.d/docker.list

runcmd:
 - echo "127.0.0.1 singlevm" >> /etc/hosts
 - rm /etc/update-motd.d/10-help-text /etc/update-motd.d/51-cloudguest
 - rm /etc/update-motd.d/90-updates-available
 - rm /etc/legal
 - curl -X PUT -d "VM Booted" 10.0.2.2:{{.HTTPServerPort}}
{{if len $.HTTPProxy }}
 - echo "HTTP_PROXY={{.HTTPProxy}}" >> /etc/environment
 - echo "http_proxy={{.HTTPProxy}}" >> /etc/environment
{{end -}}
{{- if len $.HTTPSProxy }}
 - echo "HTTPS_PROXY={{.HTTPSProxy}}" >> /etc/environment
 - echo "https_proxy={{.HTTPSProxy}}" >> /etc/environment
{{end}}
{{- if or (len .HTTPSProxy) (len .HTTPProxy) }}
 - echo no_proxy="{{if len .NoProxy}}{{.NoProxy}},{{end}}singlevm"  >> /etc/environment
{{end}}

 - curl -X PUT -d "Downloading Go" 10.0.2.2:{{.HTTPServerPort}}
 - echo "GOPATH={{.GoPath}}" >> /etc/environment
 - echo "PATH=$PATH:/usr/local/go/bin:{{$.GoPath}}/bin"  >> /etc/environment
 - {{template "PROXIES" .}}wget https://storage.googleapis.com/golang/go1.7.3.linux-amd64.tar.gz -O /tmp/go1.7.3.linux-amd64.tar.gz
 - tar -C /usr/local -xzf /tmp/go1.7.3.linux-amd64.tar.gz
 - rm /tmp/go1.7.3.linux-amd64.tar.gz
 - curl -X PUT -d "Go Installed" 10.0.2.2:{{.HTTPServerPort}}

 - groupadd docker
 - sudo gpasswd -a {{.User}} docker
 - {{template "PROXIES" .}}apt-get install apt-transport-https ca-certificates
 - {{template "PROXIES" .}}apt-key adv --keyserver hkp://p80.pool.sks-keyservers.net:80 --recv-keys 58118E89F3A912897C070ADBF76221572C52609D

 - curl -X PUT -d "Retrieving updated list of packages" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "PROXIES" .}}apt-get update
 - curl -X PUT -d "Package list updated" 10.0.2.2:{{.HTTPServerPort}}

 - curl -X PUT -d "Upgrading" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "PROXIES" .}}apt-get upgrade -y
 - curl -X PUT -d "Upgrade complete" 10.0.2.2:{{.HTTPServerPort}}

 - curl -X PUT -d "Installing Docker" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "PROXIES" .}}apt-get install docker-engine -y
 - curl -X PUT -d "Docker installed" 10.0.2.2:{{.HTTPServerPort}}

 - curl -X PUT -d "Installing GCC" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "PROXIES" .}}apt-get install gcc -y
 - curl -X PUT -d "GCC installed" 10.0.2.2:{{.HTTPServerPort}}

 - curl -X PUT -d "Installing QEMU" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "PROXIES" .}}apt-get install qemu-system-x86 -y
 - curl -X PUT -d "QEMU installed" 10.0.2.2:{{.HTTPServerPort}}

 - curl -X PUT -d "Installing xorriso" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "PROXIES" .}}apt-get install xorriso -y
 - curl -X PUT -d "Xorriso installed" 10.0.2.2:{{.HTTPServerPort}}

 - curl -X PUT -d "Installing ceph-common" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "PROXIES" .}}apt-get install ceph-common -y
 - curl -X PUT -d "Ceph-common installed" 10.0.2.2:{{.HTTPServerPort}}

 - curl -X PUT -d "Installing Openstack client" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "PROXIES" .}}apt-get install python-openstackclient -y
 - curl -X PUT -d "Openstack client installed" 10.0.2.2:{{.HTTPServerPort}}

 - curl -X PUT -d "Auto removing unused components" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "PROXIES" .}}apt-get auto-remove -y
 - curl -X PUT -d "Unused components removed" 10.0.2.2:{{.HTTPServerPort}}

 - curl -X PUT -d "Building ciao" 10.0.2.2:{{.HTTPServerPort}}
 - sudo -u {{.User}} {{template "PROXIES" .}} GOPATH={{.GoPath}} /usr/local/go/bin/go get github.com/01org/ciao/...
 - curl -X PUT -d "ciao built" 10.0.2.2:{{.HTTPServerPort}}

 - curl -X PUT -d "Installing Go development utils" 10.0.2.2:{{.HTTPServerPort}}
 - sudo -u {{.User}} {{template "PROXIES" .}} GOPATH={{.GoPath}} /usr/local/go/bin/go get github.com/fzipp/gocyclo github.com/gordonklaus/ineffassign github.com/golang/lint/golint github.com/client9/misspell/cmd/misspell
 - curl -X PUT -d "Go development utils installed" 10.0.2.2:{{.HTTPServerPort}}

 - chown {{.User}}:{{.User}} -R {{.GoPath}}

 - curl -X PUT -d "Pulling ceph/demo" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "PROXIES" .}} docker pull ceph/demo
 - curl -X PUT -d "Retrieved ceph/demo" 10.0.2.2:{{.HTTPServerPort}}

 - curl -X PUT -d "Pulling clearlinux/keystone" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "PROXIES" .}} docker pull clearlinux/keystone
 - curl -X PUT -d "Retrieved clearlinux/keystone" 10.0.2.2:{{.HTTPServerPort}}

 - mkdir -p /home/{{.User}}/local

 - curl -X PUT -d "Downloading Fedora-Cloud-Base-24-1.2.x86_64.qcow2" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "PROXIES" .}}wget https://download.fedoraproject.org/pub/fedora/linux/releases/24/CloudImages/x86_64/images/Fedora-Cloud-Base-24-1.2.x86_64.qcow2 -O /home/{{.User}}/local/Fedora-Cloud-Base-24-1.2.x86_64.qcow2
 - curl -X PUT -d "Fedora-Cloud-Base-24-1.2.x86_64.qcow2 Downloaded" 10.0.2.2:{{.HTTPServerPort}}

 - curl -X PUT -d "Downloading CNCI image" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "PROXIES" .}}wget https://download.clearlinux.org/demos/ciao/clear-8260-ciao-networking.img.xz -O /home/{{.User}}/local/clear-8260-ciao-networking.img.xz
 - curl -X PUT -d "CNCI image downloaded" 10.0.2.2:{{.HTTPServerPort}}

 - curl -X PUT -d "Downloading latest clear cloud image" 10.0.2.2:{{.HTTPServerPort}}
 - LATEST=$({{template "PROXIES" .}} curl -s https://download.clearlinux.org/latest) &&  {{template "PROXIES" .}} wget https://download.clearlinux.org/releases/"$LATEST"/clear/clear-"$LATEST"-cloud.img.xz -O /home/{{.User}}/local/clear-"$LATEST"-cloud.img.xz
 - curl -X PUT -d "Latest clear cloud image downloaded" 10.0.2.2:{{.HTTPServerPort}}

 - cd /home/{{.User}}/local && xz -T0 --decompress *.xz

 - chown {{.User}}:{{.User}} -R /home/{{.User}}/local
{{if len .GitUserName}}
 - curl -X PUT -d "Setting git user.name" 10.0.2.2:{{.HTTPServerPort}}
 - sudo -u {{.User}} git config --global user.name "{{.GitUserName}}"
{{end}}
{{if len .GitEmail}}
 - curl -X PUT -d "Setting git user.email" 10.0.2.2:{{.HTTPServerPort}}
 - sudo -u {{.User}} git config --global user.email {{.GitEmail}}
{{end}}
 - curl -X PUT -d "FINISHED" 10.0.2.2:{{.HTTPServerPort}}

users:
  - name: {{.User}}
    gecos: CIAO Demo User
    lock-passwd: true
    shell: /bin/bash
    passwd: $1$vzmNmLLD$04bivxcjdXRzZLUd.enRl1
    sudo: ALL=(ALL) NOPASSWD:ALL
    ssh-authorized-keys:
    - {{.PublicKey}}
`

const metaDataTemplate = `
{
  "uuid": "ddb4a2de-e5a5-4107-b302-e845cecd7613",
  "hostname": "singlevm"
}
`

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
		glog.Errorf("Unable to create config drive directory %s", dataDirPath)
		return err
	}

	err = ioutil.WriteFile(metaDataPath, metaData, 0644)
	if err != nil {
		glog.Errorf("Unable to create %s", metaDataPath)
		return err
	}

	err = ioutil.WriteFile(userDataPath, userData, 0644)
	if err != nil {
		glog.Errorf("Unable to create %s", userDataPath)
		return err
	}

	cmd := exec.CommandContext(ctx, "xorriso", "-as", "mkisofs", "-R", "-V", "config-2",
		"-o", isoPath, configDrivePath)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Unable to create cloudinit iso image %v", err)
	}

	return nil
}

func buildISOImage(ctx context.Context, instanceDir string, ws *workspace) error {
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

func bootVM(ctx context.Context, ws *workspace) error {
	disconnectedCh := make(chan struct{})
	socket := path.Join(ws.instanceDir, "socket")
	qmp, _, err := qemu.QMPStart(ctx, socket, qemu.QMPConfig{}, disconnectedCh)
	if err == nil {
		qmp.Shutdown()
		return fmt.Errorf("VM is already running")
	}

	vmImage := path.Join(ws.instanceDir, "image.qcow2")
	isoPath := path.Join(ws.instanceDir, "config.iso")
	fsdevParam := fmt.Sprintf("local,security_model=passthrough,id=fsdev0,path=%s",
		ws.GoPath)
	args := []string{
		"-qmp", fmt.Sprintf("unix:%s,server,nowait", socket), "-m", "8G", "-smp", "cpus=2",
		"-drive", fmt.Sprintf("file=%s,if=virtio,aio=threads,format=qcow2", vmImage),
		"-drive", fmt.Sprintf("file=%s,if=virtio,media=cdrom", isoPath),
		"-daemonize", "-enable-kvm", "-cpu", "host",
		"-net", "user,hostfwd=tcp::10022-:22", "-net", "nic,model=virtio",
		"-fsdev", fsdevParam,
		"-device", "virtio-9p-pci,id=fs0,fsdev=fsdev0,mount_tag=hostgo",
		"-display", "none", "-vga", "none",
	}
	output, err := qemu.LaunchCustomQemu(ctx, "", args, nil, nil)
	if err != nil {
		return fmt.Errorf("Failed to launch qemu : %v, %s", err, output)
	}
	return nil
}

func executeQMPCommand(ctx context.Context, instanceDir string,
	cmd func(ctx context.Context, q *qemu.QMP) error) error {
	socket := path.Join(instanceDir, "socket")
	disconnectedCh := make(chan struct{})
	qmp, _, err := qemu.QMPStart(ctx, socket, qemu.QMPConfig{}, disconnectedCh)
	if err != nil {
		return fmt.Errorf("Failed to connect to VM")
	}
	defer qmp.Shutdown()

	err = qmp.ExecuteQMPCapabilities(ctx)
	if err != nil {
		return fmt.Errorf("Unable to query QEMU caps")
	}

	err = cmd(ctx, qmp)
	if err != nil {
		return fmt.Errorf("Unable to execute vm command : %v", err)
	}

	return nil
}

func stopVM(ctx context.Context, instanceDir string) error {
	return executeQMPCommand(ctx, instanceDir, func(ctx context.Context, q *qemu.QMP) error {
		return q.ExecuteSystemPowerdown(ctx)
	})
}

func quitVM(ctx context.Context, instanceDir string) error {
	return executeQMPCommand(ctx, instanceDir, func(ctx context.Context, q *qemu.QMP) error {
		return q.ExecuteQuit(ctx)
	})
}

func statusVM(ctx context.Context, instanceDir string) {
	status := "ciao down"
	ssh := "N/A"
	socket := path.Join(instanceDir, "socket")
	disconnectedCh := make(chan struct{})
	qmp, _, err := qemu.QMPStart(ctx, socket, qemu.QMPConfig{}, disconnectedCh)
	if err == nil {
		status = "ciao up"
		ssh = fmt.Sprintf("ssh 127.0.0.1 -p %d", 10022)
		defer qmp.Shutdown()
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Fprintf(w, "Status\t:\t%s\n", status)
	fmt.Fprintf(w, "SSH\t:\t%s\n", ssh)
	w.Flush()
}

func startHTTPServer(listener net.Listener, errCh chan error) {
	finished := false
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var b bytes.Buffer
		_, err := io.Copy(&b, r.Body)
		if err != nil {
			// TODO: Figure out what to do here
			return
		}
		line := string(b.Bytes())
		if line == "FINISHED" {
			_ = listener.Close()
			finished = true
			return
		}
		fmt.Println(line)
	})

	server := &http.Server{}
	go func() {
		_ = server.Serve(listener)
		if finished {
			errCh <- nil
		} else {
			errCh <- fmt.Errorf("HTTP server exited prematurely")
		}
	}()
}

func manageInstallation(ctx context.Context, instanceDir string, ws *workspace) error {
	socket := path.Join(instanceDir, "socket")
	disconnectedCh := make(chan struct{})

	qmp, _, err := qemu.QMPStart(ctx, socket, qemu.QMPConfig{}, disconnectedCh)
	if err != nil {
		return fmt.Errorf("Unable to connect to VM : %v", err)
	}

	qemuShutdown := true
	defer func() {
		if qemuShutdown {
			ctx, cancelFn := context.WithTimeout(context.Background(), time.Second)
			_ = qmp.ExecuteQuit(ctx)
			cancelFn()
		}
		qmp.Shutdown()
	}()

	err = qmp.ExecuteQMPCapabilities(ctx)
	if err != nil {
		return fmt.Errorf("Unable to query QEMU caps")
	}

	// TODO: Cleanup

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", ws.HTTPServerPort))
	if err != nil {
		return fmt.Errorf("Unable to create listener: %v", err)
	}

	errCh := make(chan error)
	startHTTPServer(listener, errCh)
	select {
	case <-ctx.Done():
		_ = listener.Close()
		<-errCh
		return ctx.Err()
	case err := <-errCh:
		if err == nil {
			qemuShutdown = false
		}
		return err
	case <-disconnectedCh:
		qemuShutdown = false
		_ = listener.Close()
		<-errCh
		return fmt.Errorf("Lost connection to QEMU instance")
	}
}

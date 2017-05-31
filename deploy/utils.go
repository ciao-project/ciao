/*
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
*/

package deploy

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"

	"github.com/01org/ciao/ssntp"
	"github.com/pkg/errors"
)

func hostnameWithFallback() string {
	hs, err := os.Hostname()
	if err != nil {
		hs = "localhost"
	}
	return hs
}

// CertName gives file name to use for downloaded certificate
func CertName(role ssntp.Role) string {
	return fmt.Sprintf("cert-%s-%s.pem", role.String(), hostnameWithFallback())
}

// CSRName gives file name to use for pending CSR
func CSRName(role ssntp.Role) string {
	return fmt.Sprintf("csr-%s-%s.pem", role.String(), hostnameWithFallback())
}

// PrivName gives file name for private key
func PrivName(role ssntp.Role) string {
	return fmt.Sprintf("priv-%s-%s.pem", role.String(), hostnameWithFallback())
}

// SudoCopyFile copies the file from the source to dest as root
func SudoCopyFile(dest, src string) error {
	cmd := exec.Command("sudo", "cp", src, dest)
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "Error running: %v", cmd.Args)
	}
	return nil
}

// SudoMakeDirectory makes the desired set of directories as root
func SudoMakeDirectory(path string) error {
	cmd := exec.Command("sudo", "mkdir", "-p", path)
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "Error running: %v", cmd.Args)
	}
	return nil
}

// SudoDeleteFile removes the desired file as root
func SudoDeleteFile(path string) error {
	cmd := exec.Command("sudo", "rm", path)
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "Error running: %v", cmd.Args)
	}
	return nil
}

var systemdServiceData = `
[Unit]
Description=%s service
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/%s --cacert=%s --cert=%s --v 3
Restart=no
KillMode=process

[Install]
WantedBy=multi-user.target
`

// DownloadTool downloads the desired tool from the server
func DownloadTool(deployServerHost, tool string) (string, error) {
	url := fmt.Sprintf("%s/download/%s", deployServerHost, tool)
	fmt.Printf("Downloading %s: %s\n", tool, url)

	r, err := http.Get(url)
	if err != nil {
		return "", errors.Wrap(err, "Error downloading tool")
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Expected OK when downloading binary: %s", r.Status)
	}

	f, err := ioutil.TempFile("", tool)
	if err != nil {
		return "", errors.Wrap(err, "Error creating temporary file for tool")
	}
	defer f.Close()

	if _, err := io.Copy(f, r.Body); err != nil {
		return "", errors.Wrap(err, "Error downloading file")
	}

	if err := os.Chmod(f.Name(), 0755); err != nil {
		return "", errors.Wrap(err, "Error changing file permission")
	}

	return f.Name(), nil
}

// InstallTool installs a tool to its final destination and manages it via systemd
func InstallTool(toolPath, tool, caCertPath, certPath string) error {
	fmt.Printf("Installing %s\n", tool)

	fmt.Printf("Stopping %s\n", tool)
	cmd := exec.Command("sudo", "systemctl", "stop", tool)
	cmd.Run()

	if err := SudoCopyFile(path.Join("/usr/local/bin/", tool), toolPath); err != nil {
		return errors.Wrap(err, "Error copying launcher to destination")
	}

	fmt.Println("Installing systemd unit file")
	systemdData := fmt.Sprintf(systemdServiceData, tool, tool, caCertPath, certPath)
	serviceFilePath := path.Join("/etc/systemd/system", fmt.Sprintf("%s.service", tool))

	f, err := ioutil.TempFile("", fmt.Sprintf("%s.service", tool))
	if err != nil {
		return errors.Wrap(err, "Error creating temporary file for service unit")
	}
	defer f.Close()
	defer os.Remove(f.Name())

	if _, err := f.Write([]byte(systemdData)); err != nil {
		return errors.Wrap(err, "Error writing service file data")
	}

	if err := SudoCopyFile(serviceFilePath, f.Name()); err != nil {
		return errors.Wrap(err, "Error copying systemd service file to destination")
	}

	fmt.Println("Reloading systemd unit files")
	cmd = exec.Command("sudo", "systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "Error running: %v", cmd.Args)
	}

	fmt.Printf("Starting %s\n", tool)
	cmd = exec.Command("sudo", "systemctl", "start", tool)
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "Error running: %v", cmd.Args)
	}

	return nil
}

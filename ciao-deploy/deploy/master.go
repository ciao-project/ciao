// Copyright © 2017 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package deploy

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"

	yaml "gopkg.in/yaml.v2"

	"strings"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"github.com/01org/ciao/ssntp/certs"
	"github.com/pkg/errors"
)

// ClusterConfiguration provides cluster setup information
type ClusterConfiguration struct {
	CephID                  string
	HTTPSCaCertPath         string
	HTTPSCertPath           string
	KeystoneServiceUser     string
	KeystoneServicePassword string
	KeystoneURL             string
	AdminSSHKeyFile         string
	AdminSSHPassword        string
	ComputeNet              string
	MgmtNet                 string
	ServerIP                string
}

var ciaoConfigDir = "/etc/ciao"
var ciaoPKIDir = "/etc/pki/ciao"

func createConfigurationFile(ctx context.Context, clusterConf *ClusterConfiguration) (string, error) {
	var adminSSHKeyData string
	if clusterConf.AdminSSHKeyFile != "" {
		buf, err := ioutil.ReadFile(clusterConf.AdminSSHKeyFile)
		if err != nil {
			return "", errors.Wrap(err, "Error reading SSH key path")
		}
		adminSSHKeyData = strings.TrimSpace(string(buf))
	}

	ciaoConfigPath := path.Join(ciaoConfigDir, "configuration.yaml")

	config := &payloads.Configure{}
	config.InitDefaults()
	config.Configure.Scheduler.ConfigStorageURI = ciaoConfigPath

	config.Configure.Storage.CephID = clusterConf.CephID

	// TODO: Generate certs if not supplied
	config.Configure.Controller.HTTPSCACert = clusterConf.HTTPSCaCertPath
	config.Configure.Controller.HTTPSKey = clusterConf.HTTPSCertPath

	config.Configure.Controller.IdentityUser = clusterConf.KeystoneServiceUser
	config.Configure.Controller.IdentityPassword = clusterConf.KeystoneServicePassword
	config.Configure.IdentityService.URL = clusterConf.KeystoneURL

	config.Configure.Controller.AdminPassword = clusterConf.AdminSSHPassword
	config.Configure.Controller.AdminSSHKey = adminSSHKeyData

	// TODO: Try and find these automatically
	config.Configure.Launcher.ComputeNetwork = []string{clusterConf.ComputeNet}
	config.Configure.Launcher.ManagementNetwork = []string{clusterConf.MgmtNet}
	config.Configure.Launcher.DiskLimit = false
	config.Configure.Launcher.MemoryLimit = false

	data, err := yaml.Marshal(config)
	if err != nil {
		return "", errors.Wrap(err, "Error creating marshalling configuration data")
	}

	f, err := ioutil.TempFile("", "configuration.yaml")
	if err != nil {
		return "", errors.Wrap(err, "Error creating temporary file")
	}
	defer func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}()

	_, err = f.Write(data)
	if err != nil {
		return "", errors.Wrap(err, "Error writing data to temporary file")
	}

	err = SudoMakeDirectory(ctx, ciaoConfigDir)
	if err != nil {
		return "", errors.Wrap(err, "Error creating ciao configuration directory")
	}

	err = SudoCopyFile(ctx, ciaoConfigPath, f.Name())
	if err != nil {
		_ = SudoRemoveDirectory(context.Background(), ciaoConfigDir)
		return "", errors.Wrap(err, "Error copying configuration file to destination")
	}

	return ciaoConfigPath, nil
}

func createSchedulerCerts(ctx context.Context, force bool, serverIP string) (string, string, error) {
	anchorCertPath := path.Join(ciaoPKIDir, CertName(ssntp.SCHEDULER))
	caCertPath := path.Join(ciaoPKIDir, "CAcert.pem")

	if !force {
		if _, err := os.Stat(anchorCertPath); err == nil {
			if _, err := os.Stat(caCertPath); err == nil {
				fmt.Printf("Scheduler (and CA) certificate already installed. Skipping creation.\n")
				return anchorCertPath, caCertPath, nil
			} else if !os.IsNotExist(err) {
				return "", "", errors.Wrap(err, "Error stat()ing CA cert file")
			}
		} else if !os.IsNotExist(err) {
			return "", "", errors.Wrap(err, "Error stat()ing cert file")
		}
	}

	certFile, err := ioutil.TempFile("", CertName(ssntp.SCHEDULER))
	if err != nil {
		return "", "", errors.Wrap(err, "Unable to create temporary file for scheduler cert")
	}
	defer func() { _ = certFile.Close() }()
	defer func() { _ = os.Remove(certFile.Name()) }()

	caCertFile, err := ioutil.TempFile("", "CAcert.pem")
	if err != nil {
		return "", "", errors.Wrap(err, "Unable to create temporary file for CA certificate")
	}
	defer func() { _ = caCertFile.Close() }()
	defer func() { _ = os.Remove(caCertFile.Name()) }()

	hs := hostnameWithFallback()

	hosts := []string{hs}
	mgmtIPs := []string{serverIP}

	template, err := certs.CreateCertTemplate(ssntp.SCHEDULER, "Ciao Deployment", "", hosts, mgmtIPs)
	if err != nil {
		return "", "", errors.Wrap(err, "Error creating scheduler certificate template")
	}

	if err := certs.CreateAnchorCert(template, false, certFile, caCertFile); err != nil {
		return "", "", errors.Wrap(err, "Error creating anchor certificate")
	}

	if err := SudoMakeDirectory(ctx, ciaoPKIDir); err != nil {
		return "", "", errors.Wrap(err, "Error creating system PKI directory")
	}

	if err := os.Chmod(certFile.Name(), 0644); err != nil {
		return "", "", errors.Wrap(err, "Error chmod()ing anchor certificate")
	}

	if err := os.Chmod(caCertFile.Name(), 0644); err != nil {
		return "", "", errors.Wrap(err, "Error chmod()ing CA certificate")
	}

	if err := SudoCopyFile(ctx, anchorCertPath, certFile.Name()); err != nil {
		return "", "", errors.Wrap(err, "Error copying anchor certificate to system location")
	}

	if err := SudoCopyFile(ctx, caCertPath, caCertFile.Name()); err != nil {
		_ = SudoRemoveFile(context.Background(), anchorCertPath)
		return "", "", errors.Wrap(err, "Error copying CA certificate to system location")
	}

	fmt.Printf("Scheduler certificate created in: %s\n", anchorCertPath)
	fmt.Printf("CA certificate installed in: %s\n", caCertPath)
	return anchorCertPath, caCertPath, nil

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

// InstallTool installs a tool to its final destination and manages it via systemd
func InstallTool(ctx context.Context, tool string, certPath string, caCertPath string) (errOut error) {
	fmt.Printf("Installing %s\n", tool)

	fmt.Printf("Stopping %s\n", tool)
	cmd := exec.Command("sudo", "systemctl", "stop", tool)

	toolPath := InGoPath(path.Join("/bin", tool))
	// Actively ignore this error as systemctl will fail if the service file is not
	// yet installed. This is fine as that will be the case for new installs.
	_ = cmd.Run()

	systemToolPath := path.Join("/usr/local/bin/", tool)
	if err := SudoCopyFile(ctx, systemToolPath, toolPath); err != nil {
		return errors.Wrap(err, "Error copying tool to destination")
	}
	defer func() {
		if errOut != nil {
			_ = SudoRemoveFile(context.Background(), systemToolPath)
		}
	}()

	fmt.Println("Installing systemd unit file")
	systemdData := fmt.Sprintf(systemdServiceData, tool, tool, caCertPath, certPath)
	serviceFilePath := path.Join("/etc/systemd/system", fmt.Sprintf("%s.service", tool))

	f, err := ioutil.TempFile("", fmt.Sprintf("%s.service", tool))
	if err != nil {
		return errors.Wrap(err, "Error creating temporary file for service unit")
	}
	defer func() { _ = f.Close() }()
	defer func() { _ = os.Remove(f.Name()) }()

	if _, err := f.Write([]byte(systemdData)); err != nil {
		return errors.Wrap(err, "Error writing service file data")
	}

	if err := SudoCopyFile(ctx, serviceFilePath, f.Name()); err != nil {
		return errors.Wrap(err, "Error copying systemd service file to destination")
	}
	defer func() {
		if errOut != nil {
			_ = SudoRemoveFile(context.Background(), serviceFilePath)
		}
	}()

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

func installScheduler(ctx context.Context, anchorCertPath string, caCertPath string) error {
	err := InstallTool(ctx, "ciao-scheduler", anchorCertPath, caCertPath)
	return errors.Wrap(err, "Error installing scheduler")
}

func unInstallScheduler(ctx context.Context) error {
	return errors.New("Not implemented yet")
}

func createControllerCert(ctx context.Context, anchorCertPath string) (string, error) {
	controllerCertPath := path.Join(ciaoPKIDir, CertName(ssntp.Controller))

	tmpPath, err := GenerateCert(anchorCertPath, ssntp.Controller)
	if err != nil {
		return "", errors.Wrap(err, "Error creating controller certificate")
	}
	defer func() { _ = os.Remove(tmpPath) }()

	if err := os.Chmod(tmpPath, 0644); err != nil {
		return "", errors.Wrap(err, "Error chmod()ing controller certificate")
	}

	if err := SudoCopyFile(ctx, controllerCertPath, tmpPath); err != nil {
		return "", errors.Wrap(err, "Error copying controller certififcate to destination")
	}

	return controllerCertPath, nil
}

func installController(ctx context.Context, controllerCertPath string, caCertPath string) error {
	err := InstallTool(ctx, "ciao-controller", controllerCertPath, caCertPath)
	return errors.Wrap(err, "Error installing controller")
}

func unInstallController(ctx context.Context) error {
	return errors.New("Not implemented yet")
}

// SetupMaster configures this machine to be a master node of the cluster
func SetupMaster(ctx context.Context, force bool, imageCacheDir string, clusterConf *ClusterConfiguration) (errOut error) {
	ciaoConfigPath, err := createConfigurationFile(ctx, clusterConf)
	if err != nil {
		return errors.Wrap(err, "Error creating cluster configuration file")
	}
	defer func() {
		if errOut != nil {
			_ = SudoRemoveFile(context.Background(), ciaoConfigPath)
			_ = SudoRemoveDirectory(context.Background(), ciaoConfigDir)
		}
	}()

	anchorCertPath, caCertPath, err := createSchedulerCerts(ctx, force, clusterConf.ServerIP)
	if err != nil {
		return errors.Wrap(err, "Error creating scheduler certificates")
	}
	defer func() {
		if errOut != nil {
			_ = SudoRemoveFile(context.Background(), anchorCertPath)
			_ = SudoRemoveFile(context.Background(), caCertPath)
		}
	}()

	err = installScheduler(ctx, anchorCertPath, caCertPath)
	if err != nil {
		return errors.Wrap(err, "Error installing scheduler")
	}
	defer func() {
		if errOut != nil {
			_ = unInstallScheduler(context.Background())
		}
	}()

	controllerCertPath, err := createControllerCert(ctx, anchorCertPath)
	if err != nil {
		return errors.Wrap(err, "Error installing controller certs")
	}
	defer func() {
		if errOut != nil {
			_ = SudoRemoveFile(context.Background(), controllerCertPath)
		}
	}()

	err = installController(ctx, controllerCertPath, caCertPath)
	if err != nil {
		return errors.Wrap(err, "Error installing controller")
	}
	defer func() {
		if errOut != nil {
			_ = unInstallController(context.Background())
		}
	}()

	err = CreateCNCIImage(ctx, anchorCertPath, caCertPath, imageCacheDir)
	if err != nil {
		return errors.Wrap(err, "Error creating CNCI image")
	}
	defer func() {
		if errOut != nil {
			// TODO: Remove CNCI image from cluster
		}
	}()

	return nil
}

func createLocalLauncherCert(ctx context.Context, anchorCertPath string) (string, error) {
	launcherCertPath := path.Join(ciaoPKIDir, CertName(ssntp.AGENT|ssntp.NETAGENT))

	tmpPath, err := GenerateCert(anchorCertPath, ssntp.AGENT|ssntp.NETAGENT)
	if err != nil {
		return "", errors.Wrap(err, "Error creating launcher certificate")
	}
	defer func() { _ = os.Remove(tmpPath) }()

	if err := os.Chmod(tmpPath, 0644); err != nil {
		return "", errors.Wrap(err, "Error chmod()ing launcher certificate")
	}

	if err := SudoCopyFile(ctx, launcherCertPath, tmpPath); err != nil {
		return "", errors.Wrap(err, "Error copying controller certififcate to destination")
	}

	return launcherCertPath, nil
}

func unInstallLocalLauncher(ctx context.Context) error {
	return errors.New("Not implemented yet")
}

func installLocalLauncher(ctx context.Context, launcherCertPath string, caCertPath string) error {
	err := InstallTool(ctx, "ciao-launcher", launcherCertPath, caCertPath)
	return errors.Wrap(err, "Error installing launcher")
}

// SetupLocalLauncher installs launcher in dual mode on this node for testing
func SetupLocalLauncher(ctx context.Context) (errOut error) {
	anchorCertPath := path.Join(ciaoPKIDir, CertName(ssntp.SCHEDULER))
	caCertPath := path.Join(ciaoPKIDir, "CAcert.pem")

	launcherCertPath, err := createLocalLauncherCert(ctx, anchorCertPath)
	if err != nil {
		return errors.Wrap(err, "Error installing launcher cert")
	}
	defer func() {
		if errOut != nil {
			_ = SudoRemoveFile(context.Background(), launcherCertPath)
		}
	}()

	err = installLocalLauncher(ctx, launcherCertPath, caCertPath)
	if err != nil {
		return errors.Wrap(err, "Error installing launcher")
	}
	defer func() {
		if errOut != nil {
			_ = unInstallLocalLauncher(context.Background())
		}
	}()
	return nil
}

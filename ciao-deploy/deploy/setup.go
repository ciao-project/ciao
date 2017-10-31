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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"text/template"

	yaml "gopkg.in/yaml.v2"

	"strings"

	"github.com/ciao-project/ciao/payloads"
	"github.com/ciao-project/ciao/ssntp"
	"github.com/ciao-project/ciao/ssntp/certs"
	"github.com/pkg/errors"
)

// ClusterConfiguration provides cluster setup information
type ClusterConfiguration struct {
	CephID            string
	HTTPSCaCertPath   string
	HTTPSCertPath     string
	AdminSSHKeyPath   string
	AdminSSHPassword  string
	ComputeNet        string
	MgmtNet           string
	ServerIP          string
	AuthCACertPath    string
	AuthAdminCertPath string
	ServerHostname    string
	DisableLimits     bool
}

type unitFileConf struct {
	Tool       string
	User       string
	CACertPath string
	CertPath   string
	Caps       []string
}

var ciaoLockDir = "/tmp/lock/ciao"
var ciaoDataDir = "/var/lib/ciao"
var ciaoConfigDir = "/etc/ciao"
var ciaoPKIDir = "/etc/pki/ciao"
var ciaoUser = "ciao"
var ciaoUserAndGroup = ciaoUser + ":" + ciaoUser

func createConfigurationFile(ctx context.Context, clusterConf *ClusterConfiguration) (string, error) {
	var adminSSHKeyData string
	if clusterConf.AdminSSHKeyPath != "" {
		buf, err := ioutil.ReadFile(clusterConf.AdminSSHKeyPath)
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
	config.Configure.Controller.ClientAuthCACertPath = clusterConf.AuthCACertPath

	config.Configure.Controller.AdminPassword = clusterConf.AdminSSHPassword
	config.Configure.Controller.AdminSSHKey = adminSSHKeyData

	config.Configure.Launcher.ComputeNetwork = []string{clusterConf.ComputeNet}
	config.Configure.Launcher.ManagementNetwork = []string{clusterConf.MgmtNet}

	// Disk limit checking is broken and should always be disabled.
	// See issue #1541
	config.Configure.Launcher.DiskLimit = false
	config.Configure.Launcher.MemoryLimit = !clusterConf.DisableLimits
	config.Configure.Launcher.ChildUser = ciaoUser

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

	cmd := exec.Command("sudo", "chown", ciaoUserAndGroup, ciaoConfigPath)
	if err := cmd.Run(); err != nil {
		_ = SudoRemoveDirectory(context.Background(), ciaoConfigDir)
		return "", errors.Wrapf(err, "Error running: %v", cmd.Args)
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

	hs := HostnameWithFallback()

	hosts := []string{hs}
	mgmtIPs := []string{serverIP}

	template, err := certs.CreateCertTemplate(ssntp.SCHEDULER, "Ciao Deployment", "", hosts, mgmtIPs)
	if err != nil {
		return "", "", errors.Wrap(err, "Error creating scheduler certificate template")
	}

	if err := certs.CreateAnchorCert(template, certFile, caCertFile); err != nil {
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

var systemdServiceData = `[Unit]
Description={{.Tool}} service
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/{{.Tool}} --cacert={{.CACertPath}} --cert={{.CertPath}} --v 3
Restart=no
KillMode=process
TasksMax=infinity
{{with .Caps}}
CapabilityBoundingSet={{range $i, $v := .}}{{if (gt $i 0)}} {{end}}{{$v}}{{- end}}
{{else}}
User={{.User}}
{{end}}
Group={{.User}}

[Install]
WantedBy=multi-user.target
`

// InstallTool installs a tool to its final destination and manages it via systemd
func InstallTool(ctx context.Context, config unitFileConf) (errOut error) {
	var systemdData bytes.Buffer
	err := template.Must(template.New("unit").Parse(systemdServiceData)).Execute(&systemdData, config)
	if err != nil {
		return errors.Wrapf(err, "Error generating systemd file for %s", config.Tool)
	}

	fmt.Printf("Installing %s\n", config.Tool)

	fmt.Printf("Stopping %s\n", config.Tool)
	cmd := exec.Command("sudo", "systemctl", "stop", config.Tool)

	toolPath := InGoPath(path.Join("/bin", config.Tool))
	// Actively ignore this error as systemctl will fail if the service file is not
	// yet installed. This is fine as that will be the case for new installs.
	_ = cmd.Run()

	systemToolPath := path.Join("/usr/local/bin/", config.Tool)
	if err := SudoCopyFile(ctx, systemToolPath, toolPath); err != nil {
		return errors.Wrap(err, "Error copying tool to destination")
	}
	defer func() {
		if errOut != nil {
			_ = SudoRemoveFile(context.Background(), systemToolPath)
		}
	}()

	fmt.Println("Installing systemd unit file")
	serviceFilePath := path.Join("/etc/systemd/system", fmt.Sprintf("%s.service", config.Tool))

	f, err := ioutil.TempFile("", fmt.Sprintf("%s.service", config.Tool))
	if err != nil {
		return errors.Wrap(err, "Error creating temporary file for service unit")
	}
	defer func() { _ = f.Close() }()
	defer func() { _ = os.Remove(f.Name()) }()

	if _, err := f.Write(systemdData.Bytes()); err != nil {
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

	fmt.Printf("Starting %s\n", config.Tool)
	cmd = exec.Command("sudo", "systemctl", "start", config.Tool)
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "Error running: %v", cmd.Args)
	}

	return nil
}

func installScheduler(ctx context.Context, anchorCertPath string, caCertPath string) error {
	err := InstallTool(ctx, unitFileConf{
		Tool:       "ciao-scheduler",
		User:       ciaoUser,
		CertPath:   anchorCertPath,
		CACertPath: caCertPath,
	})
	return errors.Wrap(err, "Error installing scheduler")
}

func uninstallTool(ctx context.Context, tool string) {
	cmd := exec.Command("sudo", "systemctl", "stop", tool)
	_ = cmd.Run()

	serviceFilePath := path.Join("/etc/systemd/system", fmt.Sprintf("%s.service", tool))
	_ = SudoRemoveFile(context.Background(), serviceFilePath)

	cmd = exec.Command("sudo", "systemctl", "daemon-reload")
	_ = cmd.Run()

	systemToolPath := path.Join("/usr/local/bin/", tool)
	_ = SudoRemoveFile(context.Background(), systemToolPath)
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
	err := InstallTool(ctx, unitFileConf{
		Tool:       "ciao-controller",
		User:       ciaoUser,
		CertPath:   controllerCertPath,
		CACertPath: caCertPath,
	})
	return errors.Wrap(err, "Error installing controller")
}

func setupEnvironment(conf *ClusterConfiguration) {
	_ = os.Setenv("CIAO_CONTROLLER", conf.ServerHostname)
	_ = os.Setenv("CIAO_ADMIN_CLIENT_CERT_FILE", conf.AuthAdminCertPath)
}

// OutputEnvironment prints the environment to be used to access cluster
func OutputEnvironment(conf *ClusterConfiguration) {
	fmt.Printf("Environment variables to access cluster:\n\n")

	fmt.Printf("export CIAO_CONTROLLER=\"%s\"\n", conf.ServerHostname)
	fmt.Printf("export CIAO_ADMIN_CLIENT_CERT_FILE=\"%s\"\n", conf.AuthAdminCertPath)
}

func createCiaoDirectory(ctx context.Context, path string) error {
	cmd := exec.CommandContext(ctx, "sudo", "mkdir", "-p", path)
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "Error creating %s", path)
	}
	cmd = exec.CommandContext(ctx, "sudo", "chown", ciaoUserAndGroup, path)
	if err := cmd.Run(); err != nil {
		_ = SudoRemoveDirectory(context.Background(), path)
		return errors.Wrapf(err, "Error running: %v", cmd.Args)
	}
	return nil
}

func createUserAndDirs(ctx context.Context) (func(), error) {
	cmd := exec.CommandContext(ctx, "sudo", "useradd", "-r", ciaoUser, "-G", "docker,kvm")
	if err := cmd.Run(); err != nil {
		return nil, errors.Wrapf(err, "Error running: %v", cmd.Args)
	}

	if err := createCiaoDirectory(ctx, ciaoDataDir); err != nil {
		return nil, err
	}

	if err := createCiaoDirectory(ctx, ciaoLockDir); err != nil {
		_ = SudoRemoveDirectory(context.Background(), ciaoDataDir)
		return nil, err
	}

	return func() {
		_ = SudoRemoveDirectory(context.Background(), ciaoLockDir)
		_ = SudoRemoveDirectory(context.Background(), ciaoDataDir)
	}, nil
}

func installCertsAndTools(ctx context.Context, force bool, imageCacheDir string, clusterConf *ClusterConfiguration) (errOut error) {
	authCaCertPath, authCertPath, err := CreateAdminCert(ctx, force)
	if err != nil {
		return errors.Wrap(err, "Error creating authentication certs")
	}
	defer func() {
		if errOut != nil {
			_ = SudoRemoveFile(context.Background(), authCaCertPath)
			_ = SudoRemoveFile(context.Background(), authCertPath)
		}
	}()

	clusterConf.AuthCACertPath = authCaCertPath
	clusterConf.AuthAdminCertPath = authCertPath

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

	setupEnvironment(clusterConf)

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
			uninstallTool(context.Background(), "ciao-scheduler")
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
			uninstallTool(context.Background(), "ciao-controller")
		}
	}()

	err = CreateCNCIImage(ctx, anchorCertPath, caCertPath, imageCacheDir)
	if err != nil {
		return errors.Wrap(err, "Error creating CNCI image")
	}

	return nil
}

// SetupMaster configures this machine to be a master node of the cluster
func SetupMaster(ctx context.Context, force bool, imageCacheDir string, clusterConf *ClusterConfiguration) (errOut error) {
	cleanup, err := createUserAndDirs(ctx)
	if err != nil {
		return
	}
	defer func() {
		if errOut != nil {
			cleanup()
		}
	}()
	return installCertsAndTools(ctx, force, imageCacheDir, clusterConf)
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

func installLocalLauncher(ctx context.Context, launcherCertPath string, caCertPath string) error {
	err := InstallTool(ctx, unitFileConf{
		Tool:       "ciao-launcher",
		User:       ciaoUser,
		CertPath:   launcherCertPath,
		CACertPath: caCertPath,
		Caps: []string{
			"CAP_NET_ADMIN", "CAP_NET_RAW", "CAP_DAC_OVERRIDE",
			"CAP_DAC_OVERRIDE", "CAP_SETGID", "CAP_SETUID", "CAP_SYS_PTRACE",
			"CAP_SYS_MODULE",
		},
	})
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

	return nil
}

// UpdateMaster updates the running one the master
func UpdateMaster(ctx context.Context) error {
	anchorCertPath := path.Join(ciaoPKIDir, CertName(ssntp.SCHEDULER))
	controllerCertPath := path.Join(ciaoPKIDir, CertName(ssntp.Controller))
	caCertPath := path.Join(ciaoPKIDir, "CAcert.pem")

	if err := installScheduler(ctx, anchorCertPath, caCertPath); err != nil {
		return errors.Wrap(err, "Error installating scheduler")
	}

	if err := installController(ctx, controllerCertPath, caCertPath); err != nil {
		return errors.Wrap(err, "Error installng controller")
	}

	return nil
}

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
	"os"
	"path"
	"sync"
	"text/template"
	"time"

	"github.com/ciao-project/ciao/ssntp"
	"github.com/pkg/errors"
)

// InstallToolRemote installs a tool a on a remote machine and setups it up with systemd
func InstallToolRemote(ctx context.Context, sshUser string, hostname string, config unitFileConf) (errOut error) {
	var systemdData bytes.Buffer
	err := template.Must(template.New("unit").Parse(systemdServiceData)).Execute(&systemdData, config)
	if err != nil {
		return errors.Wrapf(err, "Error generating systemd file for %s", config.Tool)
	}

	fmt.Printf("%s: Installing %s\n", hostname, config.Tool)

	fmt.Printf("%s: Stopping %s\n", hostname, config.Tool)
	_ = SSHRunCommand(ctx, sshUser, hostname, fmt.Sprintf("sudo systemctl stop %s", config.Tool))
	// Actively ignore this error as systemctl will fail if the service file is not
	// yet installed. This is fine as that will be the case for new installs.

	toolPath := InGoPath(path.Join("/bin", config.Tool))

	tf, err := os.Open(toolPath)
	if err != nil {
		return errors.Wrap(err, "Error opening tool locally")
	}
	defer func() { _ = tf.Close() }()

	systemToolPath := path.Join("/usr/local/bin/", config.Tool)
	err = SSHCreateFile(ctx, sshUser, hostname, systemToolPath, tf)
	if err != nil {
		return errors.Wrap(err, "Error copying file to destination")
	}
	defer func() {
		if errOut != nil {
			_ = SSHRunCommand(context.Background(), sshUser, hostname, fmt.Sprintf("sudo rm %s", systemToolPath))
		}
	}()

	err = SSHRunCommand(ctx, sshUser, hostname, fmt.Sprintf("sudo chmod a+x %s", systemToolPath))
	if err != nil {
		return errors.Wrap(err, "Error making tool executable on node")
	}

	fmt.Printf("%s: Installing systemd unit file\n", hostname)
	serviceFilePath := path.Join("/etc/systemd/system", fmt.Sprintf("%s.service", config.Tool))

	err = SSHCreateFile(ctx, sshUser, hostname, serviceFilePath, &systemdData)
	if err != nil {
		return errors.Wrap(err, "Error copying file to destination")
	}
	defer func() {
		if errOut != nil {
			_ = SSHRunCommand(context.Background(), sshUser, hostname, fmt.Sprintf("sudo rm %s", serviceFilePath))
		}
	}()

	fmt.Printf("%s: Reloading systemd unit files\n", hostname)
	err = SSHRunCommand(ctx, sshUser, hostname, "sudo systemctl daemon-reload")
	if err != nil {
		return errors.Wrap(err, "Error restarting systemctl on node")
	}

	fmt.Printf("%s: Starting %s\n", hostname, config.Tool)
	err = SSHRunCommand(ctx, sshUser, hostname, fmt.Sprintf("sudo systemctl start %s", config.Tool))
	if err != nil {
		return errors.Wrap(err, "Error starting tool on node")
	}

	return nil
}

func createRemoteLauncherCert(ctx context.Context, anchorCertPath string, role ssntp.Role, hostname string, sshUser string) (string, error) {
	launcherCertPath := path.Join(ciaoPKIDir, fmt.Sprintf("cert-%s-%s.pem", role.String(), hostname))

	tmpPath, err := GenerateCert(anchorCertPath, role)
	if err != nil {
		return "", errors.Wrap(err, "Error creating launcher certificate")
	}
	defer func() { _ = os.Remove(tmpPath) }()

	f, err := os.Open(tmpPath)
	if err != nil {
		return "", errors.Wrap(err, "Error opening temporary cert file")
	}
	defer func() { _ = f.Close() }()

	err = SSHRunCommand(ctx, sshUser, hostname, fmt.Sprintf("sudo mkdir -p %s", ciaoPKIDir))
	if err != nil {
		return "", errors.Wrap(err, "Error creating ciao PKI directory")
	}

	err = SSHCreateFile(ctx, sshUser, hostname, launcherCertPath, f)
	if err != nil {
		return "", errors.Wrap(err, "Error copying file to destination")
	}

	return launcherCertPath, nil
}

func createRemoteCACert(ctx context.Context, caCertPath string, hostname string, sshUser string) error {
	f, err := os.Open(caCertPath)
	if err != nil {
		return errors.Wrap(err, "Error opening CA cert file")
	}
	defer func() { _ = f.Close() }()

	err = SSHCreateFile(ctx, sshUser, hostname, caCertPath, f)
	if err != nil {
		return errors.Wrap(err, "Error copying file to destination")
	}

	return nil
}

func createRemoteCiaoDirectory(ctx context.Context, hostname, sshUser, path string) error {
	err := SSHRunCommand(ctx, sshUser, hostname, fmt.Sprintf("sudo mkdir -p %s", path))
	if err != nil {
		return errors.Wrapf(err, "Error creating remote directory %s", path)
	}
	err = SSHRunCommand(ctx, sshUser, hostname, fmt.Sprintf("sudo chown %s %s", ciaoUserAndGroup, path))
	if err != nil {
		_ = SSHRunCommand(context.Background(), sshUser, hostname, fmt.Sprintf("sudo rmdir %s", path))
		return errors.Wrapf(err, "Error chowning %s", path)
	}
	return nil
}

func setupNode(ctx context.Context, anchorCertPath string, caCertPath string, hostname string, sshUser string, networkNode bool) (errOut error) {
	err := SSHRunCommand(ctx, sshUser, hostname,
		fmt.Sprintf("sudo useradd -r %s -G docker,kvm -d %s -s /bin/false", ciaoUser, ciaoDataDir))
	if err != nil {
		return errors.Wrapf(err, "Error creating %s user", ciaoUser)
	}

	err = createRemoteCiaoDirectory(ctx, hostname, sshUser, ciaoDataDir)
	if err != nil {
		return err
	}

	defer func() {
		if errOut != nil {
			_ = SSHRunCommand(context.Background(), sshUser, hostname, fmt.Sprintf("sudo rm -rf %s", ciaoDataDir))
		}
	}()

	err = createRemoteCiaoDirectory(ctx, hostname, sshUser, ciaoLockDir)
	if err != nil {
		return err
	}

	defer func() {
		if errOut != nil {
			_ = SSHRunCommand(context.Background(), sshUser, hostname, fmt.Sprintf("sudo rm -rf %s", ciaoLockDir))
		}
	}()

	var role ssntp.Role = ssntp.AGENT
	if networkNode {
		role = ssntp.NETAGENT
	}

	remoteCertPath, err := createRemoteLauncherCert(ctx, anchorCertPath, role, hostname, sshUser)
	if err != nil {
		return errors.Wrap(err, "Error generating remote launcher certificate")
	}
	defer func() {
		if errOut != nil {
			_ = SSHRunCommand(context.Background(), sshUser, hostname, fmt.Sprintf("sudo rm %s", remoteCertPath))
		}
	}()

	err = createRemoteCACert(ctx, caCertPath, hostname, sshUser)
	if err != nil {
		return errors.Wrap(err, "Error creating remote CA certificate")
	}
	defer func() {
		if errOut != nil {
			_ = SSHRunCommand(context.Background(), sshUser, hostname, fmt.Sprintf("sudo rm %s", caCertPath))
		}
	}()

	err = InstallToolRemote(ctx, sshUser, hostname, unitFileConf{
		Tool:       "ciao-launcher",
		User:       ciaoUser,
		CertPath:   remoteCertPath,
		CACertPath: caCertPath,
		Caps: []string{"CAP_NET_ADMIN", "CAP_NET_RAW", "CAP_DAC_OVERRIDE",
			"CAP_SETGID", "CAP_SETUID", "CAP_SYS_PTRACE", "CAP_SYS_MODULE"},
	})
	if err != nil {
		return errors.Wrap(err, "Error installing tool on node")
	}
	return nil
}

// SetupNodes joins the given nodes as launcher nodes
func SetupNodes(ctx context.Context, sshUser string, networkNode bool, hosts []string) error {
	anchorCertPath := path.Join(ciaoPKIDir, CertName(ssntp.SCHEDULER))
	caCertPath := path.Join(ciaoPKIDir, "CAcert.pem")

	var wg sync.WaitGroup
	for _, host := range hosts {
		wg.Add(1)
		go func(hostname string) {
			err := setupNode(ctx, anchorCertPath, caCertPath, hostname, sshUser, networkNode)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error setting up node: %s: %v\n", hostname, err)
			}
			wg.Done()
		}(host)
	}
	wg.Wait()
	return nil
}

func teardownNode(ctx context.Context, hostname string, sshUser string) error {
	var errOut error
	tool := "ciao-launcher"
	fmt.Printf("%s: Stopping %s\n", hostname, tool)
	err := SSHRunCommand(ctx, sshUser, hostname, fmt.Sprintf("sudo systemctl stop %s", tool))
	if err != nil {
		errOut = errors.Wrap(err, "Error stopping tool on node")
		fmt.Fprintln(os.Stderr, errOut.Error())
	}

	fmt.Printf("%s: Removing %s service file\n", hostname, tool)
	serviceFilePath := path.Join("/etc/systemd/system", fmt.Sprintf("%s.service", tool))
	err = SSHRunCommand(ctx, sshUser, hostname, fmt.Sprintf("sudo rm %s", serviceFilePath))
	if err != nil {
		errOut = errors.Wrap(err, "Error removing systemd service file")
		fmt.Fprintln(os.Stderr, errOut.Error())
	}

	fmt.Printf("%s: Reloading systemd unit files\n", hostname)
	err = SSHRunCommand(ctx, sshUser, hostname, "sudo systemctl daemon-reload")
	if err != nil {
		errOut = errors.Wrap(err, "Error restarting systemctl on node")
		fmt.Fprintln(os.Stderr, errOut.Error())
	}

	fmt.Printf("%s: Removing %s certificates\n", hostname, tool)
	caCertPath := path.Join(ciaoPKIDir, "CAcert.pem")
	err = SSHRunCommand(ctx, sshUser, hostname, fmt.Sprintf("sudo rm %s", caCertPath))
	if err != nil {
		errOut = errors.Wrap(err, "Error removing CA certificate")
		fmt.Fprintln(os.Stderr, errOut.Error())
	}

	// One of these can fail so ignore errors on both.
	var computeAgentRole ssntp.Role = ssntp.AGENT
	computeAgentCertPath := path.Join(ciaoPKIDir, fmt.Sprintf("cert-%s-%s.pem", computeAgentRole.String(), hostname))
	_ = SSHRunCommand(ctx, sshUser, hostname, fmt.Sprintf("sudo rm %s", computeAgentCertPath))

	var networkAgentRole ssntp.Role = ssntp.NETAGENT
	networkAgentCertPath := path.Join(ciaoPKIDir, fmt.Sprintf("cert-%s-%s.pem", networkAgentRole.String(), hostname))
	_ = SSHRunCommand(ctx, sshUser, hostname, fmt.Sprintf("sudo rm %s", networkAgentCertPath))

	err = SSHRunCommand(ctx, sshUser, hostname, fmt.Sprintf("sudo rmdir %s", ciaoPKIDir))
	if err != nil {
		errOut = errors.Wrap(err, "Error removing ciao PKI directory")
		fmt.Fprintln(os.Stderr, errOut.Error())
	}

	// Need extra timeout here due to #343
	systemToolPath := path.Join("/usr/local/bin/", tool)
	fmt.Printf("%s: Performing ciao-launcher hard reset\n", hostname)
	timeoutContext, cancelFunc := context.WithTimeout(ctx, time.Second*60)
	err = SSHRunCommand(timeoutContext, sshUser, hostname, fmt.Sprintf("sudo %s --hard-reset", systemToolPath))
	cancelFunc()
	if timeoutContext.Err() != context.DeadlineExceeded && err != nil {
		errOut = errors.Wrap(err, "Error doing hard-reset on ciao-launcher")
		fmt.Fprintln(os.Stderr, errOut.Error())
	}

	fmt.Printf("%s: Removing %s binary\n", hostname, tool)
	err = SSHRunCommand(ctx, sshUser, hostname, fmt.Sprintf("sudo rm %s", systemToolPath))
	if err != nil {
		errOut = errors.Wrap(err, "Error removing tool binary")
		fmt.Fprintln(os.Stderr, errOut.Error())
	}

	err = SSHRunCommand(ctx, sshUser, hostname, fmt.Sprintf("sudo rm -rf %s", ciaoLockDir))
	if err != nil {
		errOut = errors.Wrapf(err, "Error removing %s", ciaoLockDir)
		fmt.Fprintln(os.Stderr, errOut.Error())
	}

	err = SSHRunCommand(ctx, sshUser, hostname, fmt.Sprintf("sudo rm -rf %s", ciaoDataDir))
	if err != nil {
		errOut = errors.Wrapf(err, "Error removing %s", ciaoDataDir)
		fmt.Fprintln(os.Stderr, errOut.Error())
	}

	return errOut
}

// TeardownNodes removes launcher from the given nodes
func TeardownNodes(ctx context.Context, sshUser string, hosts []string) error {
	var wg sync.WaitGroup
	for _, host := range hosts {
		wg.Add(1)
		go func(hostname string) {
			err := teardownNode(ctx, hostname, sshUser)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error tearing down node: %s: %v\n", hostname, err)
			}
			wg.Done()
		}(host)
	}
	wg.Wait()
	return nil
}

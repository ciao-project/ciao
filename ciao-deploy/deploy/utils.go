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
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/ciao-project/ciao/ssntp"
	"github.com/ciao-project/ciao/ssntp/certs"
	"github.com/pkg/errors"
)

// DefaultImageCacheDir provides the default location for downloaded images
func DefaultImageCacheDir() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}

	return path.Join(u.HomeDir, ".cache", "ciao", "images")
}

// DownloadImage checks for a cached image in the cache directory and downloads
// otherwise. The returned string is the path to the file and the boolean
// indicates if it was downloaded on this function call.
func DownloadImage(ctx context.Context, url string, imageCacheDir string) (_ string, _ bool, errOut error) {
	ss := strings.Split(url, "/")
	localName := ss[len(ss)-1]

	imagePath := path.Join(imageCacheDir, localName)
	if _, err := os.Stat(imagePath); err == nil {
		fmt.Printf("Using already downloaded image: %s\n", imagePath)
		return imagePath, false, nil
	} else if !os.IsNotExist(err) {
		return "", false, errors.Wrap(err, "Error when stat()ing expected image path")
	}

	if err := os.MkdirAll(imageCacheDir, 0755); err != nil {
		return "", false, errors.Wrap(err, "Unable to create image cache directory")
	}

	f, err := ioutil.TempFile(imageCacheDir, localName)
	if err != nil {
		return "", false, errors.Wrap(err, "Unable to create temporary file for download")
	}
	defer func() {
		if err := f.Close(); err != nil {
			if errOut == nil {
				errOut = errors.Wrap(err, "Error closing temporary file")
			}
		}
	}()
	defer func() { _ = os.Remove(f.Name()) }()

	fmt.Printf("Downloading: %s\n", url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", false, errors.Wrap(err, "Error creating HTTP request")
	}
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", false, errors.Wrap(err, "Error making HTTP request")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("Unexpected status when downloading URL: %s: %s", url, resp.Status)
	}

	buf := make([]byte, 1<<20)
	_, err = io.CopyBuffer(f, resp.Body, buf)
	if err != nil {
		return "", false, errors.Wrap(err, "Error copying from HTTP response to file")
	}

	if err := os.Rename(f.Name(), imagePath); err != nil {
		return "", false, errors.Wrap(err, "Error moving downloaded image to destination")
	}

	fmt.Printf("Image downloaded to %s\n", imagePath)

	return imagePath, true, nil
}

// SudoCommandContext runs the given command with root privileges
func SudoCommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	newArgs := append([]string{name}, args...)
	return exec.CommandContext(ctx, "sudo", newArgs...)
}

// SudoCopyFile copies the file from the source to dest as root
func SudoCopyFile(ctx context.Context, dest string, src string) error {
	cmd := SudoCommandContext(ctx, "cp", src, dest)
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "Error running: %v", cmd.Args)
	}
	return nil
}

// SudoMakeDirectory creates the desired directory hiearchy as root
func SudoMakeDirectory(ctx context.Context, dest string) error {
	cmd := SudoCommandContext(ctx, "mkdir", "-p", dest)
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "Error running: %v", cmd.Args)
	}
	return nil
}

// SudoRemoveDirectory deletes the directory hiearchy as root
func SudoRemoveDirectory(ctx context.Context, dest string) error {
	cmd := SudoCommandContext(ctx, "rm", "-rf", dest)
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "Error running: %v", cmd.Args)
	}
	return nil
}

// SudoRemoveFile deletes the file as root
func SudoRemoveFile(ctx context.Context, dest string) error {
	cmd := SudoCommandContext(ctx, "rm", "-f", dest)
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "Error running: %v", cmd.Args)
	}
	return nil
}

// SudoChownFiles changes the user and group one or more files to ciaoUserAndGroup
func SudoChownFiles(ctx context.Context, dest ...string) error {
	cmd := SudoCommandContext(ctx, "chown", append([]string{ciaoUserAndGroup}, dest...)...)
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "Error running: %v", cmd.Args)
	}
	return nil
}

// InGoPath returns the desired path relative to $GOPATH
func InGoPath(path string) string {
	data, err := exec.Command("go", "env", "GOPATH").Output()
	gp := ""
	if err == nil {
		gp = filepath.Clean(strings.TrimSpace(string(data)))
	}
	return filepath.Join(gp, path)
}

// HostnameWithFallback returns hostname with a fallback to localhost
func HostnameWithFallback() string {
	hs, err := os.Hostname()
	if err != nil {
		hs = "localhost"
	}
	return hs
}

// CertName gives file name to use for downloaded certificate
func CertName(role ssntp.Role) string {
	return fmt.Sprintf("cert-%s-%s.pem", role.String(), HostnameWithFallback())
}

// GenerateCert creates a certificate signed by the anchor certificate for a given role
func GenerateCert(anchorCertPath string, role ssntp.Role) (path string, errOut error) {
	anchorCertBytes, err := ioutil.ReadFile(anchorCertPath)
	if err != nil {
		return "", errors.Wrap(err, "Error reading anchor cert")
	}

	t, err := certs.CreateCertTemplate(role, "Ciao Deployment", "", []string{}, []string{})
	if err != nil {
		return "", errors.Wrap(err, "Error creating certificate template")
	}

	f, err := ioutil.TempFile("", "cert")
	if err != nil {
		return "", errors.Wrap(err, "Error creating temporary certifate file")
	}
	defer func() {
		if errOut != nil {
			_ = os.Remove(f.Name())
		}
	}()

	err = certs.CreateCert(t, anchorCertBytes, f)
	if err != nil {
		_ = f.Close()
		return "", errors.Wrap(err, "Error creating certificate from anchor")
	}

	err = f.Close()
	if err != nil {
		return "", errors.Wrap(err, "Error closing certificate file")
	}

	return f.Name(), nil
}

// DialSSHContext is the equivalent of ssh.Dial but allowing the use of a context
func DialSSHContext(ctx context.Context, network string, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	d := &net.Dialer{}
	conn, err := d.DialContext(ctx, network, addr)
	if err != nil {
		return nil, errors.Wrap(err, "Error dialing connection")
	}

	c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return nil, errors.Wrap(err, "Error creating connection")
	}
	return ssh.NewClient(c, chans, reqs), nil
}

func sshClient(ctx context.Context, username string, host string) (*ssh.Client, error) {
	if sshAuthSock := os.Getenv("SSH_AUTH_SOCK"); sshAuthSock == "" {
		return nil, errors.New("ssh-agent must be running and populated with key for node")
	}

	d := &net.Dialer{}
	c, err := d.DialContext(ctx, "unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return nil, errors.Wrap(err, "Error connectin to SSH agent")
	}

	agentMethod := ssh.PublicKeysCallback(agent.NewClient(c).Signers)

	home := ""
	u, err := user.Current()
	if err == nil {
		home = u.HomeDir
	}

	knownHostsPath := path.Join(home, ".ssh", "known_hosts")
	hkcb, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, errors.Wrap(err, "Error loading SSH host key verification file")
	}

	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			agentMethod,
		},
		HostKeyCallback: hkcb,
	}

	client, err := DialSSHContext(ctx, "tcp", fmt.Sprintf("%s:22", host), sshConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Error dialing SSH connection")
	}

	return client, nil
}

// SSHRunCommand is a convenience function to run a command on a given host.
// This assumes the key is already in the keyring for the provided user.
func SSHRunCommand(ctx context.Context, user string, host string, command string) error {
	client, err := sshClient(ctx, user, host)
	if err != nil {
		return errors.Wrap(err, "Error creating client")
	}
	defer func() { _ = client.Close() }()

	session, err := client.NewSession()
	if err != nil {
		return errors.Wrap(err, "Error creating session")
	}
	defer func() { _ = session.Close() }()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return errors.Wrapf(err, "Error running %s on %s: %s", command, host, output)
	}
	return nil
}

// SSHCreateFile creates a file on a remote machine
func SSHCreateFile(ctx context.Context, user string, host string, dest string, f io.Reader) error {
	client, err := sshClient(ctx, user, host)
	if err != nil {
		return errors.Wrap(err, "Error creating client")
	}
	defer func() { _ = client.Close() }()

	session, err := client.NewSession()
	if err != nil {
		return errors.Wrap(err, "Error creating session")
	}
	defer func() { _ = session.Close() }()

	p, err := session.StdinPipe()
	if err != nil {
		return errors.Wrap(err, "Error getting STDIN pipe")
	}

	errCh := make(chan error, 1)
	go func() {
		buf := make([]byte, 1<<20)
		_, err := io.CopyBuffer(p, f, buf)
		errCh <- err

		err = p.Close()
		errCh <- err
	}()

	err = session.Run(fmt.Sprintf("sudo tee %s", dest))
	if err != nil {
		return errors.Wrap(err, "Error running tee command")
	}

	err = <-errCh
	if err != nil {
		<-errCh
		return errors.Wrap(err, "Error copying data to target")
	}

	err = <-errCh
	if err != nil {
		return errors.Wrap(err, "Error closing SSH pipe to target")
	}

	return nil
}

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
	"strings"

	"github.com/ciao-project/ciao/bat"
	"github.com/ciao-project/ciao/ssntp"
	"github.com/pkg/errors"
)

var cnciURL = "https://download.clearlinux.org/demos/ciao/clear-8260-ciao-networking.img.xz"
var cnciImageID = "4e16e743-265a-4bf2-9fd1-57ada0b28904"

func mountImage(ctx context.Context, fp string, mntDir string) (string, error) {
	cmd := SudoCommandContext(ctx, "losetup", "-f", "--show", "-P", fp)
	buf, err := cmd.Output()
	if err != nil {
		return "", errors.Wrapf(err, "Error running: %v", cmd.Args)
	}

	devPath := strings.TrimSpace(string(buf))
	fmt.Printf("Image %s available as %s\n", fp, devPath)

	pPath := fmt.Sprintf("%sp%d", devPath, 2)
	cmd = SudoCommandContext(ctx, "mount", pPath, mntDir)
	err = cmd.Run()
	if err != nil {
		_ = unMountImage(context.Background(), devPath, mntDir)
		return devPath, errors.Wrapf(err, "Error running: %v", cmd.Args)
	}
	fmt.Printf("Device %s mounted as %s\n", pPath, mntDir)

	return devPath, nil
}

func unMountImage(ctx context.Context, devPath string, mntDir string) error {
	var errOut error

	cmd := SudoCommandContext(ctx, "umount", mntDir)
	err := cmd.Run()
	if err != nil {
		if errOut == nil {
			errOut = errors.Wrapf(err, "Error running: %v", cmd.Args)
		}
		fmt.Fprintf(os.Stderr, "Error unmounting: %v\n", err)
	} else {
		fmt.Printf("Directory unmounted: %s\n", mntDir)
	}

	cmd = SudoCommandContext(ctx, "losetup", "-d", devPath)
	err = cmd.Run()
	if err != nil {
		if errOut == nil {
			errOut = errors.Wrapf(err, "Error running: %v", cmd.Args)
		}
		fmt.Fprintf(os.Stderr, "Error removing loopback: %v\n", err)
	} else {
		fmt.Printf("Loopback removed: %s\n", devPath)
	}

	return errOut
}

func copyFiles(ctx context.Context, mntDir string, agentCertPath string, caCertPath string) error {
	p := path.Join(mntDir, "/var/lib/ciao")
	err := SudoMakeDirectory(ctx, p)
	if err != nil {
		return errors.Wrap(err, "Error making certificate directory")
	}

	p = path.Join(mntDir, "/var/lib/ciao/cert-client-localhost.pem")
	err = SudoCopyFile(ctx, p, agentCertPath)
	if err != nil {
		return errors.Wrap(err, "Error copying agent cert to image")
	}

	p = path.Join(mntDir, "/var/lib/ciao/CAcert-server-localhost.pem")
	err = SudoCopyFile(ctx, p, caCertPath)
	if err != nil {
		return errors.Wrap(err, "Error copying CA cert to image")
	}

	p = path.Join(mntDir, "/usr/sbin")
	err = SudoCopyFile(ctx, p, InGoPath("/bin/ciao-cnci-agent"))
	if err != nil {
		return errors.Wrap(err, "Error copying agent binary")
	}

	p = path.Join(mntDir, "/usr/lib/systemd/system")
	err = SudoCopyFile(ctx, p, InGoPath("/src/github.com/ciao-project/ciao/networking/ciao-cnci-agent/scripts/ciao-cnci-agent.service"))
	if err != nil {
		return errors.Wrap(err, "Error copying service file into image")
	}

	p = path.Join(mntDir, "/etc/systemd/system/default.target.wants")
	err = SudoMakeDirectory(ctx, p)
	if err != nil {
		return errors.Wrap(err, "Error making systemd default directory")
	}

	cmd := SudoCommandContext(ctx, "chroot", mntDir, "systemctl", "enable", "ciao-cnci-agent.service")
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, "Error enabling cnci agent on startup")
	}

	p = path.Join(mntDir, "/var/lib/cloud")
	err = SudoRemoveDirectory(ctx, p)
	if err != nil {
		return errors.Wrap(err, "Error removing cloud-init data")
	}

	return nil
}

func prepareImage(ctx context.Context, baseImage string, agentCertPath string, caCertPath string) (_ string, errOut error) {
	preparedImagePath := strings.TrimSuffix(baseImage, ".xz")

	cmd := exec.CommandContext(ctx, "unxz", "-f", "-k", baseImage)
	err := cmd.Run()
	if err != nil {
		return "", errors.Wrap(err, "Error uncompressing cnci image")
	}
	defer func() {
		if errOut != nil {
			fmt.Printf("Removing %s\n", preparedImagePath)
			_ = os.Remove(preparedImagePath)
		}
	}()

	mntDir, err := ioutil.TempDir("", "cnci-mount")
	if err != nil {
		return "", errors.Wrap(err, "Error making mount point directory")
	}
	defer func() {
		fmt.Printf("Removing mount point: %s\n", mntDir)
		err := os.RemoveAll(mntDir)
		if err != nil {
			if errOut == nil {
				errOut = errors.Wrap(err, "Error removing mount point")
			}
		}
	}()

	devPath, err := mountImage(ctx, preparedImagePath, mntDir)
	if err != nil {
		return "", errors.Wrap(err, "Error mounting image")
	}
	defer func() {
		err := unMountImage(context.Background(), devPath, mntDir)
		if err != nil {
			if errOut == nil {
				errOut = errors.Wrap(err, "Error unmounting image")
			}
		}
	}()

	err = copyFiles(ctx, mntDir, agentCertPath, caCertPath)
	if err != nil {
		return "", errors.Wrap(err, "Error copying files into image")
	}

	return preparedImagePath, nil
}

// CreateCNCIImage creates a customised CNCI image in the system
func CreateCNCIImage(ctx context.Context, anchorCertPath string, caCertPath string, imageCacheDir string) (errOut error) {
	agentCertPath, err := GenerateCert(anchorCertPath, ssntp.CNCIAGENT)
	if err != nil {
		return errors.Wrap(err, "Error creating agent certificate")
	}
	defer func() { _ = os.Remove(agentCertPath) }()

	baseImagePath, downloaded, err := DownloadImage(ctx, cnciURL, imageCacheDir)
	if err != nil {
		return errors.Wrap(err, "Error downloading image")
	}
	defer func() {
		if errOut != nil && downloaded {
			_ = os.Remove(baseImagePath)
		}
	}()

	preparedImage, err := prepareImage(ctx, baseImagePath, agentCertPath, caCertPath)
	if err != nil {
		return errors.Wrap(err, "Error preparing image")
	}
	defer func() { _ = os.Remove(preparedImage) }()

	fmt.Printf("Image prepared at: %s\n", preparedImage)

	imageOpts := &bat.ImageOptions{
		ID:         cnciImageID,
		Visibility: "internal",
		Name:       "ciao CNCI image",
	}

	fmt.Printf("Uploading image as %s\n", imageOpts.ID)
	i, err := bat.AddImage(ctx, "", preparedImage, imageOpts)
	if err != nil {
		return errors.Wrap(err, "Error uploading image to controller")
	}

	fmt.Printf("CNCI image uploaded as %s\n", i.ID)

	return nil
}

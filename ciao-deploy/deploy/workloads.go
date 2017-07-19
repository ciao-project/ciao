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
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strings"
	"sync"

	"github.com/01org/ciao/bat"
	"github.com/pkg/errors"
)

var containerCloudInit = `
---
#cloud-config
runcmd:
    - [ /bin/bash, -c, "while true; do sleep 60; done" ]
...
`

var vmCloudInit = `
---
#cloud-config
users:
  - name: demouser
    gecos: CIAO Demo User
    lock-passwd: false
    sudo: ALL=(ALL) NOPASSWD:ALL
...
`

type baseWorkload struct {
	url        string
	imageName  string
	imageID    string
	extra      bool
	localPath  string
	cloudInit  string
	opts       bat.WorkloadOptions
	downloaded bool
	workloadID string
}

type clearWorkload struct {
	wd      baseWorkload
	version string
}

type workloadDetails interface {
	Download() error
	Extra() bool
	Upload() error
	CreateWorkload() error
}

var images = []workloadDetails{
	&baseWorkload{
		url:       "https://download.fedoraproject.org/pub/fedora/linux/releases/24/CloudImages/x86_64/images/Fedora-Cloud-Base-24-1.2.x86_64.qcow2",
		imageName: "Fedora Cloud Base 24-1.2",
		extra:     true,
		cloudInit: vmCloudInit,
		opts: bat.WorkloadOptions{
			Description: "Fedora test VM",
			VMType:      "qemu",
			FWType:      "legacy",
			Defaults: bat.DefaultResources{
				VCPUs: 2,
				MemMB: 128,
			},
		},
	},
	&baseWorkload{
		url:       "https://cloud-images.ubuntu.com/xenial/current/xenial-server-cloudimg-amd64-disk1.img",
		imageName: "Ubuntu Server 16.04",
		extra:     false,
		cloudInit: vmCloudInit,
		opts: bat.WorkloadOptions{
			Description: "Ubuntu test VM",
			VMType:      "qemu",
			FWType:      "legacy",
			Defaults: bat.DefaultResources{
				VCPUs: 2,
				MemMB: 256,
			},
		},
	},
	&clearWorkload{
		wd: baseWorkload{
			extra:     true,
			cloudInit: vmCloudInit,
			opts: bat.WorkloadOptions{
				Description: "Clear Linux test VM",
				VMType:      "qemu",
				FWType:      "efi",
				Defaults: bat.DefaultResources{
					VCPUs: 2,
					MemMB: 128,
				},
			},
		},
	},
	&baseWorkload{
		cloudInit: containerCloudInit,
		opts: bat.WorkloadOptions{
			Description: "Debian latest test container",
			VMType:      "docker",
			ImageName:   "debian:latest",
			Defaults: bat.DefaultResources{
				VCPUs: 2,
				MemMB: 128,
			},
		},
	},
	&baseWorkload{
		cloudInit: containerCloudInit,
		opts: bat.WorkloadOptions{
			Description: "Ubuntu latest test container",
			VMType:      "docker",
			ImageName:   "ubuntu:latest",
			Defaults: bat.DefaultResources{
				VCPUs: 2,
				MemMB: 128,
			},
		},
	},
}

// CreateBatWorkloads creates all necessary workloads to run BAT
func CreateBatWorkloads(allWorkloads bool) (errOut error) {
	for _, wd := range images {
		if wd.Extra() && !allWorkloads {
			continue
		}

		if err := wd.Download(); err != nil {
			return errors.Wrap(err, "Error downloading image")
		}
	}

	var wg sync.WaitGroup

	for _, wd := range images {
		if wd.Extra() && !allWorkloads {
			continue
		}

		wg.Add(1)
		go func(wd workloadDetails) {
			if err := wd.Upload(); err != nil {
				errOut = errors.Wrap(err, "Error uploading image")
			}

			if err := wd.CreateWorkload(); err != nil {
				errOut = errors.Wrap(err, "Error creating workload")
			}

			wg.Done()
		}(wd)
	}

	wg.Wait()

	return errOut
}

func imageCacheDir() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", errors.Wrap(err, "Unable to get user home directory")
	}

	icd := path.Join(u.HomeDir, ".cache", "ciao", "images")
	return icd, nil
}

func (wd *baseWorkload) download(url string) error {
	ss := strings.Split(url, "/")
	localName := ss[len(ss)-1]

	icd, err := imageCacheDir()
	if err != nil {
		return errors.Wrap(err, "Unable to get image cache directory")
	}

	imagePath := path.Join(icd, localName)
	if _, err := os.Stat(imagePath); err == nil {
		wd.localPath = imagePath
		fmt.Printf("Using already downloaded image: %s\n", wd.localPath)
		return nil
	} else if !os.IsNotExist(err) {
		return errors.Wrap(err, "Error when stat()ing expected image path")
	}

	if err := os.MkdirAll(icd, 0755); err != nil {
		return errors.Wrap(err, "Unable to create image cache directory")
	}

	f, err := ioutil.TempFile(icd, localName)
	if err != nil {
		return errors.Wrap(err, "Unable to create temporary file for download")
	}
	defer f.Close()
	defer os.Remove(f.Name())

	fmt.Printf("Downloading: %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "Unable to fetch URL")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Unexpected status when downloading URL: %s: %s", url, resp.Status)
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		return errors.Wrap(err, "Error copying from http to file")
	}

	wd.localPath = imagePath
	if err := os.Rename(f.Name(), wd.localPath); err != nil {
		return errors.Wrap(err, "Error moving downloaded image to destination")
	}

	fmt.Printf("Image downloaded to %s\n", imagePath)

	wd.downloaded = true // for later cleanup
	return nil
}

func (wd *baseWorkload) Download() error {
	if wd.opts.VMType != "qemu" {
		return nil
	}

	return wd.download(wd.url)
}

func (cwd *clearWorkload) Download() error {
	resp, err := http.Get("https://download.clearlinux.org/latest")
	if err != nil {
		return errors.Wrap(err, "Error downloading clear version info")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Unexpected status when downloading clear version info: %s", resp.Status)
	}

	versionBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "Error reading clear version info")
	}
	cwd.version = strings.TrimSpace(string(versionBytes))

	icd, err := imageCacheDir()
	if err != nil {
		return errors.Wrap(err, "Error getting image cache directory")
	}

	// Check if already extracted file is present
	fn := fmt.Sprintf("clear-%s-cloud.img", cwd.version)
	fp := path.Join(icd, fn)
	if _, err := os.Stat(fp); err == nil {
		cwd.wd.localPath = fp
		return nil
	} else if !os.IsNotExist(err) {
		return errors.Wrap(err, "Error stat()ing extracted clear image")
	}

	url := fmt.Sprintf("https://download.clearlinux.org/releases/%s/clear/%s.xz", cwd.version, fn)
	err = cwd.wd.download(url)
	if err != nil {
		return errors.Wrap(err, "Error downloading clear image")
	}

	cmd := exec.Command("unxz", "-f", cwd.wd.localPath)
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "Error when decompressing clear image")
	}
	cwd.wd.localPath = strings.TrimSuffix(cwd.wd.localPath, ".xz")

	return nil
}

func (wd *baseWorkload) Extra() bool {
	return wd.extra
}

func (cwd *clearWorkload) Extra() bool {
	return cwd.wd.extra
}

func (wd *baseWorkload) upload(fp, name string) error {
	opts := bat.ImageOptions{
		Name:       name,
		Visibility: "public",
	}
	fmt.Printf("Uploading image from %s\n", fp)

	i, err := bat.AddImage(context.Background(), "", fp, &opts)
	if err != nil {
		return errors.Wrap(err, "Error creating image")
	}

	wd.imageID = i.ID
	fmt.Printf("Image uploaded \"%s\" (%s) to %s\n", opts.Name, fp, i.ID)
	return nil
}

func (wd *baseWorkload) Upload() error {
	if wd.opts.VMType != "qemu" {
		return nil
	}

	return wd.upload(wd.localPath, wd.imageName)
}

func (cwd *clearWorkload) Upload() error {
	return cwd.wd.upload(cwd.wd.localPath, fmt.Sprintf("Clear Linux %s", cwd.version))
}

func (wd *baseWorkload) CreateWorkload() error {
	opts := wd.opts
	if opts.VMType == "qemu" {
		opts.Disks = []bat.Disk{
			{
				Source: &bat.Source{
					Type: "image",
					ID:   wd.imageID,
				},
				Ephemeral: true,
				Bootable:  true,
			},
		}
	}

	workloadID, err := bat.CreateWorkload(context.Background(), "", opts, wd.cloudInit)
	if err == nil {
		wd.workloadID = workloadID
		fmt.Printf("Workload created \"%s\" as %s\n", opts.Description, wd.workloadID)
	}

	return err
}

func (cwd *clearWorkload) CreateWorkload() error {
	return cwd.wd.CreateWorkload()
}

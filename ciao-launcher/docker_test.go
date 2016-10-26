/*
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
*/

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	storage "github.com/01org/ciao/ciao-storage"
)

type dockerTestMounter struct {
	mounts map[string]string
}

func (m dockerTestMounter) Mount(source, destination string) error {
	m.mounts[path.Base(destination)] = source
	return nil
}

func (m dockerTestMounter) Unmount(destination string, flags int) error {
	delete(m.mounts, path.Base(destination))
	return nil
}

type dockerTestStorage struct {
	root      string
	failAfter int
	count     int
}

func (s dockerTestStorage) MapVolumeToNode(volumeUUID string) (string, error) {
	if s.failAfter != -1 && s.failAfter >= s.count {
		return "", fmt.Errorf("MapVolumeToNode failure forced")
	}
	s.count++

	return "", nil
}

func (s dockerTestStorage) CreateBlockDevice(image *string, sizeGB int) (storage.BlockDevice, error) {
	return storage.BlockDevice{}, nil
}

func (s dockerTestStorage) DeleteBlockDevice(string) error {
	return nil
}

func (s dockerTestStorage) UnmapVolumeFromNode(volumeUUID string) error {
	return nil
}

func (s dockerTestStorage) GetVolumeMapping() (map[string][]string, error) {
	return nil, nil
}

func (s dockerTestStorage) cleanup() error {
	return os.RemoveAll(s.root)
}

func (s dockerTestStorage) CopyBlockDevice(volumeUUID string) (storage.BlockDevice, error) {
	return storage.BlockDevice{}, nil
}

// Checks that the logic of the code that mounts and unmounts ceph volumes in
// docker containers.
//
// We mount 4 volumes, check the mount commands are received correctly and check
// the correct directories are created.  We then unmount, and check that everything
// gets unmounted as expected.
//
// Calls to docker.mountVolumes and docker.unmountVolumes should succeed and the
// mounted volumes should be correctly cleaned up.
func TestDockerMountUnmount(t *testing.T) {
	root, err := ioutil.TempDir("", "mount-unmount")
	if err != nil {
		t.Fatalf("Unable to create temporary directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(root) }()

	instanceDir := path.Join(root, "volumes")

	s := dockerTestStorage{
		root:      root,
		failAfter: -1,
	}
	mounts := make(map[string]string)
	d := docker{
		cfg: &vmConfig{
			Volumes: []volumeConfig{
				{UUID: "92a1e4fa-8448-4260-adb1-4d2dd816cc7c"},
				{UUID: "5ce2c5bf-58d9-4573-b433-05550b945866"},
				{UUID: "11773eac-6b27-4bc1-8717-02e75ae5e063"},
				{UUID: "590603fb-c73e-4efa-941e-454b5d4f9857"},
			},
		},
		instanceDir:   root,
		storageDriver: s,
		mount:         dockerTestMounter{mounts: mounts},
	}

	_, err = d.prepareVolumes()
	if err != nil {
		t.Fatalf("Unable to prepare volumes: %v", err)
	}
	err = d.mapAndMountVolumes()
	if err != nil {
		t.Fatalf("Unable to map and mount volumes: %v", err)
	}

	dirInfo, err := ioutil.ReadDir(instanceDir)
	if err != nil {
		t.Fatalf("Unable to readdir %s: %v", instanceDir, err)
	}

	if len(dirInfo) != len(d.cfg.Volumes) {
		t.Fatalf("Unexpected number of volumes directories.  Found %d, expected %d",
			len(dirInfo), len(d.cfg.Volumes))
	}

	if len(dirInfo) != len(mounts) {
		t.Fatalf("Unexpected number of volumes mounted.  Found %d, expected %d",
			len(dirInfo), len(mounts))
	}

	for _, vol := range d.cfg.Volumes {
		var i int
		for i = 0; i < len(dirInfo); i++ {
			if vol.UUID == dirInfo[i].Name() {
				break
			}
		}
		if i == len(dirInfo) {
			t.Fatalf("%s not mounted", vol.UUID)
		}
		if _, ok := mounts[vol.UUID]; !ok {
			t.Fatalf("%s does not seem to have been mounted", vol.UUID)
		}
	}

	d.umountVolumes(d.cfg.Volumes)
	if len(mounts) != 0 {
		t.Fatalf("Not all volumes have been unmounted")
	}
}

// Checks that everything is cleaned up correctly when a call to
// docker.mountVolumes fails.
//
// We call docker.mountVolumes with 4 volumes but arrange for the call to fail
// after the second volume has been created.
//
// docker.mountVolumes should fail but everything should be cleaned up despite
// the failure.
func TestDockerBadMount(t *testing.T) {
	root, err := ioutil.TempDir("", "mount-unmount")
	if err != nil {
		t.Fatalf("Unable to create temporary directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(root) }()

	s := dockerTestStorage{
		root:      root,
		failAfter: 2,
	}

	mounts := make(map[string]string)
	d := docker{
		cfg: &vmConfig{
			Volumes: []volumeConfig{
				{UUID: "92a1e4fa-8448-4260-adb1-4d2dd816cc7c"},
				{UUID: "5ce2c5bf-58d9-4573-b433-05550b945866"},
				{UUID: "11773eac-6b27-4bc1-8717-02e75ae5e063"},
				{UUID: "590603fb-c73e-4efa-941e-454b5d4f9857"},
			},
		},
		instanceDir:   root,
		storageDriver: s,
		mount:         dockerTestMounter{mounts: mounts},
	}

	_, err = d.prepareVolumes()
	if err != nil {
		t.Fatalf("Unable to prepare volumes: %v", err)
	}

	err = d.mapAndMountVolumes()
	if err == nil {
		t.Fatal("d.mountVolumes was expected to fail")
	}

	if len(mounts) != 0 {
		t.Fatal("mounts not cleaned up correctly")
	}
}

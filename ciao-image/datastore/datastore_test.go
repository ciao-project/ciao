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

package datastore

import (
	"strings"
	"testing"
)

func testCreateAndGet(t *testing.T, d RawDataStore, m MetaDataStore) {
	i := Image{
		ID:    "validID",
		State: Created,
	}

	cache := ImageCache{}
	cache.Init(d, m)

	// create the entry
	err := cache.CreateImage(i)
	if err != nil {
		t.Fatal(err)
	}

	// retrieve the entry
	image, err := cache.GetImage(i.ID)
	if err != nil {
		t.Fatal(err)
	}

	if image.ID != i.ID {
		t.Fatal(err)
	}
}

func testGetAll(t *testing.T, d RawDataStore, m MetaDataStore) {
	i := Image{
		ID:    "validID",
		State: Created,
	}

	cache := ImageCache{}
	cache.Init(d, m)

	// create the entry
	err := cache.CreateImage(i)
	if err != nil {
		t.Fatal(err)
	}

	// retrieve the entry
	images, err := cache.GetAllImages()
	if err != nil {
		t.Fatal(err)
	}

	if len(images) != 1 {
		t.Fatalf("len is actually %d\n", len(images))
	}

	if images[0].ID != i.ID {
		t.Fatal(err)
	}
}

func testDelete(t *testing.T, d RawDataStore, m MetaDataStore) {
	i := Image{
		ID:    "validID",
		State: Created,
	}

	cache := ImageCache{}
	cache.Init(d, m)

	// create the entry
	err := cache.CreateImage(i)
	if err != nil {
		t.Fatal(err)
	}

	// delete the entry
	err = cache.DeleteImage(i.ID)
	if err != nil {
		t.Fatal(err)
	}

	// now attempt to retrive the entry
	_, err = cache.GetImage(i.ID)
	if err == nil {
		t.Fatal(err)
	}
}

func testUpload(t *testing.T, d RawDataStore, m MetaDataStore) {
	i := Image{
		ID:    "validID",
		State: Created,
	}

	cache := ImageCache{}
	cache.Init(d, m)

	// create the entry
	err := cache.CreateImage(i)
	if err != nil {
		t.Fatal(err)
	}

	// Upload a string
	err = cache.UploadImage(i.ID, strings.NewReader("Upload file"))
	if err != nil {
		t.Fatal(err)
	}
}

var mountPoint = "/var/lib/ciao/images"

func TestPosixNoopCreateAndGet(t *testing.T) {
	testCreateAndGet(t, &Posix{MountPoint: mountPoint}, &Noop{})
}

func TestPosixNoopGetAll(t *testing.T) {
	testGetAll(t, &Posix{MountPoint: mountPoint}, &Noop{})
}

func TestPosixNoopDelete(t *testing.T) {
	testDelete(t, &Posix{MountPoint: mountPoint}, &Noop{})
}

func TestPosixNoopUpload(t *testing.T) {
	testUpload(t, &Posix{MountPoint: mountPoint}, &Noop{})
}

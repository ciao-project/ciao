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
	"testing"

	"github.com/01org/ciao/ciao-image/service"
)

func TestCreateAndRetrieve(t *testing.T) {
	i := service.Image{
		ID:    "validID",
		State: service.Created,
	}

	d := Datastore{}
	d.Init()

	// create the entry
	err := d.Create(i)
	if err != nil {
		t.Fatal(err)
	}

	// retrieve the entry
	image, err := d.Retrieve(i.ID)
	if err != nil {
		t.Fatal(err)
	}

	if image.ID != i.ID {
		t.Fatal(err)
	}
}

func TestRetrieveAll(t *testing.T) {
	i := service.Image{
		ID:    "validID",
		State: service.Created,
	}

	d := Datastore{}
	d.Init()

	// create the entry
	err := d.Create(i)
	if err != nil {
		t.Fatal(err)
	}

	// retrieve the entry
	images, err := d.RetrieveAll()
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

func TestDelete(t *testing.T) {
	i := service.Image{
		ID:    "validID",
		State: service.Created,
	}

	d := Datastore{}
	d.Init()

	// create the entry
	err := d.Create(i)
	if err != nil {
		t.Fatal(err)
	}

	// delete the entry
	err = d.Delete(i.ID)
	if err != nil {
		t.Fatal(err)
	}

	// now attempt to retrive the entry
	_, err = d.Retrieve(i.ID)
	if err == nil {
		t.Fatal(err)
	}
}

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
	"errors"
	"io"
	"time"

	"github.com/01org/ciao/openstack/image"
)

// State represents the state of the image.
type State string

const (
	// Created means that an empty image has been created
	Created State = "created"

	// Saving means the image is being saved
	Saving State = "saving"

	// Active means that the image is created, uploaded and ready to use.
	Active State = "active"
)

// Status translate an image state to an openstack image status.
func (state State) Status() image.Status {
	switch state {
	case Created:
		return image.Queued
	case Saving:
		return image.Saving
	case Active:
		return image.Active
	}

	return image.Active
}

// Visibility returns the image visibility
func (i Image) Visibility() image.Visibility {
	if i.TenantID == "" {
		return image.Public
	}
	return image.Private
}

// Type represents the valid image types.
type Type string

const (
	// Raw is the raw image format.
	Raw Type = "raw"

	// QCow is the qcow2 format.
	QCow Type = "qcow2"

	// ISO is the iso format.
	ISO Type = "iso"
)

// Image contains the information that ciao will store about the image
type Image struct {
	ID         string
	State      State
	TenantID   string
	Name       string
	CreateTime time.Time
	Type       Type
}

var (
	// ErrNoImage is returned when an image is not found.
	ErrNoImage = errors.New("Image not found")

	// ErrImageSaving is returned when an image is being uploaded.
	ErrImageSaving = errors.New("Image being uploaded")
)

// MetaDataStore is the metadata storing interface that's used by
// image cache implementation.
type MetaDataStore interface {
	Write(Image) error
	Delete(ID string) error
	GetAll() ([]Image, error)
}

// RawDataStore is the raw data storage interface that's used by the
// image cache implementation.
type RawDataStore interface {
	Write(ID string, body io.Reader) (int64, error)
	Delete(ID string) error
}

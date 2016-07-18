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

package image

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

// TBD - are these thing shared enough between OpenStack services
// to be pulled out to a common area?
// ---------

// VersionStatus defines whether a reported version is supported or not.
type VersionStatus string

const (
	// Deprecated indicates the api deprecates the spec version.
	Deprecated VersionStatus = "DEPRECATED"

	// Supported indicates a spec version is supported by the api
	Supported VersionStatus = "SUPPORTED"

	// Current indicates the current spec version of the api
	// TBD: can this be eliminated? do we need both supported & current?
	Current VersionStatus = "CURRENT"
)

// Link is used by the API to create the link json strings.
type Link struct {
	Href string `json:"href"`
	Type string `json:"type,omitempty"`
	Rel  string `json:"rel,omitempty"`
}

// Version is used by the API to create the version json strings
type Version struct {
	Status VersionStatus `json:"status"`
	ID     string        `json:"id"`
	Links  []Link        `json:"links"`
}

// Versions creates multiple version json strings
type Versions struct {
	Versions []Version `json:"versions"`
}

// --------
// end possible common json

// ImageStatus defines the possible states for an image
type ImageStatus string

const (
	// The image service reserved an image ID for the image but did
	// not yet upload any image data.
	Queued ImageStatus = "queued"

	// the image service is currently uploading the raw data for the image.
	Saving ImageStatus = "saving"

	// The image is active and fully available in the image service
	Active ImageStatus = "active"

	// An image data upload error occurred
	Killed ImageStatus = "killed"

	// The image service retains information abou tthe image but the image
	// is no longer available for use.
	Deleted ImageStatus = "deleted"

	// Similar to the deleted status. An image in this state is not
	// recoverable.
	PendingDelete ImageStatus = "pending_delete"
)

// ImageVisibility defines whether an image is per tenant or public
type ImageVisibility string

const (
	// Public indicates that the image can be used by anyone.
	Public ImageVisibility = "public"

	// Private indicates that the image is only available to a tenant.
	Private ImageVisibility = "private"
)

// ContainerFormat defines the acceptable container format strings.
type ContainerFormat string

const (
	// we support the bare format only
	Bare ContainerFormat = "bare"
)

// DiskFormat defines the valid values for the disk_format string
type DiskFormat string

// we support the following disk formats
const (
	// Raw
	Raw DiskFormat = "raw"

	// QCow
	QCow DiskFormat = "qcow2"

	// ISO
	ISO DiskFormat = "iso"
)

// CreateImageRequest contains information for a create image request.
// http://developer.openstack.org/api-ref-image-v2.html#createImage-v2
type CreateImageRequest struct {
	Name            string          `json:"name,omitempty"`
	ID              string          `json:"id,omitempty"`
	Visibility      ImageVisibility `json:"visibility,omitempty"`
	Tags            []string        `json:"tags,omitempty"`
	ContainerFormat ContainerFormat `json:"container_format,omitempty"`
	DiskFormat      DiskFormat      `json:"disk_format,omitempty"`
	MinDisk         int             `json:"min_disk,omitempty"`
	MinRAM          int             `json:"min_ram,omitempty"`
	Protected       bool            `json:"protected,omitempty"`
	Properties      interface{}     `json:"properties,omitempty"`
}

// CreateImageResponse contains information about a created image
// http://developer.openstack.org/api-ref-image-v2.html#createImage-v2
type CreateImageResponse struct {
	Status          ImageStatus      `json:"status"`
	ContainerFormat *ContainerFormat `json:"container_format"`
	MinRAM          *int             `json:"min_ram"`
	UpdatedAt       *time.Time       `json:"updated_at"`
	Owner           *string          `json:"owner"`
	MinDisk         *int             `json:"min_disk"`
	Tags            []string         `json:"tags"`
	Locations       []string         `json:"locations"`
	Visibility      ImageVisibility  `json:"visibility"`
	ID              string           `json:"id"`
	Size            *int             `json:"size"`
	VirtualSize     *int             `json:"virtual_size"`
	Name            *string          `json:"name"`
	CheckSum        *string          `json:"checksum"`
	CreatedAt       time.Time        `json:"created_at"`
	DiskFormat      DiskFormat       `json:"disk_format"`
	Properties      interface{}      `json:"properties"`
	Protected       bool             `json:"protected"`
	Self            string           `json:"self"`
	File            string           `json:"file"`
	Schema          string           `json:"schema"`
}

// TBD - can we pull these structs out into some sort of common
// api service file?
// ----------

// APIConfig contains information needed to start the block api service.
type APIConfig struct {
	Port         int     // the https port of the block api service
	ImageService Service // the service interface
}

type Service interface {
	CreateImage(CreateImageRequest) (CreateImageResponse, error)
}

// Context contains data and interfaces that the image api will need.
// TBD: do we really need this, or is just a service interface sufficient?
type Context struct {
	port int
	Service
}

// APIResponse is returned from the API handlers.
type APIResponse struct {
	status   int
	response interface{}
}

// APIHandler is a custom handler for the image APIs.
// This custom handler allows us to more cleanly return an error and response,
// and pass some package level context into the handler.
type APIHandler struct {
	*Context
	Handler func(*Context, http.ResponseWriter, *http.Request) (APIResponse, error)
}

// ServeHTTP satisfies the http Handler interface.
// It wraps our api response in json as well.
func (h APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Handler(h.Context, w, r)
	if err != nil {
		http.Error(w, http.StatusText(resp.status), resp.status)
	}

	b, err := json.Marshal(resp.response)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.status)
	w.Write(b)
}

// ---------------
// end possible common service api stuff

// errorResponse maps service error responses to http responses.
// this helper function can help functions avoid having to switch
// on return values all the time.
func errorResponse(err error) APIResponse {
	switch err {
	default:
		return APIResponse{http.StatusInternalServerError, nil}
	}
}

// endpoints
func listAPIVersions(context *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	host, err := os.Hostname()
	if err != nil {
		return APIResponse{http.StatusInternalServerError, nil}, err
	}

	// maybe we should just put href in context
	href := fmt.Sprintf("https://%s:%d/v2/", host, context.port)

	// TBD clean up this code
	var resp Versions

	selfLink := Link{
		Href: href,
		Rel:  "self",
	}

	v := Version{
		Status: Current,
		ID:     "v2.3",
		Links:  []Link{selfLink},
	}

	resp.Versions = append(resp.Versions, v)

	return APIResponse{http.StatusOK, resp}, nil
}

func createImage(context *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return APIResponse{http.StatusBadRequest, nil}, err
	}

	var req CreateImageRequest

	err = json.Unmarshal(body, &req)
	if err != nil {
		return APIResponse{http.StatusInternalServerError, nil}, err
	}

	resp, err := context.CreateImage(req)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusCreated, resp}, nil
}

// Routes provides gorilla mux routes for the supported endpoints.
func Routes(config APIConfig) *mux.Router {
	// make new Context
	context := &Context{config.Port, config.ImageService}

	r := mux.NewRouter()

	// API versions
	r.Handle("/", APIHandler{context, listAPIVersions}).Methods("GET")
	r.Handle("/v2/images", APIHandler{context, createImage}).Methods("POST")

	return r
}

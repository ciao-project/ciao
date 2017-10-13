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

package block

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/gorilla/mux"
)

// MetaData is defined as a set of arbitrary key value structs.
type MetaData interface{}

// RequestedVolume contains information about a volume to be created.
// http://developer.openstack.org/api-ref-blockstorage-v2.html#createVolume
type RequestedVolume struct {
	Size               int      `json:"size"`
	AvailabilityZone   string   `json:"availability_zone,omitempty"`
	SourceVolID        string   `json:"source_volid,omitempty"`
	Description        string   `json:"description,omitempty"`
	MultiAttach        bool     `json:"multiattach"`
	SnapshotID         string   `json:"snapshot_id,omitempty"`
	Name               string   `json:"name,omitempty"`
	ImageRef           string   `json:"imageRef,omitempty"`
	VolumeType         string   `json:"volume_type,omitempty"`
	MetaData           MetaData `json:"metadata"`
	SourceReplica      string   `json:"source_replica,omitempty"`
	ConsistencyGroupID string   `json:"consistencygroup_id,omitempty"`
}

// VolumeCreateRequest is the json request for the createVolume endpoint.
// http://developer.openstack.org/api-ref-blockstorage-v2.html#createVolume
type VolumeCreateRequest struct {
	Volume RequestedVolume `json:"volume"`
}

// These errors can be returned by the Service interface
var (
	ErrQuota                = errors.New("Tenant over quota")
	ErrTenantNotFound       = errors.New("Tenant not found")
	ErrVolumeNotFound       = errors.New("Volume not found")
	ErrInstanceNotFound     = errors.New("Instance not found")
	ErrVolumeNotAvailable   = errors.New("Volume not available")
	ErrVolumeOwner          = errors.New("You are not volume owner")
	ErrInstanceOwner        = errors.New("You are not instance owner")
	ErrInstanceNotAvailable = errors.New("Instance not available")
	ErrVolumeNotAttached    = errors.New("Volume not attached")
)

// errorResponse maps service error responses to http responses.
// this helper function can help functions avoid having to switch
// on return values all the time.
func errorResponse(err error) APIResponse {
	switch err {
	case ErrQuota:
		return APIResponse{http.StatusForbidden, nil}
	case ErrTenantNotFound:
		return APIResponse{http.StatusNotFound, nil}
	case ErrVolumeNotFound:
		return APIResponse{http.StatusNotFound, nil}
	case ErrInstanceNotFound:
		return APIResponse{http.StatusNotFound, nil}
	case ErrVolumeNotAvailable,
		ErrVolumeNotAvailable,
		ErrVolumeOwner,
		ErrInstanceOwner,
		ErrInstanceNotAvailable,
		ErrVolumeNotAttached:
		return APIResponse{http.StatusForbidden, nil}
	default:
		return APIResponse{http.StatusInternalServerError, nil}
	}
}

// APIConfig contains information needed to start the block api service.
type APIConfig struct {
	VolService Service // the service interface
}

// Service contains the required interface to the block service.
// The caller who is starting the api service needs to provide this
// interface.
type Service interface {
	CreateVolume(tenant string, req RequestedVolume) (types.BlockData, error)
	DeleteVolume(tenant string, volume string) error
	AttachVolume(tenant string, volume string, instance string, mountpoint string) error
	DetachVolume(tenant string, volume string, attachment string) error
	ListVolumesDetail(tenant string) ([]types.BlockData, error)
	ShowVolumeDetails(tenant string, volume string) (types.BlockData, error)
}

// Context contains data and interfaces that the block api will need.
// TBD: do we really need this, or is just a service interface sufficient?
type Context struct {
	Service
}

// APIResponse is returned from the API handlers.
type APIResponse struct {
	status   int
	response interface{}
}

// APIHandler is a custom handler for the block APIs.
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
		http.Error(w, err.Error(), resp.status)
	}

	b, err := json.Marshal(resp.response)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.status)
	w.Write(b)
}

func createVolume(bc *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]

	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return APIResponse{http.StatusBadRequest, nil}, err
	}

	var req VolumeCreateRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		return APIResponse{http.StatusInternalServerError, nil}, err
	}

	vol, err := bc.CreateVolume(tenant, req.Volume)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusAccepted, vol}, nil
}

func listVolumesDetail(bc *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]

	// TBD: support sorting and paging

	vols, err := bc.ListVolumesDetail(tenant)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusOK, vols}, nil
}

func showVolumeDetails(bc *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]
	volume := vars["volume_id"]

	vol, err := bc.ShowVolumeDetails(tenant, volume)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusOK, vol}, nil
}

func deleteVolume(bc *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]
	volume := vars["volume_id"]

	// TBD - satisfy preconditions here, or in interface?
	err := bc.DeleteVolume(tenant, volume)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusAccepted, nil}, nil
}

func volumeActionAttach(bc *Context, m map[string]interface{}, tenant string, volume string) (APIResponse, error) {
	val := m["os-attach"]

	m = val.(map[string]interface{})

	val, ok := m["instance_uuid"]
	if !ok {
		// we have to have the instance uuid
		return APIResponse{http.StatusBadRequest, nil}, nil
	}
	instance := val.(string)

	val, ok = m["mountpoint"]
	if !ok {
		// we have to have the mountpoint ?
		return APIResponse{http.StatusBadRequest, nil}, nil
	}
	mountPoint := val.(string)

	err := bc.AttachVolume(tenant, volume, instance, mountPoint)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusAccepted, nil}, nil
}

func volumeActionDetach(bc *Context, m map[string]interface{}, tenant string, volume string) (APIResponse, error) {
	val := m["os-detach"]

	m = val.(map[string]interface{})

	// attachment-id is optional
	var attachment string
	val = m["attachment-id"]
	if val != nil {
		attachment = val.(string)
	}

	err := bc.DetachVolume(tenant, volume, attachment)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusAccepted, nil}, nil
}

func volumeAction(bc *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]
	volume := vars["volume_id"]

	var req interface{}

	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return APIResponse{http.StatusBadRequest, nil}, err
	}

	err = json.Unmarshal(body, &req)
	if err != nil {
		return APIResponse{http.StatusInternalServerError, nil}, err
	}

	m := req.(map[string]interface{})

	// for now, we will support only attach and detach

	if m["os-attach"] != nil {
		return volumeActionAttach(bc, m, tenant, volume)
	}

	if m["os-detach"] != nil {
		return volumeActionDetach(bc, m, tenant, volume)
	}

	return APIResponse{http.StatusBadRequest, nil}, err
}

// Routes provides gorilla mux routes for the supported endpoints.
func Routes(config APIConfig, r *mux.Router) *mux.Router {
	// make new Context
	context := &Context{config.VolService}

	if r == nil {
		r = mux.NewRouter()
	}

	// Volumes
	r.Handle("/v2/{tenant}/volumes",
		APIHandler{context, createVolume}).Methods("POST")
	r.Handle("/v2/{tenant}/volumes",
		APIHandler{context, listVolumesDetail}).Methods("GET")
	r.Handle("/v2/{tenant}/volumes/{volume_id}",
		APIHandler{context, showVolumeDetails}).Methods("GET")
	r.Handle("/v2/{tenant}/volumes/{volume_id}",
		APIHandler{context, deleteVolume}).Methods("DELETE")

	// Volume actions
	r.Handle("/v2/{tenant}/volumes/{volume_id}/action",
		APIHandler{context, volumeAction}).Methods("POST")

	return r
}

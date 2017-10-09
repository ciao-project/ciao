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

package compute

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

// PrivateAddresses contains information about a single instance network
// interface.
type PrivateAddresses struct {
	Addr    string `json:"addr"`
	MacAddr string `json:"mac_addr"`
}

// These errors can be returned by the Service interface
var (
	ErrQuota                = errors.New("Tenant over quota")
	ErrTenantNotFound       = errors.New("Tenant not found")
	ErrServerNotFound       = errors.New("Server not found")
	ErrServerOwner          = errors.New("You are not server owner")
	ErrInstanceNotAvailable = errors.New("Instance not currently available for this operation")
)

// errorResponse maps service error responses to http responses.
// this helper function can help functions avoid having to switch
// on return values all the time.
func errorResponse(err error) APIResponse {
	switch err {
	case ErrTenantNotFound, ErrServerNotFound:
		return APIResponse{http.StatusNotFound, nil}

	case ErrQuota, ErrServerOwner, ErrInstanceNotAvailable:
		return APIResponse{http.StatusForbidden, nil}

	default:
		return APIResponse{http.StatusInternalServerError, nil}
	}
}

// ServerDetails contains information about a specific instance.
type ServerDetails struct {
	PrivateAddresses []PrivateAddresses `json:"private_addresses"`
	Created          time.Time          `json:"created"`
	WorkloadID       string             `json:"workload_id"`
	NodeID           string             `json:"node_id"`
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	Volumes          []string           `json:"volumes"`
	Status           string             `json:"status"`
	TenantID         string             `json:"tenant_id"`
	SSHIP            string             `json:"ssh_ip"`
	SSHPort          int                `json:"ssh_port"`
}

// Servers represents the unmarshalled version of the contents of a
// /v2.1/{tenant}/servers/detail response.  It contains information about a
// set of instances within a ciao cluster.
// http://developer.openstack.org/api-ref/compute/?expanded=list-servers-detailed-detail
// BUG - TotalServers is not specified by the openstack api. We are going
// to pretend it is for now.
type Servers struct {
	TotalServers int             `json:"total_servers"`
	Servers      []ServerDetails `json:"servers"`
}

// NewServers allocates a Servers structure.
// It allocates the Servers slice as well so that the marshalled
// JSON is an empty array and not a nil pointer for, as
// specified by the OpenStack APIs.
func NewServers() (servers Servers) {
	servers.Servers = []ServerDetails{}
	return
}

// Server represents the unmarshalled version of the contents of a
// /v2.1/{tenant}/servers/{server} response.  It contains information about a
// specific instance within a ciao cluster.
type Server struct {
	Server ServerDetails `json:"server"`
}

// BlockDeviceMappingV2 represents an optional block_device_mapping_v2
// object within a /v2.1/{tenant}/servers request POST to "Create Server"
// array of block_device_mapping_v2 objects.
// NOTE: the OpenStack api-ref currently indicates in text that this is an
// object not an array, but given the implementation/usage it is clearly in
// fact an array.  Also volume size and uuid are not documented in the API
// reference, but logically must be included.
type BlockDeviceMappingV2 struct {
	// DeviceName: the name the hypervisor should assign to the block
	// device, eg: "vda"
	DeviceName string `json:"device_name,omitempty"`

	// SourceType: blank, snapshot, volume, or image
	SourceType string `json:"source_type"`

	// DestinationType: optional flag to indicate whether the block
	// device is backed locally or from the volume service
	DestinationType string `json:"destination_type,omitempty"`

	// DeleteOnTermination: optional flag to indicate the volume should
	// autodelete upon termination of the instance
	DeleteOnTermination bool `json:"delete_on_termination,omitempty"`

	// GuestFormat: optionally format a created volume as "swap" or
	// leave "ephemeral" (unformatted) for any use by the instance
	GuestFormat string `json:"guest_format,omitempty"`

	// BootIndex: hint to hypervisor for boot order among multiple
	// bootable devices, eg: floppy, cdrom, disk.  Default "none".
	// Disable booting via negative number or "none"
	BootIndex string `json:"boot_index"`

	// Tag: optional arbitrary text identifier for the block device, useful
	// for human identification or programmatic searching/sorting
	Tag string `json:"tag,omitempty"`

	// UUID: the volume/image/snapshot to attach
	UUID string `json:"uuid,omitempty"`

	// VolumeSize: integer number of gigabytes for ephemeral or swap
	VolumeSize int `json:"volume_size,omitempty"`
}

// CreateServerRequest represents the unmarshalled version of the contents of a
// /v2.1/{tenant}/servers request.  It contains the information needed to start
// one or more instances.
type CreateServerRequest struct {
	Server struct {
		ID                  string                 `json:"id"`
		Name                string                 `json:"name"`
		Image               string                 `json:"imageRef"`
		WorkloadID          string                 `json:"workload_id"`
		MaxInstances        int                    `json:"max_count"`
		MinInstances        int                    `json:"min_count"`
		BlockDeviceMappings []BlockDeviceMappingV2 `json:"block_device_mapping_v2,omitempty"`
		Metadata            map[string]string      `json:"metadata,omitempty"`
	} `json:"server"`
}

// APIConfig contains information needed to start the compute api service.
type APIConfig struct {
	ComputeService Service // the service interface
}

// Service defines the interface required by the compute service.
type Service interface {
	// server interfaces
	CreateServer(string, CreateServerRequest) (interface{}, error)
	ListServersDetail(tenant string) ([]ServerDetails, error)
	ShowServerDetails(tenant string, server string) (Server, error)
	DeleteServer(tenant string, server string) error
	StartServer(tenant string, server string) error
	StopServer(tenant string, server string) error
}

type pagerFilterType uint8

const (
	none pagerFilterType = iota
	workloadFilter
)

type serverPager struct {
	servers []ServerDetails
}

func pagerQueryParse(r *http.Request) (int, int, string) {
	values := r.URL.Query()
	limit := 0
	offset := 0
	marker := ""

	// we only support marker and offset for now.
	if values["marker"] != nil {
		marker = values["marker"][0]
	} else {
		if values["offset"] != nil {
			o, err := strconv.ParseInt(values["offset"][0], 10, 32)
			if err != nil {
				offset = 0
			} else {
				offset = (int)(o)
			}
		}
		if values["limit"] != nil {
			l, err := strconv.ParseInt(values["limit"][0], 10, 32)
			if err != nil {
				limit = 0
			} else {
				limit = (int)(l)
			}
		}
	}

	return limit, offset, marker
}

func (pager *serverPager) getServers(filterType pagerFilterType, filter string, servers []ServerDetails, limit int, offset int) (Servers, error) {
	newServers := NewServers()

	newServers.TotalServers = len(servers)
	pageLength := 0

	glog.V(2).Infof("Get servers limit [%d] offset [%d]", limit, offset)

	if servers == nil || offset >= len(servers) {
		return newServers, nil
	}

	for _, server := range servers[offset:] {
		if filterType != none &&
			pager.filter(filterType, filter, server) {
			continue
		}

		newServers.Servers = append(newServers.Servers, server)
		pageLength++
		if limit > 0 && pageLength >= limit {
			break
		}
	}

	return newServers, nil
}

func (pager *serverPager) filter(filterType pagerFilterType, filter string, server ServerDetails) bool {
	// we only support filtering by workload right now
	switch filterType {
	case workloadFilter:
		if server.WorkloadID != filter {
			return true
		}
	}

	return false
}

func (pager *serverPager) nextPage(filterType pagerFilterType, filter string, r *http.Request) (Servers, error) {
	limit, offset, lastSeen := pagerQueryParse(r)

	glog.V(2).Infof("Next page marker [%s] limit [%d] offset [%d]",
		lastSeen, limit, offset)

	if lastSeen == "" {
		if limit != 0 {
			return pager.getServers(filterType, filter,
				pager.servers, limit, offset)
		}

		return pager.getServers(filterType, filter, pager.servers,
			0, offset)
	}

	for i, server := range pager.servers {
		if server.ID == lastSeen {
			if i >= len(pager.servers)-1 {
				return pager.getServers(filterType, filter,
					nil, limit, 0)
			}

			return pager.getServers(filterType, filter,
				pager.servers[i+1:], limit, 0)
		}
	}

	return Servers{}, fmt.Errorf("Item %s not found", lastSeen)
}

type action uint8

const (
	computeActionStart action = iota
	computeActionStop
)

func dumpRequestBody(r *http.Request, body bool) {
	if glog.V(2) {
		dump, err := httputil.DumpRequest(r, body)
		if err != nil {
			glog.Errorf("HTTP request dump error %s", err)
		}

		glog.Infof("HTTP request [%q]", dump)
	}
}

// DumpRequest will dump an http request if log level is 2
func DumpRequest(r *http.Request) {
	dumpRequestBody(r, false)
}

// HTTPErrorData represents the HTTP response body for
// a compute API request error.
type HTTPErrorData struct {
	Code    int    `json:"code"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

// HTTPReturnErrorCode represents the unmarshalled version for Return codes
// when a API call is made and you need to return explicit data of
// the call as OpenStack format
// http://developer.openstack.org/api-guide/compute/faults.html
type HTTPReturnErrorCode struct {
	Error HTTPErrorData `json:"error"`
}

// Context contains information needed by the compute API service
type Context struct {
	Service
}

// APIResponse is returned from all compute API functions.
// It contains the http status and response to be marshalled if needed.
type APIResponse struct {
	Status   int
	Response interface{}
}

// APIHandler is a custom handler for the compute APIs.
// This custom handler allows us to more cleanly return an error and response,
// and pass some package level context into the handler.
type APIHandler struct {
	*Context
	Handler func(*Context, http.ResponseWriter, *http.Request) (APIResponse, error)
}

// ServeHTTP satisfies the interface for the http Handler.
// If the individual handler returns an error, then it will marshal
// an error response.
func (h APIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Handler(h.Context, w, r)
	if err != nil {
		data := HTTPErrorData{
			Code:    resp.Status,
			Name:    http.StatusText(resp.Status),
			Message: err.Error(),
		}

		code := HTTPReturnErrorCode{
			Error: data,
		}

		b, err := json.Marshal(code)
		if err != nil {
			http.Error(w, http.StatusText(resp.Status), resp.Status)
		}

		http.Error(w, string(b), resp.Status)
		return
	}

	b, err := json.Marshal(resp.Response)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.Status)
	w.Write(b)
}

func createServer(c *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {

	vars := mux.Vars(r)
	tenant := vars["tenant"]

	DumpRequest(r)

	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return APIResponse{http.StatusBadRequest, nil}, err
	}

	var req CreateServerRequest

	err = json.Unmarshal(body, &req)
	if err != nil {
		return APIResponse{http.StatusBadRequest, nil}, err
	}

	resp, err := c.CreateServer(tenant, req)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusAccepted, resp}, nil
}

// ListServersDetails provides server details by tenant or by workload.
// This function is exported for use by ciao-controller due to legacy
// endpoint using the "workload" option. It is simpler to just overload
// this function than to reimplement the legacy code.
//
func ListServersDetails(c *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]

	values := r.URL.Query()

	var workload string

	// if this function is called via an admin context, we might
	// have {workload} on the URL. If it's called from a user context,
	// we might have workload as a query value.
	workload, ok := vars["workload"]
	if !ok {
		if values["workload"] != nil {
			workload = values["workload"][0]
		}
	}

	DumpRequest(r)

	servers, err := c.ListServersDetail(tenant)
	if err != nil {
		return errorResponse(err), err
	}

	pager := serverPager{servers: servers}
	filterType := none
	filter := ""
	if workload != "" {
		filterType = workloadFilter
		filter = workload
	}

	resp, err := pager.nextPage(filterType, filter, r)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusOK, resp}, nil
}

func showServerDetails(c *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]
	server := vars["server"]

	DumpRequest(r)

	resp, err := c.ShowServerDetails(tenant, server)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusOK, resp}, nil
}

func deleteServer(c *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]
	server := vars["server"]

	DumpRequest(r)

	err := c.DeleteServer(tenant, server)
	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusNoContent, nil}, nil
}

func serverAction(c *Context, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]
	server := vars["server"]

	DumpRequest(r)

	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return APIResponse{http.StatusBadRequest, nil}, err
	}

	bodyString := string(body)

	var action action

	if strings.Contains(bodyString, "os-start") {
		action = computeActionStart
	} else if strings.Contains(bodyString, "os-stop") {
		action = computeActionStop
	} else {
		return APIResponse{http.StatusServiceUnavailable, nil},
			errors.New("Unsupported Action")
	}

	switch action {
	case computeActionStart:
		err = c.StartServer(tenant, server)
	case computeActionStop:
		err = c.StopServer(tenant, server)
	}

	if err != nil {
		return errorResponse(err), err
	}

	return APIResponse{http.StatusAccepted, nil}, nil
}

// Routes returns a gorilla mux router for the compute endpoints.
func Routes(config APIConfig, r *mux.Router) *mux.Router {
	context := &Context{config.ComputeService}

	if r == nil {
		r = mux.NewRouter()
	}

	// servers endpoints
	r.Handle("/v2.1/{tenant}/servers",
		APIHandler{context, createServer}).Methods("POST")
	r.Handle("/v2.1/{tenant}/servers/detail",
		APIHandler{context, ListServersDetails}).Methods("GET")
	r.Handle("/v2.1/{tenant}/servers/{server}",
		APIHandler{context, showServerDetails}).Methods("GET")
	r.Handle("/v2.1/{tenant}/servers/{server}",
		APIHandler{context, deleteServer}).Methods("DELETE")
	r.Handle("/v2.1/{tenant}/servers/{server}/action",
		APIHandler{context, serverAction}).Methods("POST")

	return r
}

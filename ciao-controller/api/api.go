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

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/ciao-project/ciao/service"
	"github.com/ciao-project/ciao/uuid"
	"github.com/gorilla/mux"
)

// Port is the default port number for the ciao API.
const Port = 8889

const (
	// PoolsV1 is the content-type string for v1 of our pools resource
	PoolsV1 = "x.ciao.pools.v1"

	// ExternalIPsV1 is the content-type string for v1 of our external-ips resource
	ExternalIPsV1 = "x.ciao.external-ips.v1"

	// WorkloadsV1 is the content-type string for v1 of our workloads resource
	WorkloadsV1 = "x.ciao.workloads.v1"

	// TenantsV1 is the content-type string for v1 of our tenants resource
	TenantsV1 = "x.ciao.tenants.v1"

	// NodeV1 is the content-type string for v1 of our node resource
	NodeV1 = "x.ciao.node.v1"

	// ImagesV1 is the content-type string for v1 of our images resource
	ImagesV1 = "x.ciao.images.v1"

	// VolumesV1 is the content-type string for v1 of our volumes resource
	VolumesV1 = "x.ciao.volumes.v1"
)

// ErrorImage defines all possible image handling errors
type ErrorImage error

var (
	// ErrNoImage is returned when an image is not found.
	ErrNoImage = errors.New("Image not found")

	// ErrImageSaving is returned when an image is being uploaded.
	ErrImageSaving = errors.New("Image being uploaded")

	// ErrBadUUID is returned when an invalid UUID is specified
	ErrBadUUID = errors.New("Bad UUID")

	// ErrAlreadyExists is returned when an attempt is made to add
	// an image with a UUID that already exists.
	ErrAlreadyExists = errors.New("Already Exists")

	// ErrQuota is returned when the tenant exceeds its quota
	ErrQuota = errors.New("Tenant over quota")
)

// CreateImageRequest contains information for a create image request.
type CreateImageRequest struct {
	Name       string           `json:"name,omitempty"`
	ID         string           `json:"id,omitempty"`
	Visibility types.Visibility `json:"visibility,omitempty"`
}

// RequestedVolume contains information about a volume to be created.
type RequestedVolume struct {
	Size        int    `json:"size"`
	SourceVolID string `json:"source_volid,omitempty"`
	Description string `json:"description,omitempty"`
	Name        string `json:"name,omitempty"`
	ImageRef    string `json:"imageRef,omitempty"`
}

var (
	//ErrInstanceNotFound is used if instance not found
	ErrInstanceNotFound = errors.New("Instance not found")

	// ErrVolumeNotAvailable returned if volume not available
	ErrVolumeNotAvailable = errors.New("Volume not available")

	// ErrVolumeOwner returned if permission denied
	ErrVolumeOwner = errors.New("You are not volume owner")

	// ErrVolumeNotAttached returned if volume not attached
	ErrVolumeNotAttached = errors.New("Volume not attached")
)

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

// Response contains the http status and any response struct to be marshalled.
type Response struct {
	status   int
	response interface{}
}

func errorResponse(err error) Response {
	switch err {
	case types.ErrPoolNotFound,
		types.ErrTenantNotFound,
		types.ErrAddressNotFound,
		types.ErrInstanceNotFound,
		types.ErrWorkloadNotFound:
		return Response{http.StatusNotFound, nil}

	case types.ErrQuota,
		types.ErrInstanceNotAssigned,
		types.ErrDuplicateSubnet,
		types.ErrDuplicateIP,
		types.ErrInvalidIP,
		types.ErrPoolNotEmpty,
		types.ErrInvalidPoolAddress,
		types.ErrBadRequest,
		types.ErrPoolEmpty,
		types.ErrDuplicatePoolName,
		types.ErrWorkloadInUse:
		return Response{http.StatusForbidden, nil}

	default:
		return Response{http.StatusInternalServerError, nil}
	}
}

// Handler is a custom handler for the compute APIs.
// This custom handler allows us to more cleanly return an error and response,
// and pass some package level context into the handler.
type Handler struct {
	*Context
	Handler    func(*Context, http.ResponseWriter, *http.Request) (Response, error)
	Privileged bool
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// check whether we should send permission denied for this route.
	if h.Privileged {
		privileged := service.GetPrivilege(r.Context())
		if !privileged {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
	}

	// set the content type to whatever was requested.
	contentType := r.Header.Get("Content-Type")

	resp, err := h.Handler(h.Context, w, r)
	if err != nil {
		data := HTTPErrorData{
			Code:    resp.status,
			Name:    http.StatusText(resp.status),
			Message: err.Error(),
		}

		code := HTTPReturnErrorCode{
			Error: data,
		}

		b, err := json.Marshal(code)
		if err != nil {
			http.Error(w, http.StatusText(resp.status), resp.status)
			return
		}

		http.Error(w, string(b), resp.status)
		return
	}

	b, err := json.Marshal(resp.response)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(resp.status)
	w.Write(b)
}

func listResources(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	var links []types.APILink
	vars := mux.Vars(r)
	tenantID, ok := vars["tenant"]

	// we support the "pools" resource.
	link := types.APILink{
		Rel:        "pools",
		Version:    PoolsV1,
		MinVersion: PoolsV1,
	}

	if !ok {
		link.Href = fmt.Sprintf("%s/pools", c.URL)
	} else {
		link.Href = fmt.Sprintf("%s/%s/pools", c.URL, tenantID)
	}

	links = append(links, link)

	// we support the "external-ips" resource
	link = types.APILink{
		Rel:        "external-ips",
		Version:    ExternalIPsV1,
		MinVersion: ExternalIPsV1,
	}

	if !ok {
		link.Href = fmt.Sprintf("%s/external-ips", c.URL)
	} else {
		link.Href = fmt.Sprintf("%s/%s/external-ips", c.URL, tenantID)
	}

	links = append(links, link)

	// we support the "workloads" resource
	link = types.APILink{
		Rel:        "workloads",
		Version:    WorkloadsV1,
		MinVersion: WorkloadsV1,
	}

	if !ok {
		link.Href = fmt.Sprintf("%s/workloads", c.URL)
	} else {
		link.Href = fmt.Sprintf("%s/%s/workloads", c.URL, tenantID)
	}

	links = append(links, link)

	// for the "tenants" resource
	link = types.APILink{
		Rel:        "tenants",
		Version:    TenantsV1,
		MinVersion: TenantsV1,
	}

	if !ok {
		link.Href = fmt.Sprintf("%s/tenants", c.URL)
	} else {
		link.Href = fmt.Sprintf("%s/%s/tenants", c.URL, tenantID)
	}

	links = append(links, link)

	// for the "node" resource
	if !ok {
		link = types.APILink{
			Rel:        "node",
			Version:    NodeV1,
			MinVersion: NodeV1,
		}

		link.Href = fmt.Sprintf("%s/node", c.URL)
		links = append(links, link)
	}

	// for the "images" resource
	link = types.APILink{
		Rel:        "images",
		Version:    ImagesV1,
		MinVersion: ImagesV1,
	}

	if !ok {
		link.Href = fmt.Sprintf("%s/images", c.URL)
	} else {
		link.Href = fmt.Sprintf("%s/%s/images", c.URL, tenantID)
	}

	links = append(links, link)

	// for the "volumes" resource
	if ok {
		link = types.APILink{
			Rel:        "volumes",
			Version:    VolumesV1,
			MinVersion: VolumesV1,
		}

		link.Href = fmt.Sprintf("%s/%s/volumes", c.URL, tenantID)
		links = append(links, link)
	}

	return Response{http.StatusOK, links}, nil
}

func showPool(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	ID := vars["pool"]

	pool, err := c.ShowPool(ID)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusOK, pool}, nil
}

func listPools(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	var resp types.ListPoolsResponse
	vars := mux.Vars(r)
	_, ok := vars["tenant"]

	pools, err := c.ListPools()
	if err != nil {
		return errorResponse(err), err
	}

	queries := r.URL.Query()

	names, returnNamedPool := queries["name"]

	var match bool
	for i, p := range pools {
		if returnNamedPool == true {
			for _, name := range names {
				if name == p.Name {
					match = true
				}
			}
		} else {
			match = true
		}

		if match {
			summary := types.PoolSummary{
				ID:   p.ID,
				Name: p.Name,
			}

			if !ok {
				summary.TotalIPs = &pools[i].TotalIPs
				summary.Free = &pools[i].Free
				summary.Links = pools[i].Links
			}

			resp.Pools = append(resp.Pools, summary)
		}
	}

	if returnNamedPool && !match {
		return Response{http.StatusNotFound, nil}, types.ErrPoolNotFound
	}

	return Response{http.StatusOK, resp}, err
}

func addPool(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	var req types.NewPoolRequest

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errorResponse(err), err
	}

	err = json.Unmarshal(body, &req)
	if err != nil {
		return errorResponse(err), err
	}

	var ips []string

	for _, ip := range req.IPs {
		ips = append(ips, ip.IP)
	}

	_, err = c.AddPool(req.Name, req.Subnet, ips)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusNoContent, nil}, nil
}

func deletePool(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	ID := vars["pool"]

	err := c.DeletePool(ID)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusNoContent, nil}, nil
}

func addToPool(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	ID := vars["pool"]

	var req types.NewAddressRequest

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errorResponse(err), err
	}

	err = json.Unmarshal(body, &req)
	if err != nil {
		return errorResponse(err), err
	}

	var ips []string

	for _, ip := range req.IPs {
		ips = append(ips, ip.IP)
	}

	err = c.AddAddress(ID, req.Subnet, ips)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusNoContent, nil}, nil
}

func deleteSubnet(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	poolID := vars["pool"]
	subnetID := vars["subnet"]

	err := c.RemoveAddress(poolID, &subnetID, nil)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusNoContent, nil}, nil
}

func deleteExternalIP(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	poolID := vars["pool"]
	IPID := vars["ip_id"]

	err := c.RemoveAddress(poolID, nil, &IPID)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusNoContent, nil}, nil
}

func listMappedIPs(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	tenantID, ok := vars["tenant"]
	var IPs []types.MappedIP
	var short []types.MappedIPShort

	if !ok {
		IPs = c.ListMappedAddresses(nil)
		return Response{http.StatusOK, IPs}, nil
	}

	IPs = c.ListMappedAddresses(&tenantID)
	for _, IP := range IPs {
		s := types.MappedIPShort{
			ID:         IP.ID,
			ExternalIP: IP.ExternalIP,
			InternalIP: IP.InternalIP,
			InstanceID: IP.InstanceID,
			Links:      IP.Links,
		}
		short = append(short, s)
	}

	return Response{http.StatusOK, short}, nil
}

func mapExternalIP(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	var req types.MapIPRequest

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errorResponse(err), err
	}

	err = json.Unmarshal(body, &req)
	if err != nil {
		return errorResponse(err), err
	}

	tenantID := vars["tenant"]

	err = c.MapAddress(tenantID, req.PoolName, req.InstanceID)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusNoContent, nil}, nil
}

func unmapExternalIP(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	tenantID, ok := vars["tenant"]
	mappingID := vars["mapping_id"]

	var IPs []types.MappedIP

	if !ok {
		IPs = c.ListMappedAddresses(nil)
	} else {
		IPs = c.ListMappedAddresses(&tenantID)
	}

	for _, m := range IPs {
		if m.ID == mappingID {
			err := c.UnMapAddress(m.ExternalIP)
			if err != nil {
				return errorResponse(err), err
			}

			return Response{http.StatusAccepted, nil}, nil
		}
	}

	return errorResponse(types.ErrAddressNotFound), types.ErrAddressNotFound
}

func addWorkload(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	var req types.Workload

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errorResponse(err), err
	}

	err = json.Unmarshal(body, &req)
	if err != nil {
		return errorResponse(err), err
	}

	// we allow admin to create public workloads for any tenant. However,
	// users scoped to a particular tenant may only create workloads
	// for their own tenant.
	vars := mux.Vars(r)
	tenantID, ok := vars["tenant"]
	if ok {
		req.TenantID = tenantID
	} else {
		req.TenantID = "public"
	}

	wl, err := c.CreateWorkload(req)
	if err != nil {
		return errorResponse(err), err
	}

	var ref string

	if ok {
		ref = fmt.Sprintf("%s/%s/workloads/%s", c.URL, tenantID, wl.ID)
	} else {
		ref = fmt.Sprintf("%s/workloads/%s", c.URL, wl.ID)
	}

	link := types.Link{
		Rel:  "self",
		Href: ref,
	}

	resp := types.WorkloadResponse{
		Workload: wl,
		Link:     link,
	}

	return Response{http.StatusCreated, resp}, nil
}

func deleteWorkload(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	ID := vars["workload_id"]

	// if we have no tenant variable, then we are admin
	tenantID, ok := vars["tenant"]
	if !ok {
		tenantID = "public"
	}

	err := c.DeleteWorkload(tenantID, ID)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusNoContent, nil}, nil
}

func showWorkload(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	ID := vars["workload_id"]

	// if we have no tenant variable, then we are admin
	tenant, ok := vars["tenant"]
	if !ok {
		tenant = "public"
	}

	wl, err := c.ShowWorkload(tenant, ID)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusOK, wl}, nil
}

func listWorkloads(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)

	// if we have no tenant variable, then we are admin
	tenant, ok := vars["tenant"]
	if !ok {
		tenant = "public"
	}

	wls, err := c.ListWorkloads(tenant)
	if err != nil {
		return errorResponse(err), err
	}
	return Response{http.StatusOK, wls}, nil
}

func listQuotas(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	tenantID, ok := vars["tenant"]

	if !ok {
		tenantID = vars["for_tenant"]
	}

	var resp types.QuotaListResponse
	resp.Quotas = c.ListQuotas(tenantID)

	return Response{http.StatusOK, resp}, nil
}

func updateQuotas(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	tenantID := vars["for_tenant"]

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errorResponse(err), err
	}

	var req types.QuotaUpdateRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		return errorResponse(err), err
	}

	err = c.UpdateQuotas(tenantID, req.Quotas)
	if err != nil {
		return errorResponse(err), err
	}

	var resp types.QuotaListResponse
	resp.Quotas = c.ListQuotas(tenantID)

	return Response{http.StatusCreated, resp}, nil
}

func changeNodeStatus(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	ID := vars["node_id"]

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errorResponse(err), err
	}

	var status types.CiaoNodeStatus
	err = json.Unmarshal(body, &status)
	if err != nil {
		return errorResponse(err), err
	}

	if status.Status == types.NodeStatusReady {
		err = c.RestoreNode(ID)
	} else if status.Status == types.NodeStatusMaintenance {
		err = c.EvacuateNode(ID)
	} else {
		err = fmt.Errorf("Cannot transition node %s to %s",
			ID, status.Status)
	}

	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusNoContent, nil}, nil
}

func listTenants(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	var resp types.TenantsListResponse

	queries := r.URL.Query()
	IDs, returnSingleTenant := queries["id"]

	tenants, err := c.ListTenants()
	if err != nil {
		return errorResponse(err), err
	}

	if returnSingleTenant != true {
		resp.Tenants = tenants
		return Response{http.StatusOK, resp}, nil
	}

	for _, t := range tenants {
		for _, tenantID := range IDs {
			if t.ID == tenantID {
				resp.Tenants = append(resp.Tenants, t)
			}
		}
	}

	return Response{http.StatusOK, resp}, nil
}

func showTenant(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	ID := vars["tenant"]

	resp, err := c.ShowTenant(ID)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusOK, resp}, nil
}

func updateTenant(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	ID := vars["tenant"]

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errorResponse(err), err
	}

	err = c.PatchTenant(ID, body)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusNoContent, nil}, nil
}

func createTenant(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errorResponse(err), err
	}

	var req types.TenantRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		return errorResponse(err), err
	}

	resp, err := c.CreateTenant(req.ID, req.Config)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusCreated, resp}, nil
}

func deleteTenant(c *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	ID := vars["tenant"]

	err := c.DeleteTenant(ID)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusNoContent, nil}, nil
}

func validPrivilege(visibility types.Visibility, privileged bool) bool {
	return visibility == types.Private || (visibility == types.Public || visibility == types.Internal) && privileged
}

// createImage creates information about an image, but doesn't contain
// any actual image.
func createImage(context *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	defer r.Body.Close()

	vars := mux.Vars(r)
	tenantID := vars["tenant"]

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return Response{http.StatusBadRequest, nil}, err
	}

	var req CreateImageRequest

	err = json.Unmarshal(body, &req)
	if err != nil {
		return Response{http.StatusInternalServerError, nil}, err
	}

	privileged := service.GetPrivilege(r.Context())

	if !validPrivilege(req.Visibility, privileged) {
		return Response{http.StatusForbidden, nil}, nil
	}

	resp, err := context.CreateImage(tenantID, req)

	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusCreated, resp}, nil
}

// listImages returns a list of all created images.
//
// TBD: support query & sort parameters
func listImages(context *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	tenantID, ok := vars["tenant"]
	if !ok {
		tenantID = "admin"
	}

	images, err := context.ListImages(tenantID)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusOK, images}, nil
}

// getImage get information about an image by image_id field
//
func getImage(context *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	imageID := vars["image_id"]

	tenantID, ok := vars["tenant"]
	if !ok {
		tenantID = "admin"
	}

	image, err := context.GetImage(tenantID, imageID)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusOK, image}, nil
}

func uploadImage(context *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	imageID := vars["image_id"]

	tenantID, ok := vars["tenant"]
	if !ok {
		tenantID = "admin"
	}

	err := context.UploadImage(tenantID, imageID, r.Body)
	if err != nil {
		return errorResponse(err), err
	}
	return Response{http.StatusNoContent, nil}, nil
}

func deleteImage(context *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	imageID := vars["image_id"]

	tenantID, ok := vars["tenant"]
	if !ok {
		tenantID = "admin"
	}

	err := context.DeleteImage(tenantID, imageID)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusNoContent, nil}, nil
}

func createVolume(bc *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]

	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return Response{http.StatusBadRequest, nil}, err
	}

	var req RequestedVolume
	err = json.Unmarshal(body, &req)
	if err != nil {
		return Response{http.StatusInternalServerError, nil}, err
	}

	vol, err := bc.CreateVolume(tenant, req)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusAccepted, vol}, nil
}

func listVolumesDetail(bc *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]

	vols, err := bc.ListVolumesDetail(tenant)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusOK, vols}, nil
}

func showVolumeDetails(bc *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]
	volume := vars["volume_id"]

	vol, err := bc.ShowVolumeDetails(tenant, volume)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusOK, vol}, nil
}

func deleteVolume(bc *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]
	volume := vars["volume_id"]

	// TBD - satisfy preconditions here, or in interface?
	err := bc.DeleteVolume(tenant, volume)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusAccepted, nil}, nil
}

func volumeActionAttach(bc *Context, m map[string]interface{}, tenant string, volume string) (Response, error) {
	val := m["attach"]

	m = val.(map[string]interface{})

	val, ok := m["instance_uuid"]
	if !ok {
		// we have to have the instance uuid
		return Response{http.StatusBadRequest, nil}, nil
	}
	instance := val.(string)

	val, ok = m["mountpoint"]
	if !ok {
		// we have to have the mountpoint ?
		return Response{http.StatusBadRequest, nil}, nil
	}
	mountPoint := val.(string)

	err := bc.AttachVolume(tenant, volume, instance, mountPoint)
	if err != nil {
		return errorResponse(err), err
	}

	return Response{http.StatusAccepted, nil}, nil
}

func volumeActionDetach(bc *Context, m map[string]interface{}, tenant string, volume string) (Response, error) {
	val := m["detach"]

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

	return Response{http.StatusAccepted, nil}, nil
}

func volumeAction(bc *Context, w http.ResponseWriter, r *http.Request) (Response, error) {
	vars := mux.Vars(r)
	tenant := vars["tenant"]
	volume := vars["volume_id"]

	var req interface{}

	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return Response{http.StatusBadRequest, nil}, err
	}

	err = json.Unmarshal(body, &req)
	if err != nil {
		return Response{http.StatusInternalServerError, nil}, err
	}

	m := req.(map[string]interface{})

	// for now, we will support only attach and detach

	if m["attach"] != nil {
		return volumeActionAttach(bc, m, tenant, volume)
	}

	if m["detach"] != nil {
		return volumeActionDetach(bc, m, tenant, volume)
	}

	return Response{http.StatusBadRequest, nil}, err
}

// Service is an interface which must be implemented by the ciao API context.
type Service interface {
	AddPool(name string, subnet *string, ips []string) (types.Pool, error)
	ListPools() ([]types.Pool, error)
	ShowPool(id string) (types.Pool, error)
	DeletePool(id string) error
	AddAddress(poolID string, subnet *string, IPs []string) error
	RemoveAddress(poolID string, subnetID *string, IPID *string) error
	ListMappedAddresses(tenantID *string) []types.MappedIP
	MapAddress(tenantID string, poolName *string, instanceID string) error
	UnMapAddress(ID string) error
	CreateWorkload(req types.Workload) (types.Workload, error)
	DeleteWorkload(tenantID string, workloadID string) error
	ShowWorkload(tenantID string, workloadID string) (types.Workload, error)
	ListWorkloads(tenantID string) ([]types.Workload, error)
	ListQuotas(tenantID string) []types.QuotaDetails
	UpdateQuotas(tenantID string, qds []types.QuotaDetails) error
	EvacuateNode(nodeID string) error
	RestoreNode(nodeID string) error
	ListTenants() ([]types.TenantSummary, error)
	ShowTenant(ID string) (types.TenantConfig, error)
	PatchTenant(ID string, patch []byte) error
	CreateTenant(ID string, config types.TenantConfig) (types.TenantSummary, error)
	DeleteTenant(ID string) error
	CreateImage(string, CreateImageRequest) (types.Image, error)
	UploadImage(string, string, io.Reader) error
	ListImages(string) ([]types.Image, error)
	GetImage(string, string) (types.Image, error)
	DeleteImage(string, string) error
	CreateVolume(tenant string, req RequestedVolume) (types.Volume, error)
	DeleteVolume(tenant string, volume string) error
	AttachVolume(tenant string, volume string, instance string, mountpoint string) error
	DetachVolume(tenant string, volume string, attachment string) error
	ListVolumesDetail(tenant string) ([]types.Volume, error)
	ShowVolumeDetails(tenant string, volume string) (types.Volume, error)
}

// Context is used to provide the services and current URL to the handlers.
type Context struct {
	URL string
	Service
}

// Config is used to setup the Context for the ciao API.
type Config struct {
	URL         string
	CiaoService Service
}

// Routes returns the supported ciao API endpoints.
// A plain application/json request will return v1 of the resource
// since we only have one version of this api so far, that means
// most routes will match both json as well as our custom
// content type.
func Routes(config Config, r *mux.Router) *mux.Router {
	// make new Context
	context := &Context{config.URL, config.CiaoService}

	if r == nil {
		r = mux.NewRouter()
	}

	// external IP pools
	route := r.Handle("/", Handler{context, listResources, true})
	route.Methods("GET")

	route = r.Handle("/{tenant:"+uuid.UUIDRegex+"}", Handler{context, listResources, false})
	route.Methods("GET")

	matchContent := fmt.Sprintf("application/(%s|json)", PoolsV1)

	route = r.Handle("/pools", Handler{context, listPools, true})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/{tenant:"+uuid.UUIDRegex+"}/pools", Handler{context, listPools, false})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/pools", Handler{context, addPool, true})
	route.Methods("POST")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/pools/{pool:"+uuid.UUIDRegex+"}", Handler{context, showPool, true})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/pools/{pool:"+uuid.UUIDRegex+"}", Handler{context, deletePool, true})
	route.Methods("DELETE")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/pools/{pool:"+uuid.UUIDRegex+"}", Handler{context, addToPool, true})
	route.Methods("POST")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/pools/{pool:"+uuid.UUIDRegex+"}/subnets/{subnet:"+uuid.UUIDRegex+"}", Handler{context, deleteSubnet, true})
	route.Methods("DELETE")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/pools/{pool:"+uuid.UUIDRegex+"}/external-ips/{ip_id:"+uuid.UUIDRegex+"}", Handler{context, deleteExternalIP, true})
	route.Methods("DELETE")
	route.HeadersRegexp("Content-Type", matchContent)

	// mapped external IPs
	matchContent = fmt.Sprintf("application/(%s|json)", ExternalIPsV1)

	route = r.Handle("/external-ips", Handler{context, listMappedIPs, true})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/{tenant:"+uuid.UUIDRegex+"}/external-ips", Handler{context, listMappedIPs, false})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/external-ips", Handler{context, mapExternalIP, true})
	route.Methods("POST")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/{tenant:"+uuid.UUIDRegex+"}/external-ips", Handler{context, mapExternalIP, false})
	route.Methods("POST")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/external-ips/{mapping_id:"+uuid.UUIDRegex+"}", Handler{context, unmapExternalIP, true})
	route.Methods("DELETE")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/{tenant:"+uuid.UUIDRegex+"}/external-ips/{mapping_id:"+uuid.UUIDRegex+"}", Handler{context, unmapExternalIP, false})
	route.Methods("DELETE")
	route.HeadersRegexp("Content-Type", matchContent)

	// workloads
	matchContent = fmt.Sprintf("application/(%s|json)", WorkloadsV1)

	route = r.Handle("/workloads", Handler{context, addWorkload, true})
	route.Methods("POST")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/workloads", Handler{context, listWorkloads, true})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/workloads/{workload_id:"+uuid.UUIDRegex+"}", Handler{context, deleteWorkload, true})
	route.Methods("DELETE")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/workloads/{workload_id:"+uuid.UUIDRegex+"}", Handler{context, showWorkload, true})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/{tenant:"+uuid.UUIDRegex+"}/workloads", Handler{context, addWorkload, false})
	route.Methods("POST")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/{tenant:"+uuid.UUIDRegex+"}/workloads", Handler{context, listWorkloads, false})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/{tenant:"+uuid.UUIDRegex+"}/workloads/{workload_id:"+uuid.UUIDRegex+"}", Handler{context, deleteWorkload, false})
	route.Methods("DELETE")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/{tenant:"+uuid.UUIDRegex+"}/workloads/{workload_id:"+uuid.UUIDRegex+"}", Handler{context, showWorkload, false})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	// tenants
	matchContent = fmt.Sprintf("application/(%s|json)", TenantsV1)

	route = r.Handle("/tenants", Handler{context, listTenants, true})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/tenants", Handler{context, createTenant, true})
	route.Methods("POST")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/tenants/{tenant:"+uuid.UUIDRegex+"}", Handler{context, showTenant, true})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/tenants/{tenant:"+uuid.UUIDRegex+"}", Handler{context, deleteTenant, true})
	route.Methods("DELETE")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/{tenant:"+uuid.UUIDRegex+"}/tenants", Handler{context, showTenant, false})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/tenants/{tenant:"+uuid.UUIDRegex+"}", Handler{context, updateTenant, true})
	route.Methods("PATCH")
	route.HeadersRegexp("Content-Type", `application/merge-patch\+json`)

	// tenant quotas
	route = r.Handle("/{tenant:"+uuid.UUIDRegex+"}/tenants/quotas", Handler{context, listQuotas, false})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/tenants/{for_tenant:"+uuid.UUIDRegex+"}/quotas", Handler{context, listQuotas, true})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/tenants/{for_tenant:"+uuid.UUIDRegex+"}/quotas", Handler{context, updateQuotas, true})
	route.Methods("PUT")
	route.HeadersRegexp("Content-Type", matchContent)

	// evacuation and restore
	matchContent = fmt.Sprintf("application/(%s|json)", NodeV1)

	route = r.Handle("/node/{node_id:"+uuid.UUIDRegex+"}", Handler{context, changeNodeStatus, true})
	route.Methods("PUT")
	route.HeadersRegexp("Content-Type", matchContent)

	// images
	matchContent = fmt.Sprintf("application/(%s|json)", ImagesV1)

	route = r.Handle("/{tenant}/images", Handler{context, createImage, false})
	route.Methods("POST")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/{tenant}/images/{image_id:"+uuid.UUIDRegex+"}/file", Handler{context, uploadImage, false})
	route.Methods("PUT")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/{tenant}/images", Handler{context, listImages, false})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/{tenant}/images/{image_id:"+uuid.UUIDRegex+"}", Handler{context, getImage, false})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/{tenant}/images/{image_id:"+uuid.UUIDRegex+"}", Handler{context, deleteImage, false})
	route.Methods("DELETE")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/images", Handler{context, createImage, true})
	route.Methods("POST")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/images/{image_id:"+uuid.UUIDRegex+"}/file", Handler{context, uploadImage, true})
	route.Methods("PUT")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/images", Handler{context, listImages, true})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/images/{image_id:"+uuid.UUIDRegex+"}", Handler{context, getImage, true})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/images/{image_id:"+uuid.UUIDRegex+"}", Handler{context, deleteImage, true})
	route.Methods("DELETE")
	route.HeadersRegexp("Content-Type", matchContent)

	// Volumes
	matchContent = fmt.Sprintf("application/(%s|json)", VolumesV1)
	route = r.Handle("/{tenant}/volumes", Handler{context, createVolume, false})
	route.Methods("POST")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/{tenant}/volumes", Handler{context, listVolumesDetail, false})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/{tenant}/volumes/{volume_id}", Handler{context, showVolumeDetails, false})
	route.Methods("GET")
	route.HeadersRegexp("Content-Type", matchContent)

	route = r.Handle("/{tenant}/volumes/{volume_id}", Handler{context, deleteVolume, false})
	route.Methods("DELETE")
	route.HeadersRegexp("Content-Type", matchContent)

	// Volume actions
	route = r.Handle("/{tenant}/volumes/{volume_id}/action", Handler{context, volumeAction, false})
	route.Methods("POST")
	route.HeadersRegexp("Content-Type", matchContent)

	return r
}

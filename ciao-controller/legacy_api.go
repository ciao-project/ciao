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
	"encoding/json"
	"net/http"

	"github.com/ciao-project/ciao/service"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

// APIHandler is a custom handler for the compute APIs.
// This custom handler allows us to more cleanly return an error and response,
// and pass some package level context into the handler.
type legacyAPIHandler struct {
	*controller
	Handler    func(*controller, http.ResponseWriter, *http.Request) (APIResponse, error)
	Privileged bool
}

func (h legacyAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// check to see if we should send permission denied for this route.
	if h.Privileged {
		privileged := service.GetPrivilege(r.Context())
		if !privileged {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
	}

	resp, err := h.Handler(h.controller, w, r)
	if err != nil {
		data := HTTPErrorData{
			Code:    resp.status,
			Name:    http.StatusText(resp.status),
			Message: err.Error(),
		}

		code := HTTPReturnErrorCode{
			Error: data,
		}

		glog.Warningf("Returning error response to request: %s: %v", r.URL.String(), err)

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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.status)
	_, _ = w.Write(b)
}

func listTenantQuotas(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return getResources(c, w, r)
}

func listTenantResources(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return getUsage(c, w, r)
}

// tenantServersAction will apply the operation sent in POST (as os-start, os-stop, os-delete)
// to all servers of a tenant or if ServersID size is greater than zero it will be applied
// only to the subset provided that also belongs to the tenant
func tenantServersAction(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return serversAction(c, w, r)
}

func legacyListNodes(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return listNodes(c, w, r)
}

func legacyListComputeNodes(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return listComputeNodes(c, w, r)
}

func legacyListNetworkNodes(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return listNetworkNodes(c, w, r)
}

func legacyNodesSummary(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return nodesSummary(c, w, r)
}

func legacyListNodeServers(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return listNodeServers(c, w, r)
}

func legacyListCNCIs(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return listCNCIs(c, w, r)
}

func legacyListCNCIDetails(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return listCNCIDetails(c, w, r)
}

func legacyListTraces(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return listTraces(c, w, r)
}

func legacyListEvents(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return listEvents(c, w, r)
}

func legacyListTenantEvents(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return listEvents(c, w, r)
}

func legacyClearEvents(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return clearEvents(c, w, r)
}

func legacyTraceData(c *controller, w http.ResponseWriter, r *http.Request) (APIResponse, error) {
	return traceData(c, w, r)
}

func legacyComputeRoutes(ctl *controller, r *mux.Router) *mux.Router {
	r.Handle("/v2.1/{tenant}/servers/action",
		legacyAPIHandler{ctl, tenantServersAction, false}).Methods("POST")

	r.Handle("/v2.1/{tenant}/resources",
		legacyAPIHandler{ctl, listTenantResources, false}).Methods("GET")

	r.Handle("/v2.1/{tenant}/quotas",
		legacyAPIHandler{ctl, listTenantQuotas, false}).Methods("GET")

	r.Handle("/v2.1/nodes",
		legacyAPIHandler{ctl, legacyListNodes, true}).Methods("GET")
	r.Handle("/v2.1/nodes/summary",
		legacyAPIHandler{ctl, legacyNodesSummary, true}).Methods("GET")
	r.Handle("/v2.1/nodes/{node}/servers/detail",
		legacyAPIHandler{ctl, legacyListNodeServers, true}).Methods("GET")
	r.Handle("/v2.1/nodes/compute",
		legacyAPIHandler{ctl, legacyListComputeNodes, true}).Methods("GET")
	r.Handle("/v2.1/nodes/network",
		legacyAPIHandler{ctl, legacyListNetworkNodes, true}).Methods("GET")

	r.Handle("/v2.1/cncis",
		legacyAPIHandler{ctl, legacyListCNCIs, true}).Methods("GET")
	r.Handle("/v2.1/cncis/{cnci}/detail",
		legacyAPIHandler{ctl, legacyListCNCIDetails, true}).Methods("GET")

	r.Handle("/v2.1/events",
		legacyAPIHandler{ctl, legacyListEvents, true}).Methods("GET")
	r.Handle("/v2.1/events",
		legacyAPIHandler{ctl, legacyClearEvents, true}).Methods("DELETE")
	r.Handle("/v2.1/{tenant}/events",
		legacyAPIHandler{ctl, legacyListTenantEvents, false}).Methods("GET")

	r.Handle("/v2.1/traces",
		legacyAPIHandler{ctl, legacyListTraces, true}).Methods("GET")
	r.Handle("/v2.1/traces/{label}",
		legacyAPIHandler{ctl, legacyTraceData, true}).Methods("GET")

	return r
}

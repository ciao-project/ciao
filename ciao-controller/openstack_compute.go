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

package main

import (
	"errors"
	"fmt"
	"net/http"
	"sort"

	"github.com/01org/ciao/ciao-controller/types"
	"github.com/01org/ciao/openstack/compute"
	osIdentity "github.com/01org/ciao/openstack/identity"
	"github.com/01org/ciao/payloads"
	"github.com/gorilla/mux"
)

func instanceToServer(ctl *controller, instance *types.Instance) (compute.ServerDetails, error) {
	workload, err := ctl.ds.GetWorkload(instance.WorkloadID)
	if err != nil {
		return compute.ServerDetails{}, err
	}

	imageID := workload.ImageID

	server := compute.ServerDetails{
		HostID:   instance.NodeID,
		ID:       instance.ID,
		TenantID: instance.TenantID,
		Flavor: compute.FlavorLinks{
			ID: instance.WorkloadID,
		},
		Image: compute.Image{
			ID: imageID,
		},
		Status: instance.State,
		Addresses: compute.Addresses{
			Private: []compute.PrivateAddresses{
				{
					Addr:               instance.IPAddress,
					OSEXTIPSMACMacAddr: instance.MACAddress,
				},
			},
		},
		SSHIP:   instance.SSHIP,
		SSHPort: instance.SSHPort,
	}

	return server, nil
}

func (c *controller) CreateServer(tenant string, server compute.CreateServerRequest) (resp interface{}, err error) {
	nInstances := 1

	if server.Server.MaxInstances > 0 {
		nInstances = server.Server.MaxInstances
	} else if server.Server.MinInstances > 0 {
		nInstances = server.Server.MinInstances
	}

	// openstack doesn't allow us to use our traced start workload
	// functionality. So we use the name field in our cli to indicate
	// that we want to trace this workload.
	trace := false
	label := ""
	if server.Server.Name != "" {
		trace = true
		label = server.Server.Name
	}

	instances, err := c.startWorkload(server.Server.Flavor, tenant, nInstances, trace, label)
	if err != nil {
		return server, err
	}

	var servers compute.Servers

	for _, instance := range instances {
		server, err := instanceToServer(c, instance)
		if err != nil {
			return server, err
		}
		servers.Servers = append(servers.Servers, server)
	}

	servers.TotalServers = len(instances)

	// set machine ID for OpenStack compatibility
	server.Server.ID = instances[0].ID

	// builtServers is define to meet OpenStack compatibility on result
	// format and keep CIAOs legacy behavior.
	builtServers := struct {
		compute.CreateServerRequest
		compute.Servers
	}{
		compute.CreateServerRequest{
			Server: server.Server,
		},
		compute.Servers{
			TotalServers: servers.TotalServers,
			Servers:      servers.Servers,
		},
	}

	return builtServers, nil
}

func (c *controller) ListServersDetail(tenant string) ([]compute.ServerDetails, error) {
	var servers []compute.ServerDetails
	var err error
	var instances []*types.Instance

	if tenant != "" {
		instances, err = c.ds.GetAllInstancesFromTenant(tenant)
	} else {
		instances, err = c.ds.GetAllInstances()
	}

	if err != nil {
		return servers, err
	}

	sort.Sort(types.SortedInstancesByID(instances))

	for _, instance := range instances {
		server, err := instanceToServer(c, instance)
		if err != nil {
			continue
		}

		servers = append(servers, server)
	}

	return servers, nil
}

func (c *controller) ShowServerDetails(tenant string, server string) (compute.Server, error) {
	var s compute.Server

	instance, err := c.ds.GetInstance(server)
	if err != nil {
		return s, err
	}

	if instance.TenantID != tenant {
		return s, compute.ErrServerOwner
	}

	s.Server, err = instanceToServer(c, instance)
	if err != nil {
		return s, err
	}

	return s, nil
}

func (c *controller) DeleteServer(tenant string, server string) error {
	/* First check that the instance belongs to this tenant */
	i, err := c.ds.GetInstance(server)
	if err != nil {
		return compute.ErrServerNotFound
	}

	if i.TenantID != tenant {
		return compute.ErrServerOwner
	}

	err = c.deleteInstance(server)
	if err != nil {
		return err
	}

	return nil
}

func (c *controller) StartServer(tenant string, ID string) error {
	i, err := c.ds.GetInstance(ID)
	if err != nil {
		return err
	}

	if i.TenantID != tenant {
		return compute.ErrServerOwner
	}

	return c.restartInstance(ID)
}

func (c *controller) StopServer(tenant string, ID string) error {
	i, err := c.ds.GetInstance(ID)
	if err != nil {
		return err
	}

	if i.TenantID != tenant {
		return compute.ErrServerOwner
	}

	return c.stopInstance(ID)
}

func (c *controller) ListFlavors(tenant string) (compute.Flavors, error) {
	flavors := compute.NewComputeFlavors()

	// we are ignoring tenant for now
	workloads, err := c.ds.GetWorkloads()
	if err != nil {
		return flavors, err
	}

	for _, workload := range workloads {
		flavors.Flavors = append(flavors.Flavors,
			struct {
				ID    string         `json:"id"`
				Links []compute.Link `json:"links"`
				Name  string         `json:"name"`
			}{
				ID:   workload.ID,
				Name: workload.Description,
			},
		)
	}

	return flavors, nil
}

func buildFlavorDetails(workload *types.Workload) (compute.FlavorDetails, error) {
	var details compute.FlavorDetails

	defaults := workload.Defaults
	if len(defaults) == 0 {
		return details, fmt.Errorf("Workload resources not set")
	}

	details.OsFlavorAccessIsPublic = true
	details.ID = workload.ID
	details.Disk = workload.ImageID
	details.Name = workload.Description

	for r := range defaults {
		switch defaults[r].Type {
		case payloads.VCPUs:
			details.Vcpus = defaults[r].Value
		case payloads.MemMB:
			details.RAM = defaults[r].Value
		}
	}

	return details, nil
}

func (c *controller) ListFlavorsDetail(tenant string) (compute.FlavorsDetails, error) {
	flavors := compute.NewComputeFlavorsDetails()

	// we ignore tenant for now

	workloads, err := c.ds.GetWorkloads()
	if err != nil {
		return flavors, err
	}

	for _, workload := range workloads {
		details, err := buildFlavorDetails(workload)
		if err != nil {
			continue
		}

		flavors.Flavors = append(flavors.Flavors, details)
	}

	return flavors, nil
}

func (c *controller) ShowFlavorDetails(tenant string, flavorID string) (compute.Flavor, error) {
	var flavor compute.Flavor

	workload, err := c.ds.GetWorkload(flavorID)
	if err != nil {
		return flavor, err
	}

	flavor.Flavor, err = buildFlavorDetails(workload)
	if err != nil {
		return flavor, err
	}

	return flavor, nil
}

// Start will get the Compute API endpoints from the OpenStack compute api,
// then wrap them in keystone validation. It will then start the https
// service.
func (c *controller) startComputeService() error {
	config := compute.APIConfig{Port: computeAPIPort, ComputeService: c}

	r := compute.Routes(config)
	if r == nil {
		return errors.New("Unable to start Compute Service")
	}

	// we add on some ciao specific routes for legacy purposes
	// using the openstack compute port.
	r = legacyComputeRoutes(c, r)

	// setup identity for these routes.
	validServices := []osIdentity.ValidService{
		{ServiceType: "compute", ServiceName: "ciao"},
		{ServiceType: "compute", ServiceName: "nova"},
	}

	validAdmins := []osIdentity.ValidAdmin{
		{Project: "service", Role: "admin"},
		{Project: "admin", Role: "admin"},
	}

	err := r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		h := osIdentity.Handler{
			Client:        c.id.scV3,
			Next:          route.GetHandler(),
			ValidServices: validServices,
			ValidAdmins:   validAdmins,
		}

		route.Handler(h)

		return nil
	})

	if err != nil {
		return err
	}

	// start service.
	service := fmt.Sprintf(":%d", computeAPIPort)

	return http.ListenAndServeTLS(service, httpsCAcert, httpsKey, r)
}

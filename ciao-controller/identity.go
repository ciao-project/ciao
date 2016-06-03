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
	"errors"

	"github.com/golang/glog"
	"github.com/mitchellh/mapstructure"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	v3tokens "github.com/rackspace/gophercloud/openstack/identity/v3/tokens"
)

type identity struct {
	scV3 *gophercloud.ServiceClient
}

type identityConfig struct {
	endpoint        string
	serviceUserName string
	servicePassword string
}

// Project holds project information extracted from the keystone response.
type Project struct {
	ID   string `mapstructure:"id"`
	Name string `mapstructure:"name"`
}

// RoleEntry contains the name of a role extracted from the keystone response.
type RoleEntry struct {
	Name string `mapstructure:"name"`
}

// Roles contains a list of role names extracted from the keystone response.
type Roles struct {
	Entries []RoleEntry
}

// Endpoint contains endpoint information extracted from the keystone response.
type Endpoint struct {
	ID        string `mapstructure:"id"`
	Region    string `mapstructure:"region"`
	Interface string `mapstructure:"interface"`
	URL       string `mapstructure:"url"`
}

// ServiceEntry contains information about a service extracted from the keystone response.
type ServiceEntry struct {
	ID        string     `mapstructure:"id"`
	Name      string     `mapstructure:"name"`
	Type      string     `mapstructure:"type"`
	Endpoints []Endpoint `mapstructure:"endpoints"`
}

// Services is a list of ServiceEntry structs
// These structs contain information about the services keystone knows about.
type Services struct {
	Entries []ServiceEntry
}

type getResult struct {
	v3tokens.GetResult
}

// extractProject
// Ideally we would actually contribute this functionality
// back to the gophercloud project, but for now we extend
// their object to allow us to get project information out
// of the response from the GET token validation request.
func (r getResult) extractProject() (*Project, error) {
	if r.Err != nil {
		glog.V(2).Info(r.Err)
		return nil, r.Err
	}

	// can there be more than one project?  You need to test.
	var response struct {
		Token struct {
			ValidProject Project `mapstructure:"project"`
		} `mapstructure:"token"`
	}

	err := mapstructure.Decode(r.Body, &response)
	if err != nil {
		glog.V(2).Info(err)
		return nil, err
	}

	return &Project{
		ID:   response.Token.ValidProject.ID,
		Name: response.Token.ValidProject.Name,
	}, nil
}

func (r getResult) extractServices() (*Services, error) {
	if r.Err != nil {
		glog.V(2).Info(r.Err)
		return nil, r.Err
	}

	var response struct {
		Token struct {
			Entries []ServiceEntry `mapstructure:"catalog"`
		} `mapstructure:"token"`
	}

	err := mapstructure.Decode(r.Body, &response)
	if err != nil {
		glog.Errorf(err.Error())
		return nil, err
	}

	return &Services{Entries: response.Token.Entries}, nil
}

// extractRole
// Ideally we would actually contribute this functionality
// back to the gophercloud project, but for now we extend
// their object to allow us to get project information out
// of the response from the GET token validation request.
func (r getResult) extractRoles() (*Roles, error) {
	if r.Err != nil {
		glog.V(2).Info(r.Err)
		return nil, r.Err
	}

	var response struct {
		Token struct {
			ValidRoles []RoleEntry `mapstructure:"roles"`
		} `mapstructure:"token"`
	}

	err := mapstructure.Decode(r.Body, &response)
	if err != nil {
		glog.V(2).Info(err)
		return nil, err
	}

	return &Roles{Entries: response.Token.ValidRoles}, nil
}

// validateServices
// Validates that a given user belonging to a tenant
// can access a service specified by its type and name.
func (i *identity) validateService(token string, tenantID string, serviceType string, serviceName string) bool {
	r := v3tokens.Get(i.scV3, token)
	result := getResult{r}

	p, err := result.extractProject()
	if err != nil {
		return false
	}

	if p.ID != tenantID {
		glog.Errorf("expected %s got %s\n", tenantID, p.ID)
		return false
	}

	services, err := result.extractServices()
	if err != nil {
		return false
	}

	for _, e := range services.Entries {
		if e.Type == serviceType {
			if serviceName == "" {
				return true
			}

			if e.Name == serviceName {
				return true
			}
		}
	}

	return false
}

func (i *identity) validateProjectRole(token string, project string, role string) bool {
	r := v3tokens.Get(i.scV3, token)
	result := getResult{r}
	p, err := result.extractProject()
	if err != nil {
		return false
	}

	if project != "" && p.Name != project {
		return false
	}

	roles, err := result.extractRoles()
	if err != nil {
		return false
	}

	for i := range roles.Entries {
		if roles.Entries[i].Name == role {
			return true
		}
	}
	return false
}

func newIdentityClient(config identityConfig) (*identity, error) {
	opt := gophercloud.AuthOptions{
		IdentityEndpoint: config.endpoint + "/v3/",
		Username:         config.serviceUserName,
		Password:         config.servicePassword,
		TenantName:       "service",
		DomainID:         "default",
		AllowReauth:      true,
	}
	provider, err := openstack.AuthenticatedClient(opt)
	if err != nil {
		return nil, err
	}

	v3client := openstack.NewIdentityV3(provider)
	if v3client == nil {
		return nil, errors.New("Unable to get keystone V3 client")
	}

	id := &identity{
		scV3: v3client,
	}

	return id, err
}

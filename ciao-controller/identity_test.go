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
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gorilla/mux"
	"github.com/rackspace/gophercloud"
)

func authHandler(w http.ResponseWriter, r *http.Request) {
	token := `
		{
			"token": {
				"methods": [
					"password"
				],
				"roles": [
					{
						"id" : "12345",
						"name" : "admin"
					}
				],
				"expires_at": "%s",
				"project": {
					"domain": {
						"id": "default",
						"name": "Default"
					},
					"id": "%s",
					"name": "admin"
				},
				"catalog": [
					{
						"endpoints": [
							{
								"region_id": "RegionOne",
								"url": "%s/v3",
								"region": "RegionOne",
								"interface": "public",
								"id": "068d1b359ee84b438266cb736d81de97"
							},
							{
								"region_id": "RegionOne",
								"url": "%s/v3",
								"region": "RegionOne",
								"interface": "admin",
								"id": "8bfc846841ab441ca38471be6d164ced"
							},
							{
								"region_id": "RegionOne",
								"url": "%s/v3",
								"region": "RegionOne",
								"interface": "internal",
								"id": "beb6d358c3654b4bada04d4663b640b9"
							}
						],
						"type": "identity",
						"id": "050726f278654128aba89757ae25950c",
						"name": "keystone"
					}
				],
			       "extras": {},
			       "user": {
				       "domain": {
				               "id": "default",
				               "name": "Default"
				        },
				       "id": "ee4dfb6e5540447cb3741905149d9b6e",
			               "name": "admin"
			        },
			        "audit_ids": [
				        "3T2dc1CGQxyJsHdDu1xkcw"
			        ],
			        "issued_at": "%s"
			}
		}`

	t := []byte(fmt.Sprintf(token,
		time.Now().Add(1*time.Hour).Format(gophercloud.RFC3339Milli),
		computeTestUser, testIdentityURL, testIdentityURL,
		testIdentityURL, time.Now().Format(gophercloud.RFC3339Milli)))
	w.Header().Set("X-Subject-Token", "imavalidtoken")
	w.WriteHeader(http.StatusCreated)
	w.Write(t)
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
	tenantURL := computeURL + "/v2.1/" + computeTestUser
	token := `
	{
		"token": {
		        "methods": [
		                "token"
		        ],
		        "expires_at": "%s",
		        "extras": {},
		        "user": {
				"domain": {
					"id": "default",
					"name": "Default"
				},
				"id": "10a2e6e717a245d9acad3e5f97aeca3d",
				"name": "admin"
			},
			"roles": [
				{
					"id" : "12345",
					"name" : "admin"
				}
			],
			"project": {
				"domain": {
					"id": "default",
					"name": "Default"
				},
				"id": "%s",
				"name": "admin"
			},
			"catalog": [
				{
					"endpoints": [
						{
							"region_id": "RegionOne",
							"url": "%s/v3",
							"region": "RegionOne",
							"interface": "public",
							"id": "068d1b359ee84b438266cb736d81de97"
						},
						{
							"region_id": "RegionOne",
							"url": "%s/v3",
							"region": "RegionOne",
							"interface": "admin",
							"id": "8bfc846841ab441ca38471be6d164ced"
						},
						{
							"region_id": "RegionOne",
							"url": "%s/v3",
							"region": "RegionOne",
							"interface": "internal",
							"id": "beb6d358c3654b4bada04d4663b640b9"
						}
					],
					"type": "identity",
					"id": "050726f278654128aba89757ae25950c",
					"name": "keystone"
				},
				{
			                "endpoints": [
					         {
							"region_id": "RegionOne",
							"url": "%s",
							"region": "RegionOne",
							"interface": "admin",
							"id": "2511589f262a407bb0071a814a480af4"
						},
						{
							"region_id": "RegionOne",
							"url": "%s",
							"region": "RegionOne",
							"interface": "internal",
							"id": "9cf9209ae4fc4673a7295611001cf0ae"
						},
						{
							"region_id": "RegionOne",
							"url": "%s",
							"region": "RegionOne",
							"interface": "public",
							"id": "d200b2509e1343e3887dcc465b4fa534"
						}
					],
					"type": "compute",
					"id": "a226b3eeb5594f50bf8b6df94636ed28",
					"name": "ciao"
				}
			],
			"audit_ids": [
			        "mAjXQhiYRyKwkB4qygdLVg"
			],
			"issued_at": "%s"
		}
	}`

	t := []byte(fmt.Sprintf(token,
		time.Now().Add(1*time.Hour).Format(gophercloud.RFC3339Milli),
		computeTestUser, testIdentityURL, testIdentityURL,
		testIdentityURL, tenantURL, tenantURL,
		tenantURL, time.Now().Format(gophercloud.RFC3339Milli)))
	w.WriteHeader(http.StatusOK)
	w.Write(t)
}

func identityHandlers() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/v3/auth/tokens", authHandler).Methods("POST")
	r.HandleFunc("/v3/auth/tokens", validateHandler).Methods("GET")

	return r
}

func startIdentityTestServer() *httptest.Server {
	return httptest.NewServer(identityHandlers())
}

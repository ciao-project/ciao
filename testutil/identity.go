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

package testutil

// CiaoAPIPort is the Ciao API port that API services are running on
const CiaoAPIPort = "8889"

// ComputeURL is the compute service URL the testutil identity service will use by default
var ComputeURL = "https://localhost:" + CiaoAPIPort

// VolumeURL is the volume service URL the testutil identity service will use by default
var VolumeURL = "https://localhost:" + CiaoAPIPort

// ImageURL is the image service URL the testutil identity service will use by default
var ImageURL = "https://localhost:" + CiaoAPIPort

// IdentityURL is the URL for the testutil identity service
var IdentityURL string

// ComputeUser is the test user/tenant name the testutil identity service will use by default
var ComputeUser = "f452bbc7-5076-44d5-922c-3b9d2ce1503f"

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
	"flag"

	"github.com/01org/ciao/ciao-image/datastore"
	"github.com/01org/ciao/ciao-image/service"
	"github.com/01org/ciao/openstack/image"
	"github.com/golang/glog"
)

var httpsCAcert = "/etc/pki/ciao/ciao-image-cacert.pem"
var httpsKey = "/etc/pki/ciao/ciao-image-key.pem"
var port = image.APIPort
var logDir = "/var/lib/ciao/logs/ciao-image"
var identity = "https://localhost:35357/"
var userName = "ciao"
var password = "hello"

var identityURL = flag.String("identity", identity, "URL of keystone service")

func init() {
	flag.Parse()

	if *identityURL != identity {
		identity = *identityURL
	}
}

func main() {
	// TBD Select the right datastore interface
	metaDs := &datastore.Noop{}

	config := service.Config{
		Port:             port,
		HTTPSCACert:      httpsCAcert,
		HTTPSKey:         httpsKey,
		DataStore:        nil,
		MetaDataStore:    metaDs,
		IdentityEndpoint: identity,
		Username:         userName,
		Password:         password,
	}

	glog.Fatal(service.Start(config))
}

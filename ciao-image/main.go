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
	"github.com/01org/ciao/database"
	"github.com/01org/ciao/openstack/image"
	"github.com/golang/glog"
)

var httpsCAcert = "/etc/pki/ciao/ciao-image-cacert.pem"
var httpsKey = "/etc/pki/ciao/ciao-image-key.pem"
var port = image.APIPort
var logDir = "/var/lib/ciao/logs/ciao-image"
var identity = "https://localhost:35357/"
var userName = "csr"
var password = "hello"
var mountPoint = "/var/lib/ciao/images"
var dbDir = "/var/lib/ciao/ciao-image/"
var dbFile = "ciao-image.db"

var identityURL = flag.String("identity", identity, "URL of keystone service")

func init() {
	flag.Parse()

	if *identityURL != identity {
		identity = *identityURL
	}
}

func main() {
	metaDs := &datastore.MetaDs{
		DbProvider: database.NewBoltDBProvider(),
		DbDir:      dbDir,
		DbFile:     dbFile,
	}
	metaDsTables := []string{"images"}

	err := metaDs.DbInit(metaDs.DbDir, metaDs.DbFile)
	if err != nil {
		glog.Fatalf("Error on DB Initialization:%v ", err)
	}
	defer metaDs.DbClose()

	err = metaDs.DbTablesInit(metaDsTables)
	if err != nil {
		glog.Fatalf("Error on DB Tables Initialization:%v ", err)
	}

	rawDs := &datastore.Posix{
		MountPoint: mountPoint,
	}

	config := service.Config{
		Port:             port,
		HTTPSCACert:      httpsCAcert,
		HTTPSKey:         httpsKey,
		RawDataStore:     rawDs,
		MetaDataStore:    metaDs,
		IdentityEndpoint: identity,
		Username:         userName,
		Password:         password,
	}

	glog.Fatal(service.Start(config))
}

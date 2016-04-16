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
	"flag"
	datastore "github.com/01org/ciao/ciao-controller/internal/datastore"
	"github.com/01org/ciao/ssntp"
	"github.com/golang/glog"
	"os"
	"sync"
)

type controller struct {
	client *ssntpClient
	ds     *datastore.Datastore
	id     *identity
}

var cert = flag.String("cert", "/etc/pki/ciao/cert-client-localhost.pem", "Client certificate")
var caCert = flag.String("cacert", "/etc/pki/ciao/CAcert-server-localhost.pem", "CA certificate")
var serverURL = flag.String("url", "localhost", "Server URL")
var identityURL = flag.String("identity", "identity:35357", "Keystone URL")
var serviceUser = flag.String("username", "csr", "Openstack Service Username")
var servicePassword = flag.String("password", "", "Openstack Service Username")
var port = flag.Int("port", 8889, "http port")
var computeAPIPort = flag.Int("computeport", openstackComputeAPIPort, "Openstack Compute API port")
var httpsCAcert = flag.String("httpscert", "/etc/pki/ciao/ciao-controller-cacert.pem", "HTTPS CA certificate")
var httpsKey = flag.String("httpskey", "/etc/pki/ciao/ciao-controller-key.pem", "HTTPS cert key")
var tablesInitPath = flag.String("tables_init_path", ".", "path to csv files")
var workloadsPath = flag.String("workloads_path", ".", "path to yaml files")
var noNetwork = flag.Bool("nonetwork", false, "Debug with no networking")
var debugUI = flag.Bool("debug_ui", true, "Create Debug web UI")
var persistentDatastoreLocation = flag.String("database_path", "./ciao-controller.db", "path to persistent database")
var transientDatastoreLocation = flag.String("stats_path", "/tmp/ciao-controller-stats.db", "path to stats database")
var logDir = "/var/lib/ciao/logs/controller"

func init() {
	flag.Parse()

	logDirFlag := flag.Lookup("log_dir")
	if logDirFlag == nil {
		glog.Errorf("log_dir does not exist")
		return
	}

	if logDirFlag.Value.String() == "" {
		logDirFlag.Value.Set(logDir)
	}

	if err := os.MkdirAll(logDirFlag.Value.String(), 0755); err != nil {
		glog.Errorf("Unable to create log directory (%s) %v", logDir, err)
		return
	}
}

func main() {
	var wg sync.WaitGroup
	var err error

	context := new(controller)
	context.ds = new(datastore.Datastore)

	err = context.ds.Connect(*persistentDatastoreLocation, *transientDatastoreLocation)
	if err != nil {
		glog.Fatalf("unable to connect to datastore: %s", err)
		return
	}

	err = context.ds.Init(*tablesInitPath, *workloadsPath)
	if err != nil {
		glog.Fatalf("unable to Init datastore: %s", err)
		return
	}

	config := &ssntp.Config{
		URI:    *serverURL,
		CAcert: *caCert,
		Cert:   *cert,
		Role:   ssntp.Controller,
		Log:    ssntp.Log,
	}

	context.client, err = newSSNTPClient(context, config)
	if err != nil {
		// spawn some retry routine?
		glog.Fatalf("unable to connect to SSNTP server")
		return
	}

	idConfig := identityConfig{
		endpoint:        *identityURL,
		serviceUserName: *serviceUser,
		servicePassword: *servicePassword,
	}

	context.id, err = newIdentityClient(idConfig)
	if err != nil {
		glog.Fatal("Unable to authenticate to Keystone: ", err)
		return
	}

	if *debugUI {
		wg.Add(1)
		go createDebugInterface(context)
	}

	wg.Add(1)
	go createComputeAPI(context)

	wg.Wait()
	context.ds.Disconnect()
	context.client.Disconnect()
}

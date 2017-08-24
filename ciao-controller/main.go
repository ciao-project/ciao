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
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/01org/ciao/ciao-controller/api"
	"github.com/01org/ciao/ciao-controller/internal/datastore"
	"github.com/01org/ciao/ciao-controller/internal/quotas"
	storage "github.com/01org/ciao/ciao-storage"
	"github.com/01org/ciao/clogger/gloginterface"
	"github.com/01org/ciao/database"
	osIdentity "github.com/01org/ciao/openstack/identity"
	"github.com/01org/ciao/osprepare"
	"github.com/01org/ciao/service"
	"github.com/01org/ciao/ssntp"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

type tenantConfirmMemo struct {
	ch  chan struct{}
	err error
}

type controller struct {
	storage.BlockDriver
	client              controllerClient
	ds                  *datastore.Datastore
	is                  *ImageService
	id                  *identity
	apiURL              string
	tenantReadiness     map[string]*tenantConfirmMemo
	tenantReadinessLock sync.Mutex
	qs                  *quotas.Quotas
	httpServers         []*http.Server
}

var cert = flag.String("cert", "", "Client certificate")
var caCert = flag.String("cacert", "", "CA certificate")
var serverURL = flag.String("url", "", "Server URL")
var identityURL = "identity:35357"
var serviceUser = "csr"
var servicePassword = ""
var controllerAPIPort = api.Port
var httpsCAcert = "/etc/pki/ciao/ciao-controller-cacert.pem"
var httpsKey = "/etc/pki/ciao/ciao-controller-key.pem"
var workloadsPath = flag.String("workloads_path", "/var/lib/ciao/data/controller/workloads", "path to yaml files")
var persistentDatastoreLocation = flag.String("database_path", "/var/lib/ciao/data/controller/ciao-controller.db", "path to persistent database")
var imageDatastoreLocation = flag.String("image_database_path", "/var/lib/ciao/data/image/ciao-image.db", "path to image persistent database")
var logDir = "/var/lib/ciao/logs/controller"

var clientCertCAPath = ""

var imagesPath = flag.String("images_path", "/var/lib/ciao/images", "path to ciao images")

var cephID = flag.String("ceph_id", "", "ceph client id")

var adminSSHKey = ""

// default password set to "ciao"
var adminPassword = "$6$rounds=4096$w9I3hR4g/hu$AnYjaC2DfznbPSG3vxsgtgAS4mJwWBkcR74Y/KHNB5OsfAlA4gpU5j6CHWMOkkt9j.9d7OYJXJ4icXHzKXTAO."

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

	ctl := new(controller)
	ctl.tenantReadiness = make(map[string]*tenantConfirmMemo)
	ctl.ds = new(datastore.Datastore)
	ctl.qs = new(quotas.Quotas)
	ctl.is = new(ImageService)

	dsConfig := datastore.Config{
		PersistentURI:     "file:" + *persistentDatastoreLocation,
		TransientURI:      "file:transient?mode=memory&cache=shared",
		InitWorkloadsPath: *workloadsPath,
	}

	err = ctl.ds.Init(dsConfig)
	if err != nil {
		glog.Fatalf("unable to Init datastore: %s", err)
		return
	}

	ctl.qs.Init()
	populateQuotasFromDatastore(ctl.qs, ctl.ds)

	config := &ssntp.Config{
		URI:    *serverURL,
		CAcert: *caCert,
		Cert:   *cert,
		Log:    ssntp.Log,
	}

	ctl.client, err = newSSNTPClient(ctl, config)
	if err != nil {
		// spawn some retry routine?
		glog.Fatalf("unable to connect to SSNTP server")
		return
	}

	ssntpClient := ctl.client.ssntpClient()
	clusterConfig, err := ssntpClient.ClusterConfiguration()
	if err != nil {
		glog.Fatalf("Unable to retrieve Cluster Configuration: %v", err)
		return
	}

	controllerAPIPort = clusterConfig.Configure.Controller.CiaoPort
	httpsCAcert = clusterConfig.Configure.Controller.HTTPSCACert
	httpsKey = clusterConfig.Configure.Controller.HTTPSKey
	identityURL = clusterConfig.Configure.IdentityService.URL
	serviceUser = clusterConfig.Configure.Controller.IdentityUser
	servicePassword = clusterConfig.Configure.Controller.IdentityPassword
	if *cephID == "" {
		*cephID = clusterConfig.Configure.Storage.CephID
	}

	cnciVCPUs := clusterConfig.Configure.Controller.CNCIVcpus
	cnciMem := clusterConfig.Configure.Controller.CNCIMem
	cnciDisk := clusterConfig.Configure.Controller.CNCIDisk

	adminSSHKey = clusterConfig.Configure.Controller.AdminSSHKey

	if clusterConfig.Configure.Controller.AdminPassword != "" {
		adminPassword = clusterConfig.Configure.Controller.AdminPassword
	}

	if clusterConfig.Configure.Controller.ClientAuthCACertPath != "" {
		clientCertCAPath = clusterConfig.Configure.Controller.ClientAuthCACertPath
	}

	if err := ctl.is.Init(ctl.qs); err != nil {
		glog.Fatalf("Error initialising image service: %v", err)
	}

	ctl.ds.GenerateCNCIWorkload(cnciVCPUs, cnciMem, cnciDisk, adminSSHKey, adminPassword)

	database.Logger = gloginterface.CiaoGlogLogger{}

	logger := gloginterface.CiaoGlogLogger{}
	osprepare.Bootstrap(context.TODO(), logger)
	osprepare.InstallDeps(context.TODO(), controllerDeps, logger)

	idConfig := identityConfig{
		endpoint:        identityURL,
		serviceUserName: serviceUser,
		servicePassword: servicePassword,
	}

	ctl.BlockDriver = func() storage.BlockDriver {
		driver := storage.CephDriver{
			ID: *cephID,
		}
		return driver
	}()

	ctl.id, err = newIdentityClient(idConfig)
	if err != nil {
		glog.Fatal("Unable to authenticate to Keystone: ", err)
		return
	}

	err = initializeCNCICtrls(ctl)
	if err != nil {
		glog.Fatal("Unable to initialize CNCI controllers: ", err)
		return
	}

	host := clusterConfig.Configure.Controller.ControllerFQDN
	if host == "" {
		host, _ = os.Hostname()
	}
	ctl.apiURL = fmt.Sprintf("https://%s:%d", host, controllerAPIPort)

	server, err := ctl.createCiaoServer()
	if err != nil {
		glog.Fatalf("Error creating ciao server: %v", err)
	}
	ctl.httpServers = append(ctl.httpServers, server)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-signalCh
		glog.Warningf("Received signal: %s", s)
		ctl.ShutdownHTTPServers()
		shutdownCNCICtrls(ctl)
	}()

	for _, server := range ctl.httpServers {
		wg.Add(1)
		go func(server *http.Server) {
			if err := server.ListenAndServeTLS(httpsCAcert, httpsKey); err != http.ErrServerClosed {
				glog.Errorf("Error from HTTP server: %v", err)
			}
			wg.Done()
		}(server)
	}

	wg.Wait()
	glog.Warning("Controller shutdown initiated")
	ctl.qs.Shutdown()
	ctl.ds.Exit()
	ctl.is.ds.Shutdown()
	ctl.client.Disconnect()
}

type clientCertAuthHandler struct {
	Next http.Handler
}

func (h *clientCertAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(r.TLS.VerifiedChains) != 1 {
		http.Error(w, "Unexpected number of certificate chains presented", http.StatusUnauthorized)
		return
	}

	certs := r.TLS.VerifiedChains[0]
	cert := certs[0]
	tenants := cert.Subject.Organization

	privileged := false
	if len(tenants) == 1 && tenants[0] == "admin" {
		privileged = true
	}

	r = r.WithContext(service.SetPrivilege(r.Context(), true))

	vars := mux.Vars(r)
	tenantFromVars := vars["tenant"]
	if !privileged {
		tenantMatched := false
		for i := range tenants {
			if tenants[i] == tenantFromVars {
				tenantMatched = true
				break
			}
		}
		if !tenantMatched {
			http.Error(w, "Access to tenant not permitted with certificate", http.StatusUnauthorized)
			return
		}
	}

	r = r.WithContext(service.SetTenantID(r.Context(), tenantFromVars))
	h.Next.ServeHTTP(w, r)
}

func (c *controller) createCiaoRoutes(r *mux.Router) error {
	config := api.Config{URL: c.apiURL, CiaoService: c}

	r = api.Routes(config, r)

	// wrap each route in keystone validation.
	validServices := []osIdentity.ValidService{
		{ServiceType: "compute", ServiceName: "ciao"},
		{ServiceType: "compute", ServiceName: "nova"},
		{ServiceType: "image", ServiceName: "glance"},
		{ServiceType: "image", ServiceName: "ciao"},
		{ServiceType: "volume", ServiceName: "ciao"},
		{ServiceType: "volumev2", ServiceName: "ciao"},
		{ServiceType: "volume", ServiceName: "cinder"},
		{ServiceType: "volumev2", ServiceName: "cinderv2"},
	}

	validAdmins := []osIdentity.ValidAdmin{
		{Project: "service", Role: "admin"},
		{Project: "admin", Role: "admin"},
	}

	err := r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		if clientCertCAPath != "" {
			h := &clientCertAuthHandler{
				Next: route.GetHandler(),
			}
			route.Handler(h)
		} else {
			osHandler := osIdentity.Handler{
				Client:        c.id.scV3,
				Next:          route.GetHandler(),
				ValidServices: validServices,
				ValidAdmins:   validAdmins,
			}

			route.Handler(osHandler)
		}

		return nil
	})

	return err
}

func (c *controller) createCiaoServer() (*http.Server, error) {
	r := mux.NewRouter()

	if err := c.createComputeRoutes(r); err != nil {
		return nil, errors.Wrap(err, "Error adding compute routes")
	}

	if err := c.createImageRoutes(r); err != nil {
		return nil, errors.Wrap(err, "Error adding image routes")
	}

	if err := c.createVolumeRoutes(r); err != nil {
		return nil, errors.Wrap(err, "Error adding volume routes")
	}

	err := c.createCiaoRoutes(r)
	if err != nil {
		return nil, errors.Wrap(err, "Error adding ciao routes")
	}

	addr := fmt.Sprintf(":%d", controllerAPIPort)

	server := &http.Server{
		Handler: r,
		Addr:    addr,
	}

	if clientCertCAPath != "" {
		clientCertCAbytes, err := ioutil.ReadFile(clientCertCAPath)
		if err != nil {
			return nil, errors.Wrap(err, "Error loading client cert CA")
		}
		certPool := x509.NewCertPool()
		ok := certPool.AppendCertsFromPEM(clientCertCAbytes)
		if !ok {
			return nil, errors.New("Error importing client auth CA to poool")
		}
		tlsConfig := tls.Config{
			ClientAuth: tls.RequireAndVerifyClientCert,
			ClientCAs:  certPool,
		}
		server.TLSConfig = &tlsConfig
	}

	return server, nil
}

func (c *controller) ShutdownHTTPServers() {
	glog.Warning("Shutting down HTTP servers")
	var wg sync.WaitGroup
	for _, server := range c.httpServers {
		wg.Add(1)
		go func(server *http.Server) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			err := server.Shutdown(ctx)
			if err != nil {
				glog.Errorf("Error during HTTP server shutdown")
			}
			wg.Done()
		}(server)
	}
	wg.Wait()
}

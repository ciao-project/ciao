// Copyright (c) 2017 Intel Corporation
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
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/service"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

type clientCertAuthHandler struct {
	Controller *controller
	Next       http.Handler
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
	if tenantFromVars != "" {
		err := h.Controller.confirmTenant(tenantFromVars)
		if err != nil {
			http.Error(w, "Error confirming tenant", http.StatusInternalServerError)
		}
	}

	h.Next.ServeHTTP(w, r)
}

func (c *controller) createCiaoRoutes(r *mux.Router) error {
	config := api.Config{URL: c.apiURL, CiaoService: c}

	r = api.Routes(config, r)

	err := r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		h := &clientCertAuthHandler{
			Next:       route.GetHandler(),
			Controller: c,
		}
		route.Handler(h)

		return nil
	})

	return err
}

func (c *controller) createCiaoServer() (*http.Server, error) {
	r := mux.NewRouter()

	addr := fmt.Sprintf(":%d", controllerAPIPort)

	server := &http.Server{
		Handler: r,
		Addr:    addr,
	}

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

	if err := c.createComputeRoutes(r); err != nil {
		return nil, errors.Wrap(err, "Error adding compute routes")
	}

	err = c.createCiaoRoutes(r)
	if err != nil {
		return nil, errors.Wrap(err, "Error adding ciao routes")
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

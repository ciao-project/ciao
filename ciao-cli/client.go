//
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
//

package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

// Client represents a client for accessing ciao controller
type Client struct {
	controllerURL  string
	tenantID       string
	caCertFile     string
	clientCertFile string

	caCertPool *x509.CertPool
	clientCert *tls.Certificate

	tenants []string
}

type queryValue struct {
	name, value string
}

func (client *Client) prepareCAcert() error {
	if client.caCertFile != "" {
		caCert, err := ioutil.ReadFile(client.caCertFile)
		if err != nil {
			return errors.Wrap(err, "Unable to load requested CA certificate")
		}

		client.caCertPool, err = x509.SystemCertPool()
		if err != nil {
			return errors.Wrap(err, "Unable to create system certificate pool")
		}

		client.caCertPool.AppendCertsFromPEM(caCert)
	}
	return nil
}

func getTenantsFromCertFile(clientCertFile string) ([]string, error) {
	var certBlock, p *pem.Block

	data, err := ioutil.ReadFile(clientCertFile)
	if err != nil {
		return nil, errors.Wrap(err, "Error loading client cert file")
	}

	for {
		p, data = pem.Decode(data)
		if p == nil {
			break
		}
		if p.Type == "CERTIFICATE" {
			if certBlock != nil {
				return nil, errors.Wrap(err, "Incorrect number of certificate blocks in file")
			}
			certBlock = p
		}
	}

	if certBlock == nil {
		return nil, errors.New("No certificate block block in cert file")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, errors.New("Unable to parse x509 certificate data")
	}

	return cert.Subject.Organization, nil
}

func (client *Client) prepareClientCert() error {
	cert, err := tls.LoadX509KeyPair(client.clientCertFile, client.clientCertFile)
	if err != nil {
		return errors.Wrap(err, "Unable to load client certiticate")
	}
	client.clientCert = &cert

	client.tenants, err = getTenantsFromCertFile(client.clientCertFile)
	if err != nil {
		return errors.New("No tenant specified and unable to parse from certificate file")
	}

	if client.tenantID == "" {
		if len(client.tenants) == 0 {
			return errors.New("No tenants specified in certificate")
		}

		if len(client.tenants) > 1 {
			return errors.New("Multiple tenants available. Please specify one with -tenant-id")
		}

		client.tenantID = client.tenants[0]
	}

	return nil
}

// Init initialises a client for making requests
func (client *Client) Init() error {
	if client.controllerURL == "" {
		return errors.New("Controller URL must be specified")
	}

	if client.clientCertFile == "" {
		return errors.New("Client certificate file must be specified")
	}

	if !strings.HasPrefix(client.controllerURL, "https://") {
		client.controllerURL = fmt.Sprintf("https://%s:%d", client.controllerURL, api.Port)
	}

	if err := client.prepareCAcert(); err != nil {
		return err
	}

	if err := client.prepareClientCert(); err != nil {
		return err
	}

	return nil
}

func dumpJSON(body interface{}) {
	switch b := body.(type) {
	case []byte:
		var dump bytes.Buffer

		json.Indent(&dump, b, "", "\t")
		dump.WriteTo(os.Stdout)
	case map[string]interface{}:
		new, err := json.MarshalIndent(b, "", "\t")
		if err == nil {
			os.Stdout.Write(new)
		}
	}

	fmt.Printf("\n")
}

func (client *Client) buildComputeURL(format string, args ...interface{}) string {
	prefix := fmt.Sprintf("%s/v2.1/", client.controllerURL)
	return fmt.Sprintf(prefix+format, args...)
}

func (client *Client) buildCiaoURL(format string, args ...interface{}) string {
	prefix := fmt.Sprintf("%s/", client.controllerURL)
	return fmt.Sprintf(prefix+format, args...)
}

func (client *Client) sendHTTPRequestToken(method string, url string, values []queryValue, token string, body io.Reader, content string) (*http.Response, error) {
	req, err := http.NewRequest(method, os.ExpandEnv(url), body)
	if err != nil {
		return nil, err
	}

	infof("Sending %s %s\n", method, url)

	if values != nil {
		v := req.URL.Query()

		for _, value := range values {
			infof("Adding URL query %s=%s\n", value.name, value.value)
			v.Add(value.name, value.value)
		}

		req.URL.RawQuery = v.Encode()
	}

	if content != "" {
		contentType := fmt.Sprintf("application/%s", content)
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("Accept", contentType)
	} else if body != nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
	}

	tlsConfig := &tls.Config{}

	if client.caCertPool != nil {
		tlsConfig.RootCAs = client.caCertPool
	}

	if client.clientCert != nil {
		tlsConfig.Certificates = []tls.Certificate{*client.clientCert}
		tlsConfig.BuildNameToCertificate()
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	c := &http.Client{Transport: transport}
	resp, err := c.Do(req)
	if err != nil {
		errorf("Could not send HTTP request %s\n", err)
		return nil, err
	}

	infof("Got HTTP response (status %s)\n", resp.Status)

	if resp.StatusCode >= http.StatusBadRequest {
		respBody, errBody := ioutil.ReadAll(resp.Body)
		if errBody != nil {
			errorf("Could not read the HTTP response %s\n", errBody)
			dumpJSON(respBody)
			return resp, errBody
		}

		return resp, fmt.Errorf("HTTP Error [%d] for [%s %s]: %s", resp.StatusCode, method, url, respBody)
	}

	return resp, err
}

func (client *Client) sendHTTPRequest(method string, url string, values []queryValue, body io.Reader) (*http.Response, error) {
	return client.sendHTTPRequestToken(method, url, values, scopedToken, body, "")
}

func (client *Client) unmarshalHTTPResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errorf("Could not read the HTTP response %s\n", err)
		return err
	}

	err = json.Unmarshal(body, v)
	if err != nil {
		errorf("Could not unmarshal the HTTP response %s\n", err)
		return err
	}

	if glog.V(2) {
		dumpJSON(body)
	}

	return nil
}

func (client *Client) sendCiaoRequest(method string, url string, values []queryValue, body io.Reader, content string) (*http.Response, error) {
	return client.sendHTTPRequestToken(method, url, values, scopedToken, body, content)
}

func (client *Client) getRef(rel string, links []types.Link) string {
	for _, link := range links {
		if link.Rel == rel {
			return link.Href
		}
	}
	return ""
}

func (client *Client) getCiaoResource(name string, minVersion string) (string, error) {
	var resources []types.APILink
	var url string

	if client.checkPrivilege() {
		url = client.buildCiaoURL("")
	} else {
		url = client.buildCiaoURL(fmt.Sprintf("%s", client.tenantID))
	}

	resp, err := client.sendCiaoRequest("GET", url, nil, nil, "")
	if err != nil {
		return "", err
	}

	err = client.unmarshalHTTPResponse(resp, &resources)
	if err != nil {
		return "", err
	}

	for _, l := range resources {
		if l.Rel == name && l.MinVersion == minVersion {
			return l.Href, nil
		}
	}

	return "", errors.New("Supported version of resource not found")
}

func (client *Client) checkPrivilege() bool {
	for i := range client.tenants {
		if client.tenants[i] == "admin" {
			return true
		}
	}

	return false
}

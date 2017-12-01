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

package client

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
	"github.com/intel/tfortools"
	"github.com/pkg/errors"
)

// Client represents a client for accessing ciao controller
type Client struct {
	Template       string
	ControllerURL  string
	TenantID       string
	CACertFile     string
	ClientCertFile string

	caCertPool *x509.CertPool
	clientCert *tls.Certificate

	Tenants []string
}

type queryValue struct {
	name, value string
}

func Infof(format string, args ...interface{}) {
	if glog.V(1) {
		glog.InfoDepth(1, fmt.Sprintf("ciao-cli INFO: "+format, args...))
	}
}

func Errorf(format string, args ...interface{}) {
	glog.ErrorDepth(1, fmt.Sprintf("ciao-cli ERROR: "+format, args...))
}

func Fatalf(format string, args ...interface{}) {
	glog.FatalDepth(1, fmt.Sprintf("ciao-cli FATAL: "+format, args...))
}

func (client *Client) PrettyPrint(buff *bytes.Buffer, tname string, obj interface{}) {
	if client.Template != "" {
		tfortools.OutputToTemplate(buff, tname, client.Template, obj, nil)
	} else {
		tfortools.OutputToTemplate(buff, tname, "{{table .}}", obj, nil)
	}
}

func (client *Client) prepareCAcert() error {
	if client.CACertFile != "" {
		caCert, err := ioutil.ReadFile(client.CACertFile)
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
	cert, err := tls.LoadX509KeyPair(client.ClientCertFile, client.ClientCertFile)
	if err != nil {
		return errors.Wrap(err, "Unable to load client certiticate")
	}
	client.clientCert = &cert

	client.Tenants, err = getTenantsFromCertFile(client.ClientCertFile)
	if err != nil {
		return errors.New("No tenant specified and unable to parse from certificate file")
	}

	if client.TenantID == "" {
		if len(client.Tenants) == 0 {
			return errors.New("No tenants specified in certificate")
		}

		if len(client.Tenants) > 1 {
			return errors.New("Multiple tenants available. Please specify one with -tenant-id")
		}

		client.TenantID = client.Tenants[0]
	}

	return nil
}

// Init initialises a client for making requests
func (client *Client) Init() error {
	if client.ControllerURL == "" {
		return errors.New("Controller URL must be specified")
	}

	if client.ClientCertFile == "" {
		return errors.New("Client certificate file must be specified")
	}

	if !strings.HasPrefix(client.ControllerURL, "https://") {
		client.ControllerURL = fmt.Sprintf("https://%s:%d", client.ControllerURL, api.Port)
	}

	if err := client.prepareCAcert(); err != nil {
		return err
	}

	if err := client.prepareClientCert(); err != nil {
		return err
	}

	return nil
}

func (client *Client) buildComputeURL(format string, args ...interface{}) string {
	prefix := fmt.Sprintf("%s/v2.1/", client.ControllerURL)
	return fmt.Sprintf(prefix+format, args...)
}

func (client *Client) buildCiaoURL(format string, args ...interface{}) string {
	prefix := fmt.Sprintf("%s/", client.ControllerURL)
	return fmt.Sprintf(prefix+format, args...)
}

func (client *Client) sendHTTPRequest(method string, url string, values []queryValue, body io.Reader, content string) (*http.Response, error) {
	req, err := http.NewRequest(method, os.ExpandEnv(url), body)
	if err != nil {
		return nil, err
	}

	if values != nil {
		v := req.URL.Query()

		for _, value := range values {
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
		return nil, errors.Wrap(err, "Could not send HTTP request")
	}

	if resp.StatusCode >= http.StatusBadRequest {
		respBody, errBody := ioutil.ReadAll(resp.Body)
		if errBody != nil {
			return resp, fmt.Errorf("HTTP Error: %s", resp.Status)
		}

		return resp, fmt.Errorf("HTTP Error [%d] for [%s %s]: %s", resp.StatusCode, method, url, respBody)
	}

	return resp, err
}

func (client *Client) unmarshalHTTPResponse(resp *http.Response, v interface{}) error {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "Could not read HTTP response body")
	}

	err = json.Unmarshal(body, v)
	if err != nil {
		return errors.Wrap(err, "Could not unmarshal HTTP response body")
	}

	return nil
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

	if client.IsPrivileged() {
		url = client.buildCiaoURL("")
	} else {
		url = client.buildCiaoURL(fmt.Sprintf("%s", client.TenantID))
	}

	err := client.getResource(url, "", nil, &resources)
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

// IsPrivileged returns true if the user has admin privileges
func (client *Client) IsPrivileged() bool {
	for i := range client.Tenants {
		if client.Tenants[i] == "admin" {
			return true
		}
	}

	return false
}

func (client *Client) getResource(url string, content string, query []queryValue, result interface{}) error {
	resp, err := client.sendHTTPRequest("GET", url, query, nil, content)
	if err != nil {
		return errors.Wrapf(err, "Error making HTTP request to %s", url)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP response code from %s not as expected: %d", url, resp.StatusCode)
	}

	err = client.unmarshalHTTPResponse(resp, result)
	if err != nil {
		data, _ := ioutil.ReadAll(resp.Body)
		return errors.Wrapf(err, "Error parsing HTTP response: %s", data)
	}

	return nil
}

func (client *Client) deleteResource(url string, content string) error {
	resp, err := client.sendHTTPRequest("DELETE", url, nil, nil, content)
	if err != nil {
		return errors.Wrapf(err, "Error making HTTP request to %s", url)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("HTTP response code from %s not as expected: %s", url, resp.Status)
	}

	return nil
}

func (client *Client) putResource(url string, content string, request interface{}) error {
	b, err := json.Marshal(request)
	if err != nil {
		return errors.Wrap(err, "Error marshalling JSON")
	}

	resp, err := client.sendHTTPRequest("PUT", url, nil, bytes.NewReader(b), content)
	if err != nil {
		return errors.Wrapf(err, "Error making HTTP request to %s", url)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("HTTP response code from %s not as expected: %s", url, resp.Status)
	}

	return nil
}

func (client *Client) postResource(url string, content string, request interface{}, result interface{}) error {
	b, err := json.Marshal(request)
	if err != nil {
		return errors.Wrap(err, "Error marshalling JSON")
	}

	resp, err := client.sendHTTPRequest("POST", url, nil, bytes.NewReader(b), content)
	if err != nil {
		return errors.Wrapf(err, "Error making HTTP request to %s", url)
	}
	defer resp.Body.Close()

	if result != nil && resp.StatusCode != http.StatusNoContent {
		err = client.unmarshalHTTPResponse(resp, result)
		if err != nil {
			data, _ := ioutil.ReadAll(resp.Body)
			return errors.Wrapf(err, "Error parsing HTTP response: %s", data)
		}
	}

	return nil
}

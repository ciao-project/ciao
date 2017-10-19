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
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/ciao-controller/types"
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

func (client *Client) buildComputeURL(format string, args ...interface{}) string {
	prefix := fmt.Sprintf("%s/v2.1/", client.controllerURL)
	return fmt.Sprintf(prefix+format, args...)
}

func (client *Client) buildCiaoURL(format string, args ...interface{}) string {
	prefix := fmt.Sprintf("%s/", client.controllerURL)
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
	defer resp.Body.Close()

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

	if client.checkPrivilege() {
		url = client.buildCiaoURL("")
	} else {
		url = client.buildCiaoURL(fmt.Sprintf("%s", client.tenantID))
	}

	resp, err := client.sendHTTPRequest("GET", url, nil, nil, "")
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

// ListEvents retrieves the events for either all or the desired tenant
func (client *Client) ListEvents(tenantID string) (types.CiaoEvents, error) {
	var events types.CiaoEvents
	var url string

	if tenantID == "" {
		url = client.buildComputeURL("events")
	} else {
		url = client.buildComputeURL("%s/events", tenantID)
	}

	resp, err := client.sendHTTPRequest("GET", url, nil, nil, "")
	if err != nil {
		return events, errors.Wrap(err, "Error making HTTP request")
	}

	err = client.unmarshalHTTPResponse(resp, &events)
	if err != nil {
		return events, errors.Wrap(err, "Error parsing HTTP response")
	}

	return events, nil
}

// DeleteEvents deletes all events from
func (client *Client) DeleteEvents() error {
	url := client.buildComputeURL("events")

	resp, err := client.sendHTTPRequest("DELETE", url, nil, nil, "")
	if err != nil {
		return errors.Wrap(err, "Error making HTTP request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("Events log deletion failed: %s", resp.Status)
	}

	return nil
}

func (client *Client) getCiaoExternalIPsResource() (string, string, error) {
	url, err := client.getCiaoResource("external-ips", api.ExternalIPsV1)
	return url, api.ExternalIPsV1, err
}

// TBD: in an ideal world, we'd modify the GET to take a query.
func (client *Client) getExternalIPRef(address string) (string, error) {
	var IPs []types.MappedIP

	url, ver, err := client.getCiaoExternalIPsResource()
	if err != nil {
		return "", err
	}

	resp, err := client.sendHTTPRequest("GET", url, nil, nil, ver)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("External IP list failed: %s", resp.Status)
	}

	err = client.unmarshalHTTPResponse(resp, &IPs)
	if err != nil {
		return "", err
	}

	for _, IP := range IPs {
		if IP.ExternalIP == address {
			url := client.getRef("self", IP.Links)
			if url != "" {
				return url, nil
			}
		}
	}

	return "", types.ErrAddressNotFound
}

// MapExternalIP maps an IP from the pool to the given instance
func (client *Client) MapExternalIP(pool string, instanceID string) error {
	req := types.MapIPRequest{
		InstanceID: instanceID,
	}

	if pool != "" {
		req.PoolName = &pool
	}

	b, err := json.Marshal(req)
	if err != nil {
		return errors.Wrap(err, "Error marshalling JSON")
	}

	body := bytes.NewReader(b)

	url, ver, err := client.getCiaoExternalIPsResource()
	if err != nil {
		return errors.Wrap(err, "Error getting external IP resource")
	}

	resp, err := client.sendHTTPRequest("POST", url, nil, body, ver)
	if err != nil {
		return errors.Wrap(err, "Error making HTTP request")
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("External IP map failed: %s", resp.Status)
	}

	return nil
}

// ListExternalIPs returns the mapped IPs
func (client *Client) ListExternalIPs() ([]types.MappedIP, error) {
	var IPs []types.MappedIP

	url, ver, err := client.getCiaoExternalIPsResource()
	if err != nil {
		return IPs, errors.Wrap(err, "Error getting external IP resource")
	}

	resp, err := client.sendHTTPRequest("GET", url, nil, nil, ver)
	if err != nil {
		return IPs, errors.Wrap(err, "Error making HTTP request")
	}

	if resp.StatusCode != http.StatusOK {
		return IPs, fmt.Errorf("External IP list failed: %s", resp.Status)
	}

	err = client.unmarshalHTTPResponse(resp, &IPs)
	if err != nil {
		return IPs, errors.Wrap(err, "Error parsing HTTP response")
	}

	return IPs, err
}

// UnmapExternalIP unmaps the given address from the instance
func (client *Client) UnmapExternalIP(address string) error {
	url, err := client.getExternalIPRef(address)
	if err != nil {
		return errors.Wrap(err, "Error getting external IP reference")
	}

	ver := api.ExternalIPsV1

	resp, err := client.sendHTTPRequest("DELETE", url, nil, nil, ver)
	if err != nil {
		return errors.Wrap(err, "Error making HTTP request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("Unmap of address failed: %s", resp.Status)
	}

	return nil
}

func (client *Client) getCiaoPoolsResource() (string, error) {
	return client.getCiaoResource("pools", api.PoolsV1)
}

func (client *Client) getCiaoPoolRef(name string) (string, error) {
	var pools types.ListPoolsResponse

	query := queryValue{
		name:  "name",
		value: name,
	}

	url, err := client.getCiaoPoolsResource()
	if err != nil {
		return "", err
	}

	ver := api.PoolsV1

	resp, err := client.sendHTTPRequest("GET", url, []queryValue{query}, nil, ver)
	if err != nil {
		return "", err
	}

	err = client.unmarshalHTTPResponse(resp, &pools)
	if err != nil {
		return "", err
	}

	// we have now the pool ID
	if len(pools.Pools) != 1 {
		return "", errors.New("No pool by that name found")
	}

	links := pools.Pools[0].Links
	url = client.getRef("self", links)
	if url == "" {
		return url, errors.New("Invalid Link returned from controller")
	}

	return url, nil
}

// GetExternalIPPool gets the details of a single external IP pool
func (client *Client) GetExternalIPPool(name string) (types.Pool, error) {
	var pool types.Pool

	if !client.checkPrivilege() {
		return pool, errors.New("This command is only available to admins")
	}

	url, err := client.getCiaoPoolRef(name)
	if err != nil {
		return pool, err
	}

	ver := api.PoolsV1

	resp, err := client.sendHTTPRequest("GET", url, nil, nil, ver)
	if err != nil {
		return pool, err
	}

	err = client.unmarshalHTTPResponse(resp, &pool)
	if err != nil {
		return pool, err
	}

	return pool, nil
}

// CreateExternalIPPool creates a pool of IPs
func (client *Client) CreateExternalIPPool(name string) error {
	if !client.checkPrivilege() {
		return errors.New("This command is only available to admins")
	}

	req := types.NewPoolRequest{
		Name: name,
	}

	b, err := json.Marshal(req)
	if err != nil {
		return errors.Wrap(err, "Error marshalling JSON")
	}

	body := bytes.NewReader(b)

	url, err := client.getCiaoPoolsResource()
	if err != nil {
		return errors.Wrap(err, "Error getting pool resource")
	}

	ver := api.PoolsV1

	resp, err := client.sendHTTPRequest("POST", url, nil, body, ver)
	if err != nil {
		return errors.Wrap(err, "Error making HTTP request")
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Pool creation failed: %s", resp.Status)
	}

	return nil
}

// ListExternalIPPools lists the pools in which IPs are available
func (client *Client) ListExternalIPPools() (types.ListPoolsResponse, error) {
	var pools types.ListPoolsResponse

	url, err := client.getCiaoPoolsResource()
	if err != nil {
		return pools, errors.Wrap(err, "Error getting pool resource")
	}

	ver := api.PoolsV1

	resp, err := client.sendHTTPRequest("GET", url, nil, nil, ver)
	if err != nil {
		return pools, errors.Wrap(err, "Error making HTTP request")
	}

	err = client.unmarshalHTTPResponse(resp, &pools)
	if err != nil {
		return pools, errors.Wrap(err, "Error parsing HTTP response")
	}

	return pools, nil
}

// DeleteExternalIPPool deletes the pool of the given name
func (client *Client) DeleteExternalIPPool(pool string) error {
	if !client.checkPrivilege() {
		return errors.New("This command is only available to admins")
	}

	url, err := client.getCiaoPoolRef(pool)
	if err != nil {
		return errors.Wrap(err, "Error getting pool reference")
	}

	ver := api.PoolsV1

	resp, err := client.sendHTTPRequest("DELETE", url, nil, nil, ver)
	if err != nil {
		return errors.Wrap(err, "Error making HTTP request")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Pool deletion failed: %s", resp.Status)
	}

	return nil
}

// AddExternalIPSubnet adds a subnet to the external IP pool
func (client *Client) AddExternalIPSubnet(pool string, subnet *net.IPNet) error {
	if !client.checkPrivilege() {
		return errors.New("This command is only available to admins")
	}

	var req types.NewAddressRequest

	url, err := client.getCiaoPoolRef(pool)
	if err != nil {
		return errors.Wrap(err, "Error getting pool reference")
	}

	s := subnet.String()
	req.Subnet = &s

	b, err := json.Marshal(req)
	if err != nil {
		return errors.Wrap(err, "Error marshalling JSON")
	}

	body := bytes.NewReader(b)

	ver := api.PoolsV1

	resp, err := client.sendHTTPRequest("POST", url, nil, body, ver)
	if err != nil {
		return errors.Wrap(err, "Error making HTTP request")
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Adding subnet failed: %s", resp.Status)
	}

	return nil
}

// AddExternalIPAddresses adds a set of IP addresses to the external IP pool
func (client *Client) AddExternalIPAddresses(pool string, IPs []string) error {
	if !client.checkPrivilege() {
		return errors.New("This command is only available to admins")
	}

	var req types.NewAddressRequest

	url, err := client.getCiaoPoolRef(pool)
	if err != nil {
		return errors.Wrap(err, "Error getting pool reference")
	}

	for _, IP := range IPs {
		addr := types.NewIPAddressRequest{
			IP: IP,
		}

		req.IPs = append(req.IPs, addr)
	}

	b, err := json.Marshal(req)
	if err != nil {
		return errors.Wrap(err, "Error marshalling JSON")
	}

	body := bytes.NewReader(b)

	ver := api.PoolsV1

	resp, err := client.sendHTTPRequest("POST", url, nil, body, ver)
	if err != nil {
		return errors.Wrap(err, "Error making HTTP request")
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Adding address failed: %s", resp.Status)
	}

	return nil
}

func (client *Client) getSubnetRef(pool types.Pool, cidr string) string {
	for _, sub := range pool.Subnets {
		if sub.CIDR == cidr {
			return client.getRef("self", sub.Links)
		}
	}

	return ""
}

func (client *Client) getIPRef(pool types.Pool, address string) string {
	for _, ip := range pool.IPs {
		if ip.Address == address {
			return client.getRef("self", ip.Links)
		}
	}

	return ""
}

// RemoveExternalIPSubnet removes a subnet from the pool
func (client *Client) RemoveExternalIPSubnet(pool string, subnet *net.IPNet) error {
	if !client.checkPrivilege() {
		return errors.New("This command is only available to admins")
	}

	p, err := client.GetExternalIPPool(pool)
	if err != nil {
		return errors.Wrap(err, "Error getting external IP pool")
	}

	url := client.getSubnetRef(p, subnet.String())
	if url == "" {
		return fmt.Errorf("Subnet not present in pool")
	}

	ver := api.PoolsV1

	resp, err := client.sendHTTPRequest("DELETE", url, nil, nil, ver)
	if err != nil {
		return errors.Wrap(err, "Error making HTTP request")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Subnet removal failed: %s", resp.Status)
	}

	return nil
}

// RemoveExternalIPAddress removes a single IP address from the pool
func (client *Client) RemoveExternalIPAddress(pool string, IP string) error {
	if !client.checkPrivilege() {
		return errors.New("This command is only available to admins")
	}

	p, err := client.GetExternalIPPool(pool)
	if err != nil {
		return errors.Wrap(err, "Error getting external IP pool")
	}

	url := client.getIPRef(p, IP)
	if url == "" {
		return fmt.Errorf("IP not present in pool")
	}

	ver := api.PoolsV1

	resp, err := client.sendHTTPRequest("DELETE", url, nil, nil, ver)
	if err != nil {
		return errors.Wrap(err, "Error making HTTP request")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("IP removal failed: %s", resp.Status)
	}

	return nil
}

// GetImage retrieves the details for an image
func (client *Client) GetImage(imageID string) (types.Image, error) {
	var i types.Image

	var url string
	if client.checkPrivilege() && client.tenantID == "admin" {
		url = client.buildCiaoURL("images/%s", imageID)
	} else {
		url = client.buildCiaoURL("%s/images/%s", client.tenantID, imageID)
	}

	resp, err := client.sendHTTPRequest("GET", url, nil, nil, api.ImagesV1)
	if err != nil {
		return i, errors.Wrap(err, "Error making HTTP request")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return i, fmt.Errorf("Image get failed: %s", resp.Status)
	}

	err = client.unmarshalHTTPResponse(resp, &i)
	if err != nil {
		return i, errors.Wrap(err, "Error parsing HTTP response")
	}

	return i, nil
}

func (client *Client) uploadTenantImage(tenant, image string, data io.Reader) error {
	var url string
	if client.checkPrivilege() && client.tenantID == "admin" {
		url = client.buildCiaoURL("images/%s/file", image)
	} else {
		url = client.buildCiaoURL("%s/images/%s/file", client.tenantID, image)
	}

	resp, err := client.sendHTTPRequest("PUT", url, nil, data, fmt.Sprintf("%s/octet-stream", api.ImagesV1))
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Unexpected HTTP response code (%d): %s", resp.StatusCode, resp.Status)
	}

	return err
}

// CreateImage creates and uploads a new image
func (client *Client) CreateImage(name string, visibility types.Visibility, ID string, data io.Reader) (string, error) {
	opts := api.CreateImageRequest{
		Name:       name,
		ID:         ID,
		Visibility: visibility,
	}

	b, err := json.Marshal(opts)
	if err != nil {
		return "", errors.Wrap(err, "Error marshalling JSON")
	}

	body := bytes.NewReader(b)

	var url string
	if client.checkPrivilege() && client.tenantID == "admin" {
		url = client.buildCiaoURL("images")
	} else {
		url = client.buildCiaoURL("%s/images", client.tenantID)
	}

	resp, err := client.sendHTTPRequest("POST", url, nil, body, api.ImagesV1)
	if err != nil {
		return "", errors.Wrap(err, "Error making HTTP request")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("Image creation failed: %s", resp.Status)
	}

	var image types.Image
	err = client.unmarshalHTTPResponse(resp, &image)
	if err != nil {
		return "", errors.Wrap(err, "Error parsing HTTP response")
	}

	err = client.uploadTenantImage(client.tenantID, image.ID, data)
	if err != nil {
		return "", errors.Wrap(err, "Error uploading image data")
	}

	return image.ID, nil
}

// ListImages retrieves the set of available images
func (client *Client) ListImages() ([]types.Image, error) {
	var images []types.Image

	var url string
	if client.checkPrivilege() && client.tenantID == "admin" {
		url = client.buildCiaoURL("images")
	} else {
		url = client.buildCiaoURL("%s/images", client.tenantID)
	}

	resp, err := client.sendHTTPRequest("GET", url, nil, nil, api.ImagesV1)
	if err != nil {
		return images, errors.Wrap(err, "Error making HTTP request")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return images, fmt.Errorf("Image list failed: %s", resp.Status)
	}

	err = client.unmarshalHTTPResponse(resp, &images)
	if err != nil {
		return images, errors.Wrap(err, "Error parsing HTTP response")
	}

	return images, nil
}

// DeleteImage deletes the given image
func (client *Client) DeleteImage(imageID string) error {
	var url string
	if client.checkPrivilege() && client.tenantID == "admin" {
		url = client.buildCiaoURL("images/%s", imageID)
	} else {
		url = client.buildCiaoURL("%s/images/%s", client.tenantID, imageID)
	}

	resp, err := client.sendHTTPRequest("DELETE", url, nil, nil, api.ImagesV1)
	if err != nil {
		return errors.Wrap(err, "Error making HTTP request")
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Image delete failed: %s", resp.Status)
	}

	return nil
}

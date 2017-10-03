//
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
//

package sdk

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

	"github.com/ciao-project/ciao/payloads"
	"github.com/ciao-project/ciao/ciao-controller/api"
	"github.com/ciao-project/ciao/ciao-controller/types"
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"gopkg.in/yaml.v2"
)

var scopedToken string

var Template string

var InstanceFlags = new(InstanceCmd)

func infof(format string, args ...interface{}) {
	if glog.V(1) {
		glog.InfoDepth(1, fmt.Sprintf("ciao INFO: "+format, args...))
	}
}

func errorf(format string, args ...interface{}) {
	glog.ErrorDepth(1, fmt.Sprintf("ciao ERROR: "+format, args...))
}

func fatalf(format string, args ...interface{}) {
	glog.FatalDepth(1, fmt.Sprintf("ciao FATAL: "+format, args...))
}

var (
	tenantID       = new(string)
	controllerURL  = new(string)
	ciaoPort       = new(int)
	caCertFile     = new(string)
	clientCertFile = new(string)
)

const (
	ciaoControllerEnv     = "CIAO_CONTROLLER"
	ciaoCACertFileEnv     = "CIAO_CA_CERT_FILE"
	ciaoClientCertFileEnv = "CIAO_CLIENT_CERT_FILE"
)

var caCertPool *x509.CertPool
var clientCert *tls.Certificate
var tenants []string

type queryValue struct {
	name, value string
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

func buildComputeURL(format string, args ...interface{}) string {
	prefix := fmt.Sprintf("https://%s:%d/v2.1/", *controllerURL, *ciaoPort)
	return fmt.Sprintf(prefix+format, args...)
}

func buildCiaoURL(format string, args ...interface{}) string {
	prefix := fmt.Sprintf("https://%s:%d/", *controllerURL, *ciaoPort)
	return fmt.Sprintf(prefix+format, args...)
}

func buildBlockURL(format string, args ...interface{}) string {
	prefix := fmt.Sprintf("https://%s:%d/v2/", *controllerURL, *ciaoPort)
	return fmt.Sprintf(prefix+format, args...)
}

func buildImageURL(format string, args ...interface{}) string {
	prefix := fmt.Sprintf("https://%s:%d/v2/", *controllerURL, *ciaoPort)
	prefix = fmt.Sprintf("%s%s/", prefix, *tenantID)
	return fmt.Sprintf(prefix+format, args...)
}

func sendHTTPRequestToken(method string, url string, values []queryValue, token string, body io.Reader, content string) (*http.Response, error) {
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

	if caCertPool != nil {
		tlsConfig.RootCAs = caCertPool
	}

	if clientCert != nil {
		tlsConfig.Certificates = []tls.Certificate{*clientCert}
		tlsConfig.BuildNameToCertificate()
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	client := &http.Client{Transport: transport}
	resp, err := client.Do(req)
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

func sendHTTPRequest(method string, url string, values []queryValue, body io.Reader) (*http.Response, error) {
	return sendHTTPRequestToken(method, url, values, scopedToken, body, "")
}

func unmarshalHTTPResponse(resp *http.Response, v interface{}) error {
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

func sendCiaoRequest(method string, url string, values []queryValue, body io.Reader, content string) (*http.Response, error) {
	return sendHTTPRequestToken(method, url, values, scopedToken, body, content)
}

func getRef(rel string, links []types.Link) string {
	for _, link := range links {
		if link.Rel == rel {
			return link.Href
		}
	}
	return ""
}

func getCiaoResource(name string, minVersion string) (string, error) {
	var resources []types.APILink
	var url string

	if checkPrivilege() {
		url = buildCiaoURL("")
	} else {
		url = buildCiaoURL(fmt.Sprintf("%s", *tenantID))
	}

	resp, err := sendCiaoRequest("GET", url, nil, nil, "")
	if err != nil {
		return "", err
	}

	err = unmarshalHTTPResponse(resp, &resources)
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

func getCiaoWorkloadsResource() (string, error) {
	return getCiaoResource("workloads", api.WorkloadsV1)
}

func checkPrivilege() bool {
	for i := range tenants {
		if tenants[i] == "admin" {
			return true
		}
	}

	return false
}

func limitToString(limit int) string {
	if limit == -1 {
		return "Unlimited"
	}

	return fmt.Sprintf("%d", limit)
}

func GetCiaoEnvVariables() {
	controller := os.Getenv(ciaoControllerEnv)
	ca := os.Getenv(ciaoCACertFileEnv)
	clientCert := os.Getenv(ciaoClientCertFileEnv)

	infof("Ciao environment variables:\n")
	infof("\t%s:%s\n", ciaoControllerEnv, controller)
	infof("\t%s:%s\n", ciaoCACertFileEnv, ca)
	infof("\t%s:%s\n", ciaoClientCertFileEnv, clientCert)

	if controller != "" && *controllerURL == "" {
		*controllerURL = controller
	}

	if ca != "" && *caCertFile == "" {
		*caCertFile = ca
	}

	if clientCert != "" && *clientCertFile == "" {
		*clientCertFile = clientCert
	}
}

func CheckCompulsoryOptions() {
	fatal := ""

	if *clientCertFile == "" {
		fatal += "Missing required client certificate file\n"
	}
	if *controllerURL == "" {
		fatal += "Missing required Ciao controller URL\n"
	}

	if fatal != "" {
		fatalf(fatal)
	}
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

func prepareWithClientCert() {
	cert, err := tls.LoadX509KeyPair(*clientCertFile, *clientCertFile)
	if err != nil {
		fatalf("Unable to load client certiticate: %s", err)
	}
	clientCert = &cert

	tenants, err = getTenantsFromCertFile(*clientCertFile)
	if err != nil {
		fatalf("No tenant specified and unable to parse from certificate file")
	}

	if *tenantID == "" {
		if len(tenants) == 0 {
			fatalf("No tenants specified in certificate")
		}

		if len(tenants) > 1 {
			fmt.Println("Tenants available:")
			for i := range tenants {
				fmt.Println(tenants[i])
			}
			fatalf("Multiple tenants available. Please specify one with -tenant-id")
		}

		*tenantID = tenants[0]
	}

}

func PrepareForCommand() {
	/* Load CA file if necessary */
	if *caCertFile != "" {
		caCert, err := ioutil.ReadFile(*caCertFile)
		if err != nil {
			fatalf("Unable to load requested CA certificate: %s\n", err)
		}
		caCertPool, err = x509.SystemCertPool()
		if err != nil {
			fatalf("Unable to create system certificate pool: %s\n", err)
		}
		caCertPool.AppendCertsFromPEM(caCert)
	}

	prepareWithClientCert()
	if *ciaoPort == 0 {
		*ciaoPort = api.Port
	}
}

func optToReqStorage(opt workloadOptions) ([]types.StorageResource, error) {
	storage := make([]types.StorageResource, 0)
	bootableCount := 0
	for _, disk := range opt.Disks {
		res := types.StorageResource{
			Size:      disk.Size,
			Bootable:  disk.Bootable,
			Ephemeral: disk.Ephemeral,
		}

		// Use existing volume
		if disk.ID != nil {
			res.ID = *disk.ID
		} else {
			// Create a new one
			if disk.Source.Type == "" {
				disk.Source.Type = types.Empty
			}

			if disk.Source.Type != types.Empty {
				res.SourceType = disk.Source.Type
				res.SourceID = disk.Source.ID

				if res.SourceID == "" {
					return nil, errors.New("Invalid workload yaml: when using a source an id must also be specified")
				}
			} else {
				if disk.Bootable == true {
					// you may not request a bootable drive
					// from an empty source
					return nil, errors.New("Invalid workload yaml: empty disk source may not be bootable")
				}

				if disk.Size <= 0 {
					return nil, errors.New("Invalid workload yaml: size required when creating a volume")
				}
			}
		}

		if disk.Bootable {
			bootableCount++
		}

		storage = append(storage, res)
	}

	if payloads.Hypervisor(opt.VMType) == payloads.QEMU && bootableCount == 0 {
		return nil, errors.New("Invalid workload yaml: no bootable disks specified for a VM")
	}

	return storage, nil
}

func optToReq(opt workloadOptions, req *types.Workload) error {
	b, err := ioutil.ReadFile(opt.CloudConfigFile)
	if err != nil {
		return err
	}

	config := string(b)

	// this is where you'd validate that the options make
	// sense.
	req.Description = opt.Description
	req.VMType = payloads.Hypervisor(opt.VMType)
	req.FWType = opt.FWType
	req.ImageName = opt.ImageName
	req.Config = config
	req.Storage, err = optToReqStorage(opt)

	if err != nil {
		return err
	}

	// all default resources are required.
	defaults := opt.Defaults

	r := payloads.RequestedResource{
		Type:  payloads.VCPUs,
		Value: defaults.VCPUs,
	}
	req.Defaults = append(req.Defaults, r)

	r = payloads.RequestedResource{
		Type:  payloads.MemMB,
		Value: defaults.MemMB,
	}
	req.Defaults = append(req.Defaults, r)

	return nil
}

func outputWorkload(w types.Workload) {
	var opt workloadOptions

	opt.Description = w.Description
	opt.VMType = string(w.VMType)
	opt.FWType = w.FWType
	opt.ImageName = w.ImageName
	for _, d := range w.Defaults {
		if d.Type == payloads.VCPUs {
			opt.Defaults.VCPUs = d.Value
		} else if d.Type == payloads.MemMB {
			opt.Defaults.MemMB = d.Value
		}
	}

	for _, s := range w.Storage {
		d := disk{
			Size:      s.Size,
			Bootable:  s.Bootable,
			Ephemeral: s.Ephemeral,
		}
		if s.ID != "" {
			d.ID = &s.ID
		}

		src := source{
			Type: s.SourceType,
			ID:   s.SourceID,
		}

		d.Source = src

		opt.Disks = append(opt.Disks, d)
	}

	b, err := yaml.Marshal(opt)
	if err != nil {
		fatalf(err.Error())
	}

	fmt.Println(string(b))
	fmt.Println(w.Config)
}

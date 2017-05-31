/*
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
*/
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"encoding/pem"

	"crypto/x509"

	"path/filepath"

	"io"

	"github.com/01org/ciao/deploy"
	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"github.com/01org/ciao/ssntp/certs"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type signingRequest struct {
	role     ssntp.Role
	csr      []byte
	approved bool
}

var anchorCertPath string
var caCertPath string

var serverURL string
var secretToken string
var autoApprove bool
var scheduler bool
var systemPkiLocation string
var serveRequests bool

// Used when generating the configuration file
var cephID string
var httpsCaCertPath string
var httpsCertPath string
var identityUser string
var identityPassword string
var identityURL string
var adminSSHKey string
var adminPassword string
var computeNet string
var mgmtNet string
var serverIP string

var signingRequestMutex sync.Mutex
var requests map[string]*signingRequest

var ciaoConfigurationDirectory = "/etc/ciao"

func init() {
	flag.StringVar(&serverURL, "server", ":9000", "Port and optional IP to listen on (host:port)")
	flag.StringVar(&secretToken, "token", "ciao", "Secret token shared with client")
	flag.BoolVar(&autoApprove, "auto-approve", true, "Automatically approve requests")
	flag.BoolVar(&scheduler, "scheduler", true, "Setup scheduler on this machine")
	flag.StringVar(&systemPkiLocation, "system-pki-location", "/etc/pki/ciao", "Filesystem location for certificates")
	flag.BoolVar(&serveRequests, "serve-requests", true, "Setup server to listen for join requests")
	flag.StringVar(&serverIP, "server-ip", "", "IP address that scheduler is reachable on")

	// For configuration file generation
	flag.StringVar(&cephID, "ceph-id", "ciao", "The ceph id for the storage cluster")
	flag.StringVar(&httpsCaCertPath, "https-ca-cert", "", "Path to CA certificate for HTTP service")
	flag.StringVar(&httpsCertPath, "https-cert", "", "Path to certificate for HTTPS service")
	flag.StringVar(&identityUser, "keystone-user", "", "Username for controller to access keystone (service user)")
	flag.StringVar(&identityPassword, "keystone-password", "", "Password for controller to access keystone (service user)")
	flag.StringVar(&identityURL, "keystone-url", "", "URL for keystone server")
	flag.StringVar(&adminSSHKey, "admin-ssh-key", "", "SSH public key for accessing CNCI")
	flag.StringVar(&adminPassword, "admin-password", "", "Password for accessing CNCI")
	flag.StringVar(&computeNet, "compute-net", "", "Network range for compute network")
	flag.StringVar(&mgmtNet, "mgmt-net", "", "Network range for management network")

	requests = make(map[string]*signingRequest)
}

func sign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	if vars["token"] != secretToken {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	var role ssntp.Role
	switch vars["role"] {
	case "agent":
		role = ssntp.AGENT
	case "netagent":
		role = ssntp.NETAGENT
	case "dual":
		role = ssntp.AGENT | ssntp.NETAGENT
	case "controller":
		role = ssntp.Controller
	default:
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	request := signingRequest{
		role:     role,
		approved: autoApprove,
	}

	var err error
	request.csr, err = ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading from socket: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	csrBlock, _ := pem.Decode(request.csr)
	if csrBlock.Type != "CERTIFICATE REQUEST" {
		fmt.Fprintf(os.Stderr, "Error decoding provided certificate\n")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	certReq, err := x509.ParseCertificateRequest(csrBlock.Bytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing certificate request: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fp := certs.FingerPrint(certReq)
	fmt.Printf("Received signing request for: %s\n", fp)
	signingRequestMutex.Lock()
	requests[fp] = &request
	signingRequestMutex.Unlock()
	w.WriteHeader(http.StatusCreated)
}

func cacert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	anchorCertByes, err := ioutil.ReadFile(anchorCertPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read anchor certificate: %s: %v\n", anchorCertPath, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	certBlock, _ := pem.Decode(anchorCertByes)
	if certBlock.Type != "CERTIFICATE" {
		fmt.Fprintf(os.Stderr, "Need to have certificate as first block\n")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write(pem.EncodeToMemory(certBlock))
}

func cert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	if vars["token"] != secretToken {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	fp := vars["fingerprint"]
	signingRequestMutex.Lock()
	request, ok := requests[fp]
	signingRequestMutex.Unlock()
	if !ok {
		fmt.Fprintf(os.Stderr, "Certificate request for unknown fingerprint: %s\n", fp)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if !request.approved {
		fmt.Fprintf(os.Stderr, "Certificate request for unapproved certifcate with fingerprint: %s\n", fp)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	anchorCertByes, err := ioutil.ReadFile(anchorCertPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read anchor certificate: %s: %v\n", anchorCertPath, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var certOutput bytes.Buffer
	err = certs.CreateCertFromCSR(request.role, request.csr, anchorCertByes, &certOutput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create cert from CSR: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Printf("Serving signed cert for: %s\n", fp)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write(certOutput.Bytes())
}

func checkTool(tool string) bool {
	switch tool {
	case "ciao-launcher":
		fallthrough
	case "ciao-controller":
		fallthrough
	case "ciao-deploy-client":
		return true
	default:
		return false
	}
}

func toolPath(tool string) string {
	data, err := exec.Command("go", "env", "GOPATH").Output()
	gp := ""
	if err == nil {
		gp = filepath.Clean(strings.TrimSpace(string(data)))
	}
	path := filepath.Join(gp, "bin", tool)

	return path
}

func download(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)

	tool := vars["tool"]
	if !checkTool(tool) {
		fmt.Fprintf(os.Stderr, "Request for unexpected tool: %s\n", tool)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var path string
	if path = toolPath(tool); path == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)

	_, err = io.Copy(w, f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error copying data via http: %v", err)
	}
}

func serve(wg *sync.WaitGroup) {
	r := mux.NewRouter()
	r.HandleFunc("/sign/{token}/{role}", sign)
	r.HandleFunc("/cacert", cacert)
	r.HandleFunc("/cert/{token}/{fingerprint}", cert)
	r.HandleFunc("/download/{tool}", download)
	http.Handle("/", r)

	server := &http.Server{
		Addr:    serverURL,
		Handler: r,
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT)
	wg.Add(1)
	go func() {
		_ = <-signalCh
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		server.Shutdown(ctx)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Error serving deployment service")
		}
		wg.Done()
	}()
}

func copyCertificatesToDestination(certFilePath, caCertFilePath string) error {
	systemCertPath := path.Join(systemPkiLocation, deploy.CertName(ssntp.SCHEDULER))
	systemCaCertPath := path.Join(systemPkiLocation, "CAcert.pem")
	if err := os.Chmod(certFilePath, 0644); err != nil {
		return errors.Wrap(err, "Error chmod()ing anchor certificate")
	}
	if err := os.Chmod(caCertFilePath, 0644); err != nil {
		return errors.Wrap(err, "Error chmod()ing CA certificate")
	}

	cmd := exec.Command("sudo", "mkdir", "-p", systemPkiLocation)
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "Error running: %v", cmd.Args)
	}

	err := deploy.SudoCopyFile(systemCertPath, certFilePath)
	if err != nil {
		return errors.Wrap(err, "Error copying certificate to system location")
	}

	err = deploy.SudoCopyFile(systemCaCertPath, caCertFilePath)
	if err != nil {
		if err := deploy.SudoDeleteFile(systemCertPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting system certificate on error cleanup: %v\n", err)
		}
		return errors.Wrap(err, "Error copying CA certificate to system location")
	}
	return nil
}

func createSchedulerCert() (string, string, error) {
	systemCertPath := path.Join(systemPkiLocation, deploy.CertName(ssntp.SCHEDULER))
	systemCaCertPath := path.Join(systemPkiLocation, "CAcert.pem")

	if _, err := os.Stat(systemCertPath); err == nil {
		if _, err := os.Stat(systemCaCertPath); err == nil {
			fmt.Printf("Scheduler (and CA) certificate already installed. Skipping creation.\n")
			return systemCertPath, systemCaCertPath, nil
		} else if !os.IsNotExist(err) {
			return "", "", errors.Wrap(err, "Error stat()ing CA cert file")
		}
	} else if !os.IsNotExist(err) {
		return "", "", errors.Wrap(err, "Error stat()ing cert file")
	}

	certFile, err := ioutil.TempFile("", deploy.CertName(ssntp.SCHEDULER))
	if err != nil {
		return "", "", errors.Wrap(err, "Unable to create temporary file for scheduler cert")
	}
	defer certFile.Close()
	defer os.Remove(certFile.Name())

	caCertFile, err := ioutil.TempFile("", "CAcert.pem")
	if err != nil {
		return "", "", errors.Wrap(err, "Unable to create temporary file for CA certificate")
	}
	defer caCertFile.Close()
	defer os.Remove(caCertFile.Name())

	hs, err := os.Hostname()
	if err != nil {
		hs = "localhost"
	}

	hosts := []string{hs}
	mgmtIPs := []string{serverIP}

	template, err := certs.CreateCertTemplate(ssntp.SCHEDULER, "Ciao Deployment", "", hosts, mgmtIPs)
	if err != nil {
		return "", "", errors.Wrap(err, "Error creating scheduler certificate template")
	}
	if err := certs.CreateAnchorCert(template, false, certFile, caCertFile); err != nil {
		return "", "", errors.Wrap(err, "Error creating anchor certificate")
	}

	if err := copyCertificatesToDestination(certFile.Name(), caCertFile.Name()); err != nil {
		return "", "", errors.Wrap(err, "Error copying certificates to destination")
	}

	fmt.Printf("Scheduler certificate created in: %s\n", systemCertPath)
	fmt.Printf("CA certificate installed in: %s\n", systemCaCertPath)
	return systemCertPath, systemCaCertPath, nil
}

func createConfiguration() (string, error) {
	config := &payloads.Configure{}
	config.InitDefaults()
	config.Configure.Scheduler.ConfigStorageURI = "/etc/ciao/configuration.yaml"

	config.Configure.Storage.CephID = cephID

	config.Configure.Controller.HTTPSCACert = httpsCaCertPath
	config.Configure.Controller.HTTPSKey = httpsCertPath

	config.Configure.Controller.IdentityUser = identityUser
	config.Configure.Controller.IdentityPassword = identityPassword
	config.Configure.IdentityService.URL = identityURL

	config.Configure.Controller.AdminPassword = adminPassword
	config.Configure.Controller.AdminSSHKey = adminSSHKey

	config.Configure.Launcher.ComputeNetwork = []string{computeNet}
	config.Configure.Launcher.ManagementNetwork = []string{mgmtNet}
	config.Configure.Launcher.DiskLimit = false
	config.Configure.Launcher.MemoryLimit = false

	data, err := yaml.Marshal(config)
	if err != nil {
		return "", errors.Wrap(err, "Error creating marshalling configuration data")
	}

	f, err := ioutil.TempFile("", "configuration.yaml")
	if err != nil {
		return "", errors.Wrap(err, "Error creating temporary file")
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return "", errors.Wrap(err, "Error writing data to temporary file")
	}

	return f.Name(), nil
}

func installScheduler(caCertPath, certPath string) error {
	fmt.Println("Installing scheduler")
	err := deploy.SudoMakeDirectory(ciaoConfigurationDirectory)
	if err != nil {
		return errors.Wrap(err, "Error making configuration directory")
	}

	fmt.Println("Creating configuration file")
	p, err := createConfiguration()
	if err != nil {
		return errors.Wrap(err, "Error creating configuration file")
	}
	defer os.Remove(p)

	configPath := path.Join(ciaoConfigurationDirectory, "configuration.yaml")
	err = deploy.SudoCopyFile(configPath, p)
	if err != nil {
		return errors.Wrap(err, "Error copying configuration to destination")
	}

	fmt.Printf("Configuration file created: %s\n", configPath)

	err = deploy.InstallTool(toolPath("ciao-scheduler"), "ciao-scheduler", caCertPath, certPath)
	if err != nil {
		return errors.Wrap(err, "Error installing scheduler binary")
	}

	return nil
}

func main() {
	flag.Parse()

	var err error
	anchorCertPath, caCertPath, err = createSchedulerCert()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating scheduler certificate: %s\n", err)
		os.Exit(1)
	}
	if scheduler {
		installScheduler(caCertPath, anchorCertPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error installing scheduler: %s\n", err)
			os.Exit(1)
		}
	}

	if serveRequests {
		fmt.Println("Listening for enrollment requests")
		var wg sync.WaitGroup
		serve(&wg)
		wg.Wait()
	}
}

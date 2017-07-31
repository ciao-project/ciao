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
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os/user"
	"syscall"

	"os"

	"path"

	"github.com/01org/ciao/deploy"
	"github.com/01org/ciao/ssntp"
	"github.com/01org/ciao/ssntp/certs"
	"github.com/pkg/errors"
)

var deployServerHost string
var schedulerHost string
var token string
var role string
var downloadClient bool
var force bool
var systemPKILocation string

var ssntpRole ssntp.Role

func init() {
	flag.StringVar(&deployServerHost, "deploy-server", "", "URI for deployment server e.g. http://deploy.example.com:9000")
	flag.StringVar(&schedulerHost, "server", "", "Server on which scheduler is running")
	flag.StringVar(&token, "token", "ciao", "Secret token shared with server")
	flag.StringVar(&role, "role", "", "One of: controller, agent, netagent or dual (agent and netagent)")
	flag.BoolVar(&downloadClient, "download-client", true, "Download client from server")
	flag.BoolVar(&force, "force", false, "Force steps that would normally skippped")
	flag.StringVar(&systemPKILocation, "system-pki-location", "/etc/pki/ciao", "Filesystem location for certificates")
}

func roleToSSNTPRole(role string) ssntp.Role {
	switch role {
	case "agent":
		return ssntp.AGENT
	case "netagent":
		return ssntp.NETAGENT
	case "dual":
		return ssntp.NETAGENT | ssntp.AGENT
	case "controller":
		return ssntp.Controller
	default:
		return ssntp.UNKNOWN
	}
}

func validateFlags() error {
	if deployServerHost == "" {
		return errors.New("--deploy-server must be specified")
	}

	switch role {
	case "agent":
	case "netagent":
	case "controller":
	case "dual":
	default:
		return errors.New("--role must be one of controller, agent, netagent or dual")
	}

	return nil
}

func deploymentDirectory() string {
	u, err := user.Current()
	d := "."
	if err == nil {
		d = u.HomeDir
	}
	return path.Join(d, ".ciao-deployment")
}

func csrPath(ssntpRole ssntp.Role) string {
	return path.Join(deploymentDirectory(), deploy.CSRName(ssntpRole))
}

func privPath(ssntpRole ssntp.Role) string {
	return path.Join(deploymentDirectory(), deploy.PrivName(ssntpRole))
}

// Generate and upload. Saving to disk for subsequent certificate fetching.
func uploadCSR() (fp string, err error) {
	var csrOutput, privKeyOutput bytes.Buffer
	hs, err := os.Hostname()
	if err != nil {
		hs = "localhost"
	}

	hosts := []string{hs}

	request := certs.CreateCertificateRequest(false, "Ciao Deployment", "", hosts, nil)

	err = certs.CreateCSR(request, false, &csrOutput, &privKeyOutput)
	if err != nil {
		return "", errors.Wrap(err, "Error creating CSR")
	}

	csrBlock, _ := pem.Decode(csrOutput.Bytes())
	certReq, _ := x509.ParseCertificateRequest(csrBlock.Bytes)
	fp = certs.FingerPrint(certReq)
	fmt.Printf("Requesting CSR with fingerprint: %s\n", fp)

	url := fmt.Sprintf("%s/sign/%s/%s", deployServerHost, token, role)
	fmt.Printf("Uploading CSR to %s\n", url)

	defer func() {
		if err != nil {
			os.Remove(csrPath(ssntpRole))
			os.Remove(privPath(ssntpRole))
		}
	}()

	err = ioutil.WriteFile(csrPath(ssntpRole), csrOutput.Bytes(), 0600)
	if err != nil {
		return "", errors.Wrap(err, "Error saving CSR to file")
	}

	err = ioutil.WriteFile(privPath(ssntpRole), privKeyOutput.Bytes(), 0600)
	if err != nil {
		return "", errors.Wrap(err, "Error saving private key to file")
	}

	r, err := http.NewRequest("PUT", url, &csrOutput)
	if err != nil {
		return "", errors.Wrap(err, "Error requesting signing of CSR")
	}

	r.Header.Set("Content-Type", "text/plain")

	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		return "", errors.Wrap(err, "Error requesting signing of CSR")
	}
	res.Body.Close()

	return fp, nil
}

// Request the signed certificate from the server and save to system location.
func downloadSignedCert(systemCertPath, fp string) error {
	url := fmt.Sprintf("%s/cert/%s/%s", deployServerHost, token, fp)
	fmt.Printf("Downloading certificate: %s\n", url)
	r, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "Error downloading certificate")
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("Expected OK when downloading certificate: %s", r.Status)
	}

	var signedCert bytes.Buffer
	io.Copy(&signedCert, r.Body)
	certBlock, _ := pem.Decode(signedCert.Bytes())
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return errors.Wrap(err, "Error parsing downloaded certificate")
	}

	fp = certs.FingerPrint(cert)
	fmt.Printf("Got certificate with fingerprint: %s\n", fp)

	privKey, err := os.Open(privPath(ssntpRole))
	if err != nil {
		return errors.Wrap(err, "Error opening private key file")
	}

	var certOutput bytes.Buffer
	err = certs.AddPrivateKeyToCert(&signedCert, privKey, &certOutput)
	if err != nil {
		return errors.Wrap(err, "Error adding private key to cert")
	}

	f, err := ioutil.TempFile("", deploy.CertName(ssntpRole))
	if err != nil {
		return errors.Wrap(err, "Error creating certificate file")
	}
	defer f.Close()
	defer os.Remove(f.Name())

	_, err = io.Copy(f, &certOutput)
	if err != nil {
		return errors.Wrap(err, "Error copying certificate to file")
	}

	if err := deploy.SudoCopyFile(systemCertPath, f.Name()); err != nil {
		return errors.Wrap(err, "Error copying certificate to final destination")
	}

	fmt.Printf("Certificate installed in: %s\n", systemCertPath)
	return nil
}

func getCertFingerprint(path string) (string, error) {
	certBytes, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return "", err
	}
	if err != nil {
		return "", errors.Wrap(err, "Error reading certificate from file")
	}

	certBlock, _ := pem.Decode(certBytes)
	if certBlock == nil {
		return "", errors.Wrap(err, "Error decoding PEM data")
	}

	var cert interface{}
	if certBlock.Type == "CERTIFICATE REQUEST" {
		cert, err = x509.ParseCertificateRequest(certBlock.Bytes)
		if err != nil {
			return "", errors.Wrap(err, "Error parsing certificate")
		}
	} else if certBlock.Type == "CERTIFICATE" {
		cert, err = x509.ParseCertificate(certBlock.Bytes)
		if err != nil {
			return "", errors.Wrap(err, "Error parsing certificate")
		}
	} else {
		return "", errors.New("Unknown PEM type")
	}

	fp := certs.FingerPrint(cert)
	return fp, nil
}

// If the CSR certificate is locally available then return the fingerprint from it, otherwise generate a new CSR and post to the server.
func getCSRFingerprint() (string, error) {
	path := csrPath(ssntpRole)
	if !force {
		fp, err := getCertFingerprint(path)
		if err == nil {
			fmt.Printf("CSR already present. Fingerprint: %s\n", fp)
			return fp, nil
		}

		if !os.IsNotExist(err) {
			return "", errors.Wrap(err, "Error getting fingerprint")
		}
	}

	fp, err := uploadCSR()

	if err != nil {
		return "", errors.Wrap(err, "Error getting fingerprint")
	}
	return fp, nil
}

// Provides path to role certificate. If not locally available tries to request from the server possibly creating a
// CSR in the process and submitting that to the server (uses a local CSR if already present.)
func obtainCert() (string, error) {
	systemCertPath := path.Join(systemPKILocation, deploy.CertName(ssntpRole))

	// TODO: Check if certificate installed is valid
	if !force {
		_, err := os.Stat(systemCertPath)

		if err == nil {
			fmt.Printf("Certificate already installed. Skipping download.\n")
			return systemCertPath, nil
		}

		if !os.IsNotExist(err) {
			return "", errors.Wrap(err, "Error stat()ing certificate path")
		}
	}

	fp, err := getCSRFingerprint()
	if err != nil {
		return "", errors.Wrap(err, "error getting CSR fingerprint")
	}

	err = downloadSignedCert(systemCertPath, fp)
	if err != nil {
		return "", errors.Wrap(err, "error downloading signed certificate")
	}

	// Cleanup saved CSR and private key
	os.Remove(privPath(ssntpRole))
	os.Remove(csrPath(ssntpRole))

	return systemCertPath, nil
}

// Provides path to CA certificate. If not locally available then this is downloaded from the server.
func obtainCACert() (string, error) {
	systemCaCertPath := path.Join(systemPKILocation, "CAcert.pem")

	// TODO: Check if certificate installed is valid
	if !force {
		_, err := os.Stat(systemCaCertPath)

		if err == nil {
			fmt.Printf("CA certificate already installed. Skipping download.\n")
			return systemCaCertPath, nil
		}

		if !os.IsNotExist(err) {
			return "", errors.Wrap(err, "Error stat()ing CA certificate path")
		}
	}

	url := fmt.Sprintf("%s/cacert", deployServerHost)
	fmt.Printf("Downloading CA certificate: %s\n", url)
	r, err := http.Get(url)
	if err != nil {
		return "", errors.Wrap(err, "Error downloading certificate")
	}
	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Expected OK when downloading certificate: %s", r.Status)
	}

	var certOutput bytes.Buffer
	io.Copy(&certOutput, r.Body)
	certBlock, _ := pem.Decode(certOutput.Bytes())
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return "", errors.Wrap(err, "Error parsing CA certificate")
	}

	fp := certs.FingerPrint(cert)
	fmt.Printf("Got CA cert with fingerprint: %s\n", fp)

	f, err := ioutil.TempFile("", "CAcert.pem")
	if err != nil {
		return "", errors.Wrap(err, "Error creating CA certificate file")
	}
	defer f.Close()
	defer os.Remove(f.Name())

	_, err = io.Copy(f, &certOutput)
	if err != nil {
		return "", errors.Wrap(err, "Error copying CA certificate to file")
	}

	if err := f.Chmod(0644); err != nil {
		return "", errors.Wrap(err, "Error changing mode on certificate")
	}

	if err := deploy.SudoCopyFile(systemCaCertPath, f.Name()); err != nil {
		return "", errors.Wrap(err, "Error copying CA certificate to final destination")
	}

	fmt.Printf("CA certificate installed in: %s\n", systemCaCertPath)
	return systemCaCertPath, nil
}

func refreshClient() error {
	if !downloadClient {
		return nil
	}

	downloadedClientPath, err := deploy.DownloadTool(deployServerHost, "ciao-deploy-client")
	if err != nil {
		return errors.Wrap(err, "Error when downloading client")
	}
	defer os.Remove(downloadedClientPath)

	fmt.Println("Downloaded client")

	fmt.Println("Relaunching client")
	args := append([]string{downloadedClientPath, "--download-client=false"}, os.Args[1:]...)
	err = syscall.Exec(args[0], args, os.Environ())
	if err != nil {
		return errors.Wrap(err, "Error when exec()ing to new client")
	}

	return nil
}

func prepare() error {
	if err := os.Mkdir(deploymentDirectory(), 0700); err != nil && !os.IsExist(err) {
		return errors.Wrap(err, "Error creating deployment directory")
	}

	err := deploy.SudoMakeDirectory(systemPKILocation)
	if err != nil {
		return errors.Wrap(err, "Error making system PKI directory")
	}

	return nil
}

func installController(caCertPath, certPath string) error {
	toolPath, err := deploy.DownloadTool(deployServerHost, "ciao-controller")
	if err != nil {
		return errors.Wrap(err, "Error downloading controller")
	}
	defer os.Remove(toolPath)

	return deploy.InstallTool(toolPath, "ciao-controller", caCertPath, certPath)
}

func installLauncher(caCertPath, certPath string) error {
	toolPath, err := deploy.DownloadTool(deployServerHost, "ciao-launcher")
	if err != nil {
		return errors.Wrap(err, "Error downloading launcher")
	}
	defer os.Remove(toolPath)

	return deploy.InstallTool(toolPath, "ciao-launcher", caCertPath, certPath)
}

func installTool(caCertPath, certPath string) error {
	switch role {
	case "agent":
		fallthrough
	case "netagent":
		fallthrough
	case "dual":
		return installLauncher(caCertPath, certPath)
	case "controller":
		return installController(caCertPath, certPath)
	default:
		return errors.New("Unknown role to provision")
	}
}

func main() {
	flag.Parse()

	if err := validateFlags(); err != nil {
		fmt.Fprintf(os.Stderr, "Error validating flags: %s\n", err)
		os.Exit(1)
	}

	ssntpRole = roleToSSNTPRole(role)

	if err := prepare(); err != nil {
		fmt.Fprintf(os.Stderr, "Error preparing client: %s\n", err)
		os.Exit(1)
	}

	if err := refreshClient(); err != nil {
		fmt.Fprintf(os.Stderr, "Errror refreshing client: %s\n", err)
		os.Exit(1)
	}

	caCertPath, err := obtainCACert()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error obtaining CA certficate: %s\n", err)
		os.Exit(1)
	}

	certPath, err := obtainCert()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error obtaining certificate: %s\n", err)
		os.Exit(1)
	}

	if err := installTool(caCertPath, certPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error installing tool: %s\n", err)
		os.Exit(1)
	}
}

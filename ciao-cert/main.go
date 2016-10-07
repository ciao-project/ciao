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
// Initial implementation based on
//    golang/src/pkg/crypto/tls/generate_cert.go
//
// which is:
//
//    Copyright 2009 The Go Authors. All rights reserved.
//    Use of this source code is governed by a BSD-style
//    license that can be found in the golang LICENSE file.
//

package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	"github.com/01org/ciao/ssntp"
)

var (
	host         = flag.String("host", "", "Comma-separated hostnames to generate a certificate for")
	mgmtIP       = flag.String("ip", "", "Comma-separated IPs to generate a certificate for")
	serverCert   = flag.String("server-cert", "", "Server certificate for signing a client one")
	isServer     = flag.Bool("server", false, "Whether this cert should be a server one")
	verify       = flag.Bool("verify", false, "Verify client certificate")
	isElliptic   = flag.Bool("elliptic-key", false, "Use elliptic curve algorithms")
	email        = flag.String("email", "ciao-devel@lists.clearlinux.org", "Certificate email address")
	organization = flag.String("organization", "", "Certificates organization")
	installDir   = flag.String("directory", ".", "Installation directory")
)

func verifyCert(CACert string, certName string) {
	if *isServer == true || *verify == false {
		return
	}

	bytesServerCert, err := ioutil.ReadFile(CACert)
	if err != nil {
		log.Printf("Could not load [%s] %s", CACert, err)
	}

	bytesClientCert, err := ioutil.ReadFile(certName)
	if err != nil {
		log.Printf("Could not load [%s] %s", certName, err)
	}

	blockClient, _ := pem.Decode(bytesClientCert)
	certClient, err := x509.ParseCertificate(blockClient.Bytes)
	if err != nil {
		log.Printf("Could not parse [%s] %s", certName, err)
	}

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(bytesServerCert)
	if !ok {
		log.Printf("Could not add CA cert to poll")
	}

	opts := x509.VerifyOptions{
		Roots: roots,
	}

	if _, err = certClient.Verify(opts); err != nil {
		log.Printf("Failed to verify certificate: %s", err)
	}
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) (*pem.Block, error) {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}, nil
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil, fmt.Errorf("Unable to marshal ECDSA private key: %v", err)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}, nil
	default:
		return nil, fmt.Errorf("No private key found")
	}
}

func keyFromPemBlock(block *pem.Block) (interface{}, error) {
	if block.Type == "EC PRIVATE KEY" {
		return x509.ParseECPrivateKey(block.Bytes)
	} else {
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	}
}

func addOIDs(role ssntp.Role, oids []asn1.ObjectIdentifier) []asn1.ObjectIdentifier {
	if role.IsAgent() {
		oids = append(oids, ssntp.RoleAgentOID)
	}

	if role.IsScheduler() {
		oids = append(oids, ssntp.RoleSchedulerOID)
	}

	if role.IsController() {
		oids = append(oids, ssntp.RoleControllerOID)
	}

	if role.IsNetAgent() {
		oids = append(oids, ssntp.RoleNetAgentOID)
	}

	if role.IsServer() {
		oids = append(oids, ssntp.RoleServerOID)
	}

	if role.IsCNCIAgent() {
		oids = append(oids, ssntp.RoleCNCIAgentOID)
	}

	return oids
}

func instructionDisplay(server bool, CAcert string, Cert string) {
	if server {
		fmt.Printf("--------------------------------------------------------\n")
		fmt.Printf("CA certificate:     [%s]\n", CAcert)
		fmt.Printf("Server certificate: [%s]\n", Cert)
		fmt.Printf("--------------------------------------------------------\n")
		fmt.Printf("You should now copy \"%s\" and \"%s\" ", CAcert, Cert)
		fmt.Printf("to a safe location of your choice, and pass them to your ")
		fmt.Printf("SSNTP server through its Config CAcert and Cert fields.\n")
	} else {
		fmt.Printf("--------------------------------------------------------\n")
		fmt.Printf("CA certificate: [%s]\n", CAcert)
		fmt.Printf("Client certificate: [%s]\n", Cert)
		fmt.Printf("--------------------------------------------------------\n")
		fmt.Printf("You should now copy \"%s\" and \"%s\" ", CAcert, Cert)
		fmt.Printf("to a safe location of your choice, and pass them to your ")
		fmt.Printf("SSNTP client through its Config CAcert and Cert fields.\n")
	}
}

func getFirstHost() string {
	hosts := strings.Split(*host, ",")
	return hosts[0]
}

func generatePrivateKey(ell bool) (interface{}, error) {
	if ell == false {
		return rsa.GenerateKey(rand.Reader, 2048)
	} else {
		return ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	}
}

func checkCompulsoryOptions() {
	if *host == "" {
		log.Fatalf("Missing required --host parameter")
	}

	if *isServer == false && *serverCert == "" {
		log.Fatalf("Missing required --server-cert parameter")
	}
}

func addMgmtIPs(mgmtIPs []string, ips []net.IP) []net.IP {
	for _, i := range mgmtIPs {
		if ip := net.ParseIP(i); ip != nil {
			ips = append(ips, ip)
		}
	}

	return ips
}

func addDNSNames(hosts []string, names []string) []string {
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			continue
		} else {
			names = append(names, h)
		}
	}

	return names
}

// createCertTemplate provides the certificate template from which client or server certificated can be derived.
func createCertTemplate(role ssntp.Role, organization string, email string, hosts []string, mgmtIPs []string) (*x509.Certificate, error) {
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("Gailed to generate certificate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{organization},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		EmailAddresses:        []string{email},
		BasicConstraintsValid: true,
	}

	template.DNSNames = addDNSNames(hosts, template.DNSNames)
	template.IPAddresses = addMgmtIPs(mgmtIPs, template.IPAddresses)
	template.UnknownExtKeyUsage = addOIDs(role, template.UnknownExtKeyUsage)
	return &template, nil
}

// createServerCert creates the server certificate and the CA certificate. Both are written out PEM encoded.
func createServerCert(template *x509.Certificate, useElliptic bool, certOutput io.Writer, caCertOutput io.Writer) error {
	priv, err := generatePrivateKey(useElliptic)
	if err != nil {
		return fmt.Errorf("Unable to create private key: %v", err)
	}

	template.IsCA = true

	// Create self-signed certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, publicKey(priv), priv)
	if err != nil {
		return fmt.Errorf("Unable to create server certificate: %v", err)
	}

	// Write out CA cert
	err = pem.Encode(caCertOutput, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err != nil {
		return fmt.Errorf("Unable to encode PEM block: %v", err)
	}

	// Write out certificate (including private key)
	err = pem.Encode(certOutput, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err != nil {
		return fmt.Errorf("Unable to encode PEM block: %v", err)
	}
	block, err := pemBlockForKey(priv)
	if err != nil {
		return fmt.Errorf("Unable to get PEM block from key: %v", err)
	}
	err = pem.Encode(certOutput, block)
	if err != nil {
		return fmt.Errorf("Unable to encode PEM block: %v", err)
	}

	return nil
}

// createClientCert creates the client certificate signed by the giver server certificate. It is written PEM encoded.
func createClientCert(template *x509.Certificate, useElliptic bool, serverCert []byte, certOutput io.Writer) error {
	priv, err := generatePrivateKey(useElliptic)
	if err != nil {
		return fmt.Errorf("Unable to create private key: %v", err)
	}

	// Parent public key first
	certBlock, rest := pem.Decode(serverCert)
	parentCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return fmt.Errorf("Unable to parse server cert: %v", err)
	}

	// Parent private key
	privKeyBlock, _ := pem.Decode(rest)
	if privKeyBlock == nil {
		return fmt.Errorf("Unable to extract private key from server cert: %v", err)
	}

	serverPrivKey, err := keyFromPemBlock(privKeyBlock)
	if err != nil {
		return fmt.Errorf("Unable to parse private key from server cert: %v", err)
	}

	// Create certificate signed by private key from serverCert
	derBytes, err := x509.CreateCertificate(rand.Reader, template, parentCert, publicKey(priv), serverPrivKey)
	if err != nil {
		return fmt.Errorf("Unable to create client certificate: %v", err)
	}

	// Write out certificate (including private key)
	err = pem.Encode(certOutput, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err != nil {
		return fmt.Errorf("Unable to encode PEM block: %v", err)
	}

	block, err := pemBlockForKey(priv)
	if err != nil {
		return fmt.Errorf("Unable to get PEM block from key: %v", err)
	}
	err = pem.Encode(certOutput, block)
	if err != nil {
		return fmt.Errorf("Unable to encode PEM block: %v", err)
	}

	return nil
}

func main() {
	var role ssntp.Role

	flag.Var(&role, "role", "Comma separated list of SSNTP role [agent, scheduler, controller, netagent, server, cnciagent]")
	flag.Parse()

	checkCompulsoryOptions()
	mgmtIPs := strings.Split(*mgmtIP, ",")
	hosts := strings.Split(*host, ",")
	template, err := createCertTemplate(role, *organization, *email, hosts, mgmtIPs)
	if err != nil {
		log.Fatalf("Failed to create certificate template: %v", err)
	}

	firstHost := getFirstHost()
	CAcertName := fmt.Sprintf("%s/CAcert-%s.pem", *installDir, firstHost)
	certName := fmt.Sprintf("%s/cert-%s%s.pem", *installDir, role.String(), firstHost)
	if *isServer == true {
		CAcertOut, err := os.Create(CAcertName)
		if err != nil {
			log.Fatalf("Failed to open %s for writing: %s", CAcertName, err)
		}
		certOut, err := os.Create(certName)
		if err != nil {
			log.Fatalf("Failed to open %s for writing: %s", certName, err)
		}
		err = createServerCert(template, *isElliptic, certOut, CAcertOut)
		if err != nil {
			log.Fatalf("Failed to create certificate: %v", err)
		}
		err = certOut.Close()
		if err != nil {
			log.Fatalf("Error closing file %s: %v", certName, err)
		}
		err = CAcertOut.Close()
		if err != nil {
			log.Fatalf("Error closing file %s: %v", CAcertName, err)
		}
	} else {
		// Need to fetch the public and private key from the signer
		bytesCert, err := ioutil.ReadFile(*serverCert)
		if err != nil {
			log.Fatalf("Could not load %s", *serverCert)
		}

		// Create certificate: Concatenate public and private key
		certOut, err := os.Create(certName)
		if err != nil {
			log.Fatalf("Failed to open %s for writing: %s", certName, err)
		}

		err = createClientCert(template, *isElliptic, bytesCert, certOut)
		if err != nil {
			log.Fatalf("Failed to create certificate: %v", err)
		}
		err = certOut.Close()
		if err != nil {
			log.Fatalf("Error closing file %s: %v", certName, err)
		}
	}

	verifyCert(*serverCert, certName)
	instructionDisplay(*isServer, CAcertName, certName)
}

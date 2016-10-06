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

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}

func keyFromPemBlock(block *pem.Block) (serverPrivKey interface{}, err error) {
	if block.Type == "EC PRIVATE KEY" {
		serverPrivKey, err = x509.ParseECPrivateKey(block.Bytes)
	} else {
		serverPrivKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	}
	return
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

func generatePrivateKey(ell bool) interface{} {
	var priv interface{}
	var err error

	if ell == false {
		priv, err = rsa.GenerateKey(rand.Reader, 2048)
	} else {
		priv, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	}

	if err != nil {
		log.Fatalf("failed to generate private key: %s", err)
	}

	return priv
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

func main() {
	var serverPrivKey interface{}
	var err error
	var parentCert *x509.Certificate
	var role ssntp.Role

	flag.Var(&role, "role", "Comma separated list of SSNTP role [agent, scheduler, controller, netagent, server, cnciagent]")
	flag.Parse()

	checkCompulsoryOptions()
	priv := generatePrivateKey(*isElliptic)

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
		template.IsCA = true
		parentCert = template
		serverPrivKey = priv
	} else {
		// Need to fetch the public and private key from the signer
		bytesCert, err := ioutil.ReadFile(*serverCert)
		if err != nil {
			log.Fatalf("Could not load %s", *serverCert)
		}

		// Parent public key first
		certBlock, rest := pem.Decode(bytesCert)
		if certBlock == nil {
		}
		cert, err := x509.ParseCertificate(certBlock.Bytes)
		if err != nil {
			log.Fatalf("Could not parse %s %s", *serverCert, err)
		}
		parentCert = cert

		// Parent private key
		privKeyBlock, _ := pem.Decode(rest)
		if privKeyBlock == nil {
			log.Fatalf("Invalid server certificate %s", certName)
		}
		serverPrivKey, err = keyFromPemBlock(privKeyBlock)
		if err != nil {
			log.Fatalf("Could not get server private key %s", err)
		}
	}

	// The certificate is created
	// Self signed for the server case
	// Signed by --server-cert for the client case
	derBytes, err := x509.CreateCertificate(rand.Reader, template, parentCert, publicKey(priv), serverPrivKey)
	if err != nil {
		log.Fatalf("Failed to create certificate: %s", err)
	}

	// Create CA certificate, i.e. the server public key
	if *isServer == true {
		CAcertOut, err := os.Create(CAcertName)
		if err != nil {
			log.Fatalf("failed to open %s for writing: %s", certName, err)
		}
		pem.Encode(CAcertOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
		CAcertOut.Close()
	}

	// Create certificate: Concatenate public and private key
	certOut, err := os.Create(certName)
	if err != nil {
		log.Fatalf("failed to open %s for writing: %s", certName, err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	pem.Encode(certOut, pemBlockForKey(priv))
	certOut.Close()

	verifyCert(*serverCert, certName)
	instructionDisplay(*isServer, CAcertName, certName)
}

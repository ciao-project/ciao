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
	"encoding/pem"
	"flag"
	"fmt"
	"github.com/01org/ciao/ssntp"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"os"
	"strings"
	"time"
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
)

func verifyCert(CACert string, certName string) {
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

func main() {
	var priv, serverPrivKey interface{}
	var err error
	var CAcertName, certName string
	var parentCert x509.Certificate
	var role ssntp.Role

	flag.Var(&role, "role", "SSNTP role [agent, scheduler, controller, netagent, server, cnciagent]")
	flag.Parse()

	flag.Parse()

	if len(*host) == 0 {
		log.Fatalf("Missing required --host parameter")
	}

	if *isServer == false && len(*serverCert) == 0 {
		log.Fatalf("Missing required --server-cert parameter")
	}

	if *isElliptic == false {
		priv, err = rsa.GenerateKey(rand.Reader, 2048)
	} else {
		priv, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	}
	if err != nil {
		log.Fatalf("failed to generate private key: %s", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{*organization},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		EmailAddresses:        []string{*email},
		BasicConstraintsValid: true,
	}

	hosts := strings.Split(*host, ",")
	firstHost := hosts[0]
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			continue
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	mgmtIPs := strings.Split(*mgmtIP, ",")
	for _, i := range mgmtIPs {
		if ip := net.ParseIP(i); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		}
	}

	switch role {
	case ssntp.AGENT:
		template.UnknownExtKeyUsage = append(template.UnknownExtKeyUsage, ssntp.RoleAgentOID)
	case ssntp.SCHEDULER:
		template.UnknownExtKeyUsage = append(template.UnknownExtKeyUsage, ssntp.RoleSchedulerOID)
	case ssntp.Controller:
		template.UnknownExtKeyUsage = append(template.UnknownExtKeyUsage, ssntp.RoleControllerOID)
	case ssntp.NETAGENT:
		template.UnknownExtKeyUsage = append(template.UnknownExtKeyUsage, ssntp.RoleNetAgentOID)
	case ssntp.SERVER:
		template.UnknownExtKeyUsage = append(template.UnknownExtKeyUsage, ssntp.RoleServerOID)
	case ssntp.CNCIAGENT:
		template.UnknownExtKeyUsage = append(template.UnknownExtKeyUsage, ssntp.RoleCNCIAgentOID)
	default:
		break
	}

	CAcertName = fmt.Sprintf("CAcert-%s.pem", firstHost)
	if *isServer == true {
		template.IsCA = true
		certName = fmt.Sprintf("cert-%s-%s.pem", role.String(), firstHost)
		parentCert = template
		serverPrivKey = priv
	} else {
		certName = fmt.Sprintf("cert-%s-%s.pem", role.String(), firstHost)
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
		parentCert = *cert

		// Parent private key
		privKeyBlock, _ := pem.Decode(rest)
		if privKeyBlock == nil {
			log.Fatalf("Invalid server certificate %s", certName)
		}
		if *isElliptic == false {
			serverPrivKey, err = x509.ParsePKCS1PrivateKey(privKeyBlock.Bytes)
		} else {
			serverPrivKey, err = x509.ParseECPrivateKey(privKeyBlock.Bytes)
		}
		if err != nil {
			log.Fatalf("Could not get server private key %s", err)
		}
	}

	// The certificate is created
	// Self signed for the server case
	// Signed by --server-cert for the client case
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &parentCert, publicKey(priv), serverPrivKey)
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

	if *isServer == false && *verify == true {
		verifyCert(*serverCert, certName)
	}

	instructionDisplay(*isServer, CAcertName, certName)
}

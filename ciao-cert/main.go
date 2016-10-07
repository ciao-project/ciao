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
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/01org/ciao/ssntp"
	"github.com/01org/ciao/ssntp/certs"
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

func checkCompulsoryOptions() {
	if *host == "" {
		log.Fatalf("Missing required --host parameter")
	}

	if *isServer == false && *serverCert == "" {
		log.Fatalf("Missing required --server-cert parameter")
	}
}

func main() {
	var role ssntp.Role

	flag.Var(&role, "role", "Comma separated list of SSNTP role [agent, scheduler, controller, netagent, server, cnciagent]")
	flag.Parse()

	checkCompulsoryOptions()
	mgmtIPs := strings.Split(*mgmtIP, ",")
	hosts := strings.Split(*host, ",")
	template, err := certs.CreateCertTemplate(role, *organization, *email, hosts, mgmtIPs)
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
		err = certs.CreateServerCert(template, *isElliptic, certOut, CAcertOut)
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

		err = certs.CreateClientCert(template, *isElliptic, bytesCert, certOut)
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

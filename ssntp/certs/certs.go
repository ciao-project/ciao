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

package certs

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"time"

	"io/ioutil"

	"github.com/ciao-project/ciao/ssntp"
	"github.com/pkg/errors"
)

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
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
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

func generatePrivateKey() (interface{}, error) {
	return rsa.GenerateKey(rand.Reader, 2048)

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

// CreateCertTemplate provides the certificate template from which trust anchor or derivative certificates can be derived.
func CreateCertTemplate(role ssntp.Role, organization string, email string, hosts []string, mgmtIPs []string) (*x509.Certificate, error) {
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate certificate serial number: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{organization},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		EmailAddresses:        []string{email},
		BasicConstraintsValid: true,
	}

	template.DNSNames = addDNSNames(hosts, template.DNSNames)
	template.IPAddresses = addMgmtIPs(mgmtIPs, template.IPAddresses)
	template.UnknownExtKeyUsage = addOIDs(role, template.UnknownExtKeyUsage)
	return &template, nil
}

// CreateAnchorCert creates the trust anchor certificate and the CA certificate. Both are written out PEM encoded.
func CreateAnchorCert(template *x509.Certificate, certOutput io.Writer, caCertOutput io.Writer) error {
	priv, err := generatePrivateKey()
	if err != nil {
		return fmt.Errorf("Unable to create private key: %v", err)
	}

	template.IsCA = true
	template.KeyUsage = template.KeyUsage | x509.KeyUsageCertSign

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

// CreateCert creates the certificate signed by the giver trust anchor certificate. It is written PEM encoded.
func CreateCert(template *x509.Certificate, anchorCert []byte, certOutput io.Writer) error {
	priv, err := generatePrivateKey()
	if err != nil {
		return fmt.Errorf("Unable to create private key: %v", err)
	}

	template.IsCA = false

	// Parent public key first
	certBlock, rest := pem.Decode(anchorCert)
	parentCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return fmt.Errorf("Unable to parse anchor cert: %v", err)
	}

	// Parent private key
	privKeyBlock, _ := pem.Decode(rest)
	if privKeyBlock == nil {
		return fmt.Errorf("Unable to extract private key from anchor cert: %v", err)
	}

	anchorPrivKey, err := keyFromPemBlock(privKeyBlock)
	if err != nil {
		return fmt.Errorf("Unable to parse private key from anchor cert: %v", err)
	}

	// Create certificate signed by private key from anchorCert
	derBytes, err := x509.CreateCertificate(rand.Reader, template, parentCert, publicKey(priv), anchorPrivKey)
	if err != nil {
		return fmt.Errorf("Unable to create certificate: %v", err)
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

func createCertTemplateFromCSR(role ssntp.Role, request *x509.CertificateRequest) (*x509.Certificate, error) {
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate certificate serial number")
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      request.Subject,
		NotBefore:    notBefore,
		NotAfter:     notAfter,

		Signature:          request.Signature,
		SignatureAlgorithm: request.SignatureAlgorithm,
		PublicKeyAlgorithm: request.PublicKeyAlgorithm,
		PublicKey:          request.PublicKey,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		EmailAddresses:        request.EmailAddresses,
		BasicConstraintsValid: true,
	}

	template.DNSNames = request.DNSNames
	template.IPAddresses = request.IPAddresses
	template.UnknownExtKeyUsage = addOIDs(role, template.UnknownExtKeyUsage)
	return &template, nil
}

// CreateCertFromCSR creates a certificate from a CSR signed by the given anchor certificate. It is written in PEM format.
func CreateCertFromCSR(role ssntp.Role, csr []byte, anchorCert []byte, certOutput io.Writer) error {
	// Parent public key first
	certBlock, rest := pem.Decode(anchorCert)
	parentCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return errors.Wrap(err, "Unable to parse anchor cert")
	}

	// Parent private key
	privKeyBlock, _ := pem.Decode(rest)
	if privKeyBlock == nil {
		return errors.Wrap(err, "Unable to extract private key from anchor cert")
	}

	anchorPrivKey, err := keyFromPemBlock(privKeyBlock)
	if err != nil {
		return errors.Wrap(err, "Unable to parse private key from anchor cert")
	}

	// Decode and parse csr
	csrBlock, _ := pem.Decode(csr)
	request, err := x509.ParseCertificateRequest(csrBlock.Bytes)
	if err != nil {
		return errors.Wrap(err, "Unable to parse CSR")
	}

	// Check signature on request (proves that creator of csr has private key)
	err = request.CheckSignature()
	if err != nil {
		return errors.Wrap(err, "Signature check on request failed")
	}

	template, err := createCertTemplateFromCSR(role, request)
	if err != nil {
		return errors.Wrap(err, "Unable to create template from CSR")
	}

	// Create certificate signed by private key from anchorCert
	derBytes, err := x509.CreateCertificate(rand.Reader, template, parentCert, request.PublicKey, anchorPrivKey)
	if err != nil {
		return errors.Wrap(err, "Unable to create certificate")
	}

	// Write out certificate
	err = pem.Encode(certOutput, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err != nil {
		return errors.Wrap(err, "Unable to encode PEM block")
	}

	return nil
}

// CreateCertificateRequest creates a certificate request template from the supplied details.
func CreateCertificateRequest(organization string, email string, hosts []string, mgmtIPs []string) *x509.CertificateRequest {
	request := x509.CertificateRequest{
		Subject: pkix.Name{
			Organization: []string{organization},
		},

		EmailAddresses: []string{email},
	}

	request.DNSNames = addDNSNames(hosts, request.DNSNames)
	request.IPAddresses = addMgmtIPs(mgmtIPs, request.IPAddresses)

	request.SignatureAlgorithm = x509.SHA1WithRSA

	return &request
}

// CreateCSR creates a CSR from the incoming template for a newly generated private key.
func CreateCSR(template *x509.CertificateRequest, csrOutput io.Writer, privKeyOutput io.Writer) error {
	// Generate private key
	priv, err := generatePrivateKey()
	if err != nil {
		return errors.Wrap(err, "error generating private key")
	}

	privBlock, err := pemBlockForKey(priv)
	if err != nil {
		return errors.Wrap(err, "error creating pem block for private key")
	}

	// Send private key to caller to save
	err = pem.Encode(privKeyOutput, privBlock)
	if err != nil {
		return errors.Wrap(err, "error encoding private key")
	}

	// Create csr
	csr, err := x509.CreateCertificateRequest(rand.Reader, template, priv)
	if err != nil {
		return errors.Wrap(err, "error creating certificate request")
	}

	// Send csr to calle to pass on
	err = pem.Encode(csrOutput, &pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csr,
	})
	if err != nil {
		return errors.Wrap(err, "error creating pem block for csr")
	}

	return nil
}

// AddPrivateKeyToCert adds a private key to existing certificate (created from signing a CSR)
func AddPrivateKeyToCert(certInput io.Reader, privKeyInput io.Reader, certOutput io.Writer) error {
	certData, err := ioutil.ReadAll(certInput)
	if err != nil {
		return errors.Wrap(err, "error reading certificate data")
	}

	privData, err := ioutil.ReadAll(privKeyInput)
	if err != nil {
		return errors.Wrap(err, "error reading private key")
	}

	certBlock, _ := pem.Decode(certData)
	privBlock, _ := pem.Decode(privData)

	// Write out certificate (including private key)
	err = pem.Encode(certOutput, certBlock)
	if err != nil {
		return errors.Wrap(err, "Unable to encode PEM block")
	}

	err = pem.Encode(certOutput, privBlock)
	if err != nil {
		return errors.Wrap(err, "Unable to encode PEM block")
	}

	return nil
}

// VerifyCert verifies that bytesCert is valid in terms of the CA in bytesAnchorCert
func VerifyCert(bytesAnchorCert, bytesCert []byte) error {
	blockCert, _ := pem.Decode(bytesCert)
	cert, err := x509.ParseCertificate(blockCert.Bytes)
	if err != nil {
		return errors.Wrap(err, "error parsing certificate")
	}

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(bytesAnchorCert)
	if !ok {
		return errors.New("Could not add CA cert to poll")
	}

	opts := x509.VerifyOptions{
		Roots: roots,
	}

	if _, err = cert.Verify(opts); err != nil {
		return errors.Wrap(err, "failed to verify certificate")
	}
	return nil
}

// FingerPrint returns the SHA-256 fingerprint of the public key
func FingerPrint(c interface{}) string {
	var input *[]byte
	switch c.(type) {
	case *x509.CertificateRequest:
		input = &c.(*x509.CertificateRequest).RawSubjectPublicKeyInfo
	case *x509.Certificate:
		input = &c.(*x509.Certificate).RawSubjectPublicKeyInfo
	default:
		return ""
	}
	h := sha256.New()
	h.Write(*input)
	return fmt.Sprintf("%x", h.Sum(nil))
}

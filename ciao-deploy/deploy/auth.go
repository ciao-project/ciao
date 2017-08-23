// Copyright © 2017 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package deploy

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path"
	"time"

	"github.com/pkg/errors"
)

func createCertTemplate(username string, tenants []string) (*x509.Certificate, error) {
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate certificate serial number: %v", err)
	}

	subject := pkix.Name{}
	if username != "" {
		subject.CommonName = username
	}
	if len(tenants) > 0 {
		subject.Organization = tenants
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      subject,
		NotBefore:    notBefore,
		NotAfter:     notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		BasicConstraintsValid: true,
	}

	return &template, nil
}

// CreateAdminCert creates and installs the authentication certificates
func CreateAdminCert(ctx context.Context) (_ string, _ string, errOut error) {
	template, err := createCertTemplate("admin", []string{"admin"})
	if err != nil {
		return "", "", errors.Wrap(err, "Error creating certificate template")
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", errors.Wrap(err, "Unable to create private key")
	}

	template.IsCA = true
	template.KeyUsage = template.KeyUsage | x509.KeyUsageCertSign

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return "", "", errors.Wrap(err, "Error creating certificate")
	}

	caCertFile, err := ioutil.TempFile("", "auth-CA")
	if err != nil {
		return "", "", errors.Wrap(err, "Error creating temporary file")
	}
	defer func() { _ = caCertFile.Close() }()
	defer func() { _ = os.Remove(caCertFile.Name()) }()

	err = pem.Encode(caCertFile, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err != nil {
		return "", "", errors.Wrap(err, "Unable to encode PEM block")
	}

	certFile, err := ioutil.TempFile("", "auth-admin")
	if err != nil {
		return "", "", errors.Wrap(err, "Error creating temporary file")
	}
	defer func() { _ = certFile.Close() }()
	defer func() { _ = os.Remove(certFile.Name()) }()

	err = pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err != nil {
		return "", "", errors.Wrap(err, "Unable to encode PEM block")
	}

	err = pem.Encode(certFile,
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(priv),
		})
	if err != nil {
		return "", "", errors.Wrap(err, "Unable to encode PEM block")
	}

	if err := SudoMakeDirectory(ctx, ciaoPKIDir); err != nil {
		return "", "", errors.Wrap(err, "Error creating system PKI directory")
	}

	if err := os.Chmod(certFile.Name(), 0644); err != nil {
		return "", "", errors.Wrap(err, "Error chmod()ing anchor certificate")
	}

	if err := os.Chmod(caCertFile.Name(), 0644); err != nil {
		return "", "", errors.Wrap(err, "Error chmod()ing CA certificate")
	}

	caCertPath := path.Join(ciaoPKIDir, "auth-CA.pem")
	certPath := path.Join(ciaoPKIDir, "auth-admin.pem")

	if err := SudoCopyFile(ctx, certPath, certFile.Name()); err != nil {
		return "", "", errors.Wrap(err, "Error copying admin auth certificate to system location")
	}

	if err := SudoCopyFile(ctx, caCertPath, caCertFile.Name()); err != nil {
		_ = SudoRemoveFile(context.Background(), certPath)
		return "", "", errors.Wrap(err, "Error copying CA auth certificate to system location")
	}

	return caCertPath, certPath, nil
}

func loadAdminCert() (*x509.Certificate, *rsa.PrivateKey, error) {
	adminCertPath := path.Join(ciaoPKIDir, "auth-admin.pem")
	adminCertFileBytes, err := ioutil.ReadFile(adminCertPath)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error reading admin (anchor) certificate")
	}

	var adminCertBlock, adminPrivBlock, p *pem.Block
	for {
		p, adminCertFileBytes = pem.Decode(adminCertFileBytes)
		if p == nil {
			break
		}
		if p.Type == "CERTIFICATE" {
			adminCertBlock = p
		}
		if p.Type == "RSA PRIVATE KEY" {
			adminPrivBlock = p
		}
	}

	if adminCertBlock == nil {
		return nil, nil, errors.New("Unable to find certificate PEM block in data")
	}

	if adminPrivBlock == nil {
		return nil, nil, errors.New("Unable to find private key PEM block in data")
	}

	adminCert, err := x509.ParseCertificate(adminCertBlock.Bytes)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error parsing x509 certificate data")
	}

	adminPrivKey, err := x509.ParsePKCS1PrivateKey(adminPrivBlock.Bytes)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error parsing private key")
	}

	return adminCert, adminPrivKey, nil
}

// CreateUserCert creates a user certificate in current working directory
func CreateUserCert(ctx context.Context, username string, tenants []string) (_ string, errOut error) {
	adminCert, adminPrivKey, err := loadAdminCert()
	if err != nil {
		return "", errors.Wrap(err, "Error loading admin certificate")
	}

	template, err := createCertTemplate(username, tenants)
	if err != nil {
		return "", errors.Wrap(err, "Error creating certificate template")
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", errors.Wrap(err, "Unable to create private key")
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, adminCert, &priv.PublicKey, adminPrivKey)
	if err != nil {
		return "", errors.Wrap(err, "Error creating certificate")
	}

	certPath := fmt.Sprintf("auth-%s.pem", username)
	certFile, err := os.Create(certPath)
	if err != nil {
		return "", errors.Wrap(err, "Error creating file")
	}
	defer func() { _ = certFile.Close() }()
	defer func() {
		if errOut != nil {
			_ = os.Remove(certFile.Name())
		}
	}()

	err = pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err != nil {
		return "", errors.Wrap(err, "Unable to encode PEM block")
	}

	err = pem.Encode(certFile,
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(priv),
		})
	if err != nil {
		return "", errors.Wrap(err, "Unable to encode PEM block")
	}

	return certPath, nil
}

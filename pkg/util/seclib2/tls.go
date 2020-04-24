// Copyright 2019 Yunion
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

package seclib2

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

var CERT_SEP = []byte("-END CERTIFICATE-")

func findCertEndIndex(certBytes []byte) int {
	endpos := bytes.Index(certBytes, CERT_SEP)
	if endpos < 0 {
		return endpos
	}
	endpos += len(CERT_SEP)
	for endpos < len(certBytes) && certBytes[endpos] != '\n' {
		endpos += 1
	}
	return endpos
}

func splitCert(certBytes []byte) [][]byte {
	ret := make([][]byte, 0)
	for {
		endpos := findCertEndIndex(certBytes)
		if endpos > 0 {
			ret = append(ret, certBytes[:endpos])
			for endpos < len(certBytes) && certBytes[endpos] != '-' {
				endpos += 1
			}
			if endpos < len(certBytes) {
				certBytes = certBytes[endpos:]
			} else {
				break
			}
		}
	}
	return ret
}

func InitTLSConfigFromCertDatas(certPem, pkeyPem, caCertPem string) (*tls.Config, error) {
	cert, err := tls.X509KeyPair([]byte(certPem), []byte(pkeyPem))
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertData := []byte(caCertPem)
	for {
		var block *pem.Block
		block, caCertData = pem.Decode(caCertData)
		if block == nil {
			break
		}
		cacert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, errors.Wrap(err, "parse cacert file")
		}
		caCertPool.AddCert(cacert)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	return tlsConfig, nil
}

func InitTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	allCertPEM, err := ioutil.ReadFile(certFile)
	if err != nil {
		log.Errorf("read tls certfile fail %s", err)
		return nil, err
	}
	certPEMs := splitCert(allCertPEM)
	keyPEM, err := ioutil.ReadFile(keyFile)
	if err != nil {
		log.Errorf("read tls keyfile fail %s", err)
		return nil, err
	}
	cert, err := tls.X509KeyPair(certPEMs[0], keyPEM)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	for i := 1; i < len(certPEMs); i += 1 {
		caCertPool.AppendCertsFromPEM(certPEMs[i])
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	// tlsConfig.ServerName = "CN=*"
	tlsConfig.BuildNameToCertificate()
	return tlsConfig, nil
}

func InitTLSConfigWithCA(certFile, keyFile, caCertFile string) (*tls.Config, error) {
	allCertPEM, err := ioutil.ReadFile(certFile)
	if err != nil {
		log.Errorf("read tls certfile fail %s", err)
		return nil, err
	}
	certPEMs := splitCert(allCertPEM)
	keyPEM, err := ioutil.ReadFile(keyFile)
	if err != nil {
		log.Errorf("read tls keyfile fail %s", err)
		return nil, err
	}
	cert, err := tls.X509KeyPair(certPEMs[0], keyPEM)
	if err != nil {
		return nil, err
	}

	caCertPool := x509.NewCertPool()
	for i := 1; i < len(certPEMs); i += 1 {
		caCertPool.AppendCertsFromPEM(certPEMs[i])
	}

	data, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		return nil, errors.Wrap(err, "read cacert file")
	}
	for {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		cacert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, errors.Wrap(err, "parse cacert file")
		}
		caCertPool.AddCert(cacert)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig, nil
}

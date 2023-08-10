package quichelper

import (
	"crypto/x509"
	"encoding/pem"
	"github.com/pkg/errors"
	"os"
	"path"
)

func GetRootCertPool(certDirPath string) (*x509.CertPool, error) {
	caCertPath := path.Join(certDirPath, "ca.pem")
	caCertRaw, err := os.ReadFile(caCertPath)
	if err != nil {
		return nil, errors.Wrap(err, "os.ReadFile(caCertPath)")
	}

	p, _ := pem.Decode(caCertRaw)
	if p.Type != "CERTIFICATE" {
		return nil, errors.Wrap(err, "ca.pem Type != CERTIFICATE")
	}

	caCert, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		panic(err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(caCert)

	return certPool, nil
}

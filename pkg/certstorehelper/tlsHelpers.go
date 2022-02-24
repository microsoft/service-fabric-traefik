package certstorehelper

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
)

// ClientTLS holds the TLS specific configurations as client
// CA, Cert and Key can be either path or file contents.
type ClientTLS struct {
	CA                 string `json:"ca,omitempty" toml:"ca,omitempty" yaml:"ca,omitempty"`
	CAOptional         bool   `json:"caOptional,omitempty" toml:"caOptional,omitempty" yaml:"caOptional,omitempty" export:"true"`
	Cert               string `json:"cert,omitempty" toml:"cert,omitempty" yaml:"cert,omitempty"`
	Key                string `json:"key,omitempty" toml:"key,omitempty" yaml:"key,omitempty"`
	CertStoreSearchKey string `json:"certStoreSearchKey,omitempty" toml:"key,omitempty" yaml:"key,omitempty"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty" toml:"insecureSkipVerify,omitempty" yaml:"insecureSkipVerify,omitempty" export:"true"`
}

// CreateTLSConfig creates a TLS config from ClientTLS structures.
func (c *ClientTLS) CreateTLSConfig() (*tls.Config, error) {
	if c == nil {
		return nil, nil
	}

	var err error
	caPool := x509.NewCertPool()
	clientAuth := tls.NoClientCert
	if c.CA != "" {
		var ca []byte
		if _, errCA := os.Stat(c.CA); errCA == nil {
			ca, err = ioutil.ReadFile(c.CA)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA. %w", err)
			}
		} else {
			ca = []byte(c.CA)
		}

		if !caPool.AppendCertsFromPEM(ca) {
			return nil, fmt.Errorf("failed to parse CA")
		}

		if c.CAOptional {
			clientAuth = tls.VerifyClientCertIfGiven
		} else {
			clientAuth = tls.RequireAndVerifyClientCert
		}
	}

	if c.CertStoreSearchKey == "" && (len(c.Cert) == 0 && len(c.Key) == 0) {
		return nil, fmt.Errorf("TLS Certificate or Key file must be set when TLS configuration is created")
	}

	var certPtr *tls.Certificate
	if c.CertStoreSearchKey != "" {
		certPtr, err = searchCert(c.CertStoreSearchKey)
		if err != nil {
			return nil, err
		}
	}

	cert := tls.Certificate{}

	if certPtr != nil {
		cert = *certPtr
		log.Infof("Found certificate for string [%s]", c.CertStoreSearchKey)
	} else {
		_, errKeyIsFile := os.Stat(c.Key)

		if len(c.Cert) > 0 && len(c.Key) > 0 {
			if _, errCertIsFile := os.Stat(c.Cert); errCertIsFile == nil {
				if errKeyIsFile == nil {
					cert, err = tls.LoadX509KeyPair(c.Cert, c.Key)
					if err != nil {
						return nil, fmt.Errorf("failed to load TLS keypair: %w", err)
					}
				} else {
					return nil, fmt.Errorf("tls cert is a file, but tls key is not")
				}
			} else {
				if errKeyIsFile != nil {
					cert, err = tls.X509KeyPair([]byte(c.Cert), []byte(c.Key))
					if err != nil {
						return nil, fmt.Errorf("failed to load TLS keypair: %w", err)
					}
				} else {
					return nil, fmt.Errorf("TLS key is a file, but tls cert is not")
				}
			}
		}
	}

	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caPool,
		InsecureSkipVerify: c.InsecureSkipVerify, // nolint:gosec
		ClientAuth:         clientAuth,
	}, nil
}

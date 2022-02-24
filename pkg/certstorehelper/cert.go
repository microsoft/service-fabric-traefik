package certstorehelper

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
)

func loadPkcs8(path string) (*tls.Certificate, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cert tls.Certificate

	for {
		block, rest := pem.Decode(data)
		if block == nil {
			break
		}

		switch block.Type {
		case "CERTIFICATE":
			cert.Certificate = append(cert.Certificate, block.Bytes)
		case "PRIVATE KEY":
			key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				continue
			}

			cert.PrivateKey = key
		}

		data = rest
	}

	if len(cert.Certificate) == 0 {
		return nil, fmt.Errorf("no certificate in pem")
	}

	if cert.PrivateKey == nil {
		return nil, fmt.Errorf("no private key in pem")
	}

	return &cert, nil
}

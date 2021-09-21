// +build windows

package certstorehelper

import (
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"strings"

	"github.com/github/certstore"
)

func certHasKeyword(c *x509.Certificate, keyword string) bool {
	k := strings.ToLower(keyword)

	for _, s := range []string{
		c.Issuer.CommonName,
		c.Subject.CommonName,
		c.Subject.String(),
		c.Issuer.String(),
	} {
		if strings.Contains(strings.ToLower(s), k) {
			return true
		}
	}

	for _, dns := range c.DNSNames {
		if strings.Contains(strings.ToLower(dns), k) {
			return true
		}
	}

	thumb := fmt.Sprintf("%x", sha1.Sum(c.Raw))
	return strings.Contains(thumb, k)
}

func searchCertWithStore(fac func() (certstore.Store, error), keyword string) (*tls.Certificate, error) {
	store, err := fac()
	if err != nil {
		return nil, err
	}
	defer store.Close()

	idents, err := store.Identities()
	if err != nil {
		return nil, err
	}

	found := false

	var cert tls.Certificate
	for _, i := range idents {
		c, err := i.Certificate()

		if err != nil {
			continue
		}

		if certHasKeyword(c, keyword) {
			s, err := i.Signer()
			if err != nil {
				return nil, err
			}
			cert.Certificate = [][]byte{c.Raw}
			cert.PrivateKey = s

			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("cert not found")
	}

	return &cert, nil
}

func searchCert(keyword string) (*tls.Certificate, error) {
	log.Printf("searching for certificate containing string: [%v]", keyword)

	for _, fac := range []func() (certstore.Store, error){
		func() (certstore.Store, error) {
			log.Printf("searching [%v] in keystore LocalMachine/MY", keyword)
			return certstore.OpenStoreWindows("MY", certstore.StoreLocationLocalMachine)
		},
		func() (certstore.Store, error) {
			log.Printf("searching [%v] in keystore CurrentUser/MY", keyword)
			return certstore.OpenStoreWindows("MY", certstore.StoreLocationCurrentUser)
		},
	} {
		c, err := searchCertWithStore(fac, keyword)
		if err != nil {
			continue
		}

		return c, nil
	}

	return loadPkcs8(keyword)
}

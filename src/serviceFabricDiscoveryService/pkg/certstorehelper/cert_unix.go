// +build !windows

package certstorehelper

import (
	"crypto/tls"
)

func searchCert(keyword string) (*tls.Certificate, error) {
	return loadPkcs8(keyword)
}

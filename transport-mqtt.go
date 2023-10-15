package zerosvc

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/url"
)

func sanitizeClientID(id string) string {
	if len(id) <= 23 {
		return id
	} else {
		h := sha256.New()
		h.Write([]byte(id))
		hash := base64.URLEncoding.EncodeToString(h.Sum(nil))
		return id[len(id)-12:] + hash[:11]
	}
}

func getTLSConfigFromURL(url *url.URL) (tlsConfig *tls.Config, clientCertSubject string, err error) {
	var tlsCfg tls.Config
	if len(url.Query().Get("cert")) > 0 {
		certPath := url.Query().Get("cert")
		keyPath := url.Query().Get("certkey")
		if len(keyPath) == 0 {
			keyPath = certPath
		}
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, "", fmt.Errorf("error loading cert/key: %s", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
		parsedCert, err := x509.ParseCertificate(cert.Certificate[0])
		// make user-friendly clientId if we can parse the cert
		if err == nil {
			clientCertSubject = parsedCert.Subject.CommonName
		}
	}
	if len(url.Query().Get("ca")) > 0 {
		certpool := x509.NewCertPool()
		pem, err := ioutil.ReadFile(url.Query().Get("ca"))
		if err != nil {
			return nil, "", fmt.Errorf("error loading CA: %s", err)
		}
		certpool.AppendCertsFromPEM(pem)
		tlsCfg.RootCAs = certpool
	}
	return &tlsCfg, clientCertSubject, err
}

package auth

import (
	"crypto/x509"
	//	"encoding/pem"
	"fmt"
	"io/ioutil"
)

type Auth struct {
	root *x509.CertPool
}

func New() *Auth {
	var a Auth
	return &a
}

func (a *Auth) LoadCA(file string) (err error) {
	a.root = x509.NewCertPool()
	out, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}
	ok := a.root.AppendCertsFromPEM([]byte(out))
	if !ok {
		return fmt.Errorf("Can't load cert from %s", file)
	}
	return
}

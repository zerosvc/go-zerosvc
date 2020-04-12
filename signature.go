package zerosvc

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"golang.org/x/crypto/ed25519"
)

//Signature types

const (
	// raw 25519 public key
	SigTypeEd25519 = 1
	SigTypeX509    = 2
)

type Signer interface {
	Sign(data []byte) (signature []byte)
	// Verify verifies data with currently set up pubkey. Uninitialized Signer should panic
	Verify(data []byte, signature []byte) (ok bool)
	// PublicKey retrieves public key
	PublicKey() []byte
	// PrivateKey retrieves private key
	PrivateKey() []byte
	Type() uint8
}

// verifier is subset of signer that only does message verify
type Verifier interface {
	// Verify verifies data with currently set up pubkey. Uninitialized Verifier should panic
	Verify(data []byte, signature []byte) (ok bool)
	Type() uint8
	PublicKey() []byte
	// PrivateKey retrieves private key
}

type SigEd25519 struct {
	Pub  ed25519.PublicKey
	Priv ed25519.PrivateKey
}

// NewSignerEd25519 returns new signer; if key is not specified the new random one will be generated
func NewSignerEd25519(key ...ed25519.PrivateKey) (Signer, error) {
	var s SigEd25519
	if len(key) > 0 {
		if len(key[0]) != ed25519.PrivateKeySize {
			return nil, fmt.Errorf("wrong private key size [%d:%d]", len(key[0]), ed25519.PrivateKeySize)
		}
		s.Priv = key[0]
		s.Pub = key[0].Public().(ed25519.PublicKey)
	} else {
		var err error
		s.Pub, s.Priv, err = ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
	}
	return &s, nil
}

func SignerEd25519FromPub(pub ed25519.PublicKey) (Signer, error) {
	var s SigEd25519
	if len(pub) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("wrong private key size [%d:%d]", len(pub), ed25519.PrivateKeySize)
	}
	s.Pub = pub
	return &s, nil
}

func (s *SigEd25519) PublicKey() []byte {
	return s.Pub
}

func (s *SigEd25519) PrivateKey() []byte {
	return s.Priv
}

func (s *SigEd25519) Sign(data []byte) []byte {
	if bytes.Equal(s.Priv, ed25519.PrivateKey{}) {
		panic("tried to sign with no key")
	}
	return ed25519.Sign(s.Priv, data)
}

func (s *SigEd25519) Verify(data []byte, signature []byte) (ok bool) {
	if bytes.Equal(s.Pub, ed25519.PublicKey{}) {
		panic("tried to verify with no key")
	}
	return ed25519.Verify(s.Pub, data, signature)
}
func (s *SigEd25519) Type() uint8 {
	return SigTypeEd25519
}

package config

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
)

// convert public key pem file to rsa.PublicKey struct
func ParsePublicKeyPEM(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// remove pem headers, decodes base64 block and returns it
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New("invalid PEM")
	}

	// parse public key into one of (rsa.PublicKey, ecdsa.PublicKey etc)
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	// parse into rsa.PublicKey struct
	rsaKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not RSA public key")
	}
	return rsaKey, nil
}
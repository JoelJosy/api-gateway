package config

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// ParsePublicKeyPEM converts a PEM file into an RSA public key.
func ParsePublicKeyPEM(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ParsePublicKeyPEMData(data)
}

// LoadPublicKey loads the JWT verification key from Secrets Manager when enabled,
// otherwise it falls back to the local PEM file path in config.
func LoadPublicKey(cfg Config) (*rsa.PublicKey, error) {
	if os.Getenv("USE_SECRETS_MANAGER") == "true" {
		secretID := os.Getenv("PUBLIC_KEY_SECRET_ID")
		if secretID == "" {
			return nil, errors.New("PUBLIC_KEY_SECRET_ID is required when USE_SECRETS_MANAGER=true")
		}

		awsCfg, err := awsconfig.LoadDefaultConfig(context.Background())
		if err != nil {
			return nil, fmt.Errorf("load aws config: %w", err)
		}

		client := secretsmanager.NewFromConfig(awsCfg)
		secretValue, err := client.GetSecretValue(context.Background(), &secretsmanager.GetSecretValueInput{
			SecretId: &secretID,
		})
		if err != nil {
			return nil, fmt.Errorf("fetch public key from secrets manager: %w", err)
		}

		if secretValue.SecretString == nil {
			return nil, errors.New("public key secret has no string value")
		}

		return ParsePublicKeyPEMData([]byte(*secretValue.SecretString))
	}

	return ParsePublicKeyPEM(cfg.PubKeyPath)
}

// ParsePublicKeyPEMData converts PEM bytes into an RSA public key.
func ParsePublicKeyPEMData(data []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New("invalid PEM")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not RSA public key")
	}
	return rsaKey, nil
}
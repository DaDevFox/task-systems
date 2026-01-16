package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

const (
	// PrivateKeyPerms is 0o600 (owner read/write only)
	PrivateKeyPerms = 0o600
	// PublicKeyPerms is 0o644 (owner read/write, group/others read)
	PublicKeyPerms = 0o644
)

// GenerateRSAKeyPair generates a new RSA key pair with the specified bit size.
// Minimum recommended size is 2048 bits, 3072 or 4096 for production.
func GenerateRSAKeyPair(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate rsa key: %w", err)
	}
	return privateKey, &privateKey.PublicKey, nil
}

// SavePrivateKey saves an RSA private key to a PEM-encoded file.
// The file is written with secure permissions (0o600).
func SavePrivateKey(path string, key *rsa.PrivateKey) error {
	if key == nil {
		return fmt.Errorf("private key cannot be nil")
	}

	privBytes := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	pemData := pem.EncodeToMemory(block)

	err := os.WriteFile(path, pemData, PrivateKeyPerms)
	if err != nil {
		return fmt.Errorf("failed to write private key file: %w", err)
	}
	return nil
}

// SavePublicKey saves an RSA public key to a PEM-encoded file.
// The file is written with permissions 0o644.
func SavePublicKey(path string, key *rsa.PublicKey) error {
	if key == nil {
		return fmt.Errorf("public key cannot be nil")
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}
	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}
	pemData := pem.EncodeToMemory(block)

	err = os.WriteFile(path, pemData, PublicKeyPerms)
	if err != nil {
		return fmt.Errorf("failed to write public key file: %w", err)
	}
	return nil
}

// LoadRSAPrivateKey loads an RSA private key from a PEM-encoded file.
func LoadRSAPrivateKey(path string) (*rsa.PrivateKey, error) {
	if path == "" {
		return nil, fmt.Errorf("key path cannot be empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}
	return ParseRSAPrivateKeyPEM(data)
}

// ParseRSAPrivateKeyPEM parses an RSA private key from PEM-encoded data.
func ParseRSAPrivateKeyPEM(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode pem block")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return key, nil
	}
	pkcs8Key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key (pkcs1/pkcs8): %w", err)
	}
	rsaKey, ok := pkcs8Key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not an rsa private key")
	}
	return rsaKey, nil
}

// LoadRSAPublicKey loads an RSA public key from a PEM-encoded file.
func LoadRSAPublicKey(path string) (*rsa.PublicKey, error) {
	if path == "" {
		return nil, fmt.Errorf("key path cannot be empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}
	return ParseRSAPublicKeyPEM(data)
}

// ParseRSAPublicKeyPEM parses an RSA public key from PEM-encoded data.
func ParseRSAPublicKeyPEM(data []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode pem block")
	}
	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not an rsa public key")
	}
	return rsaKey, nil
}

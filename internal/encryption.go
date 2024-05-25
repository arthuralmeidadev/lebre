package internal

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

func GenerateRSAKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, &privateKey.PublicKey, nil
}

func RSAEncrypt(data []byte, publicKey *rsa.PublicKey) ([]byte, error) {
	ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, data)
	if err != nil {
		return nil, err
	}
	return ciphertext, nil
}

func RSADecrypt(ciphertext []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	data, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, ciphertext)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func PublicKeyToPEM(publicKey *rsa.PublicKey) ([]byte, error) {
	publicKeyASN1, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, err
	}

	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyASN1,
	}

	return pem.EncodeToMemory(pemBlock), nil
}

func PEMToPublicKey(pemData string) (*rsa.PublicKey, error) {
	pemBlock, _ := pem.Decode([]byte(pemData))
	if pemBlock == nil || pemBlock.Type != "PUBLIC KEY" {
		return nil, errors.New("failed to decode PEM block containing public key")
	}

	publicKey, err := x509.ParsePKIXPublicKey(pemBlock.Bytes)
	if err != nil {
		return nil, err
	}

	switch publicKey := publicKey.(type) {
	case *rsa.PublicKey:
		return publicKey, nil
	default:
		return nil, fmt.Errorf("not an RSA public key")
	}
}

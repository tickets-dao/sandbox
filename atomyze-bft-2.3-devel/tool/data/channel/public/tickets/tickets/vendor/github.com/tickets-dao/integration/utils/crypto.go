package utils

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"

	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"
)

// signMessage - sign arguments with private key in ed25519
func signMessage(privateKey ed25519.PrivateKey, publicKey ed25519.PublicKey, result []string) ([]byte, error) {
	message := sha3.Sum256([]byte(strings.Join(result, "")))
	sig := ed25519.Sign(privateKey, message[:])
	if !ed25519.Verify(publicKey, message[:], sig) {
		err := fmt.Errorf("valid signature rejected")
		return nil, err
	}
	return sig, nil
}

// Sign - sign arguments before send to hlf. create message with certain order arguments expected by chaincode validation in foundation library
func Sign(privateKey ed25519.PrivateKey, publicKey ed25519.PublicKey, channel string, chaincode string, methodName string, args []string) ([]string, error) {
	nonce := GetNonce()
	result := append(append([]string{methodName, "", chaincode, channel}, args...), nonce, ConvertPublicKeyToBase58(publicKey))

	sMsg, err := signMessage(privateKey, publicKey, result)
	if err != nil {
		return nil, fmt.Errorf("sign message: %w", err)
	}

	return append(result[1:], base58.Encode(sMsg)), nil
}

// GeneratePrivateAndPublicKey - create new private and public key
func GeneratePrivateAndPublicKey() (ed25519.PrivateKey, ed25519.PublicKey, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	return privateKey, publicKey, err
}

// GetAddressByPublicKey - get address by encoded string in standard encoded for project is 'base58.Check'
func GetAddressByPublicKey(publicKey ed25519.PublicKey) (string, error) {
	if len(publicKey) == 0 {
		return "", errors.New("publicKey can't be empty")
	}

	hash := sha3.Sum256(publicKey)
	return base58.CheckEncode(hash[1:], hash[0]), nil
}

// GetPrivateKeyFromBase58Check - get private key type Ed25519 by string - Base58Check encoded private key
func GetPrivateKeyFromBase58Check(secretKey string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	decode, ver, err := base58.CheckDecode(secretKey)
	if err != nil {
		return nil, nil, fmt.Errorf("check decode: %w", err)
	}
	privateKey := ed25519.PrivateKey(append([]byte{ver}, decode...))
	publicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return nil, nil, errors.New("type assertion failed")
	}
	return privateKey, publicKey, nil
}

// ConvertPublicKeyToBase58 - use publicKey with standard encoded type - Base58
func ConvertPublicKeyToBase58(publicKey ed25519.PublicKey) string {
	return base58.Encode(publicKey)
}

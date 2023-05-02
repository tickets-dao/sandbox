package main

import (
	"encoding/pem"
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	issuerPrKey, err := readPrivateKeyFromFile("../keys/issuer.pem")
	if err != nil {
		log.Fatalf("failed to read private key from pem file: %v", err)
	}

	issuerPKey := issuerPrKey.Public().(ed25519.PublicKey)

	signedEmitArgs, err := Sign(issuerPrKey, issuerPKey, "tickets", "tickets", "initialize", nil)
	if err != nil {
		log.Fatalf("failed to sign arguments: %v", err)
		return
	}

	fmt.Println(signedEmitArgs)
}

// Чтение приватного ключа из файла
func readPrivateKeyFromFile(filename string) (ed25519.PrivateKey, error) {
	fileBytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(fileBytes)
	if block == nil || block.Type != "ED25519 PRIVATE KEY" {
		return nil, err
	}

	privateKey := ed25519.NewKeyFromSeed(block.Bytes)
	return privateKey, nil
}

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
	nonce := strconv.FormatInt(time.Now().UnixMilli(), 10)
	result := append(append([]string{methodName, "", chaincode, channel}, args...), nonce, ConvertPublicKeyToBase58(publicKey))

	sMsg, err := signMessage(privateKey, publicKey, result)
	if err != nil {
		return nil, fmt.Errorf("sign message: %w", err)
	}

	return append(result[1:], base58.Encode(sMsg)), nil
}

// ConvertPublicKeyToBase58 - use publicKey with standard encoded type - Base58
func ConvertPublicKeyToBase58(publicKey ed25519.PublicKey) string {
	return base58.Encode(publicKey)
}

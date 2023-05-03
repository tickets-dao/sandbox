package integration

import (
	"encoding/pem"
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	"github.com/tickets-dao/integration/utils"
	"golang.org/x/crypto/ed25519"
	"os"
	"testing"
)

// TestTransfer - create user 'from' and user 'userTo', emit amount to user 'userFrom' and transfer token from 'userFrom' to 'userTo'
func TestGenerateKeys(t *testing.T) {
	err := generateAndSaveKeys("issuer")
	if err != nil {
		t.Fatalf("failed to generate and save keys for issuer: %v", err)
	}

	err = generateAndSaveKeys("user")
	if err != nil {
		t.Fatalf("failed to generate and save keys for user: %v", err)
	}
}

func TestPrintIssuer(t *testing.T) {
	prKey, err := readPrivateKeyFromFile(filenameByUser(issuerUsername))
	if err != nil {
		t.Fatalf("failed to read private key from file: %v", err)
	}

	fmt.Println(utils.GetAddressByPublicKey(prKey.Public().(ed25519.PublicKey)))
}

// Сохранение приватного ключа в файл
func savePrivateKeyToFile(privateKey ed25519.PrivateKey, filename string) error {
	keyBytes := privateKey.Seed()

	block := &pem.Block{
		Type:  "ED25519 PRIVATE KEY",
		Bytes: keyBytes,
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	err = pem.Encode(file, block)
	if err != nil {
		return err
	}

	return nil
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

func generateAndSaveKeys(username string) error {
	privateKey, _, err := utils.GeneratePrivateAndPublicKey()
	if err != nil {
		return fmt.Errorf("failed to generate keys: %v", err)
	}

	return savePrivateKeyToFile(privateKey, "./keys/"+username+".pem")
}

func readKeysFromFile(username string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	filename := "./keys/" + username + ".private"
	secretBytes, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("faile to open privae key filename at '%s': %v", filename, err)
	}

	return utils.GetPrivateKeyFromBase58Check(base58.CheckEncode(secretBytes[1:], secretBytes[0]))
}

func filenameByUser(username string) string {
	return fmt.Sprintf("./keys/%s.pem", username)
}

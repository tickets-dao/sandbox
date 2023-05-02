package integration

import (
	"context"
	"fmt"
	"golang.org/x/crypto/ed25519"
	"os"
	"testing"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/runner"
	"github.com/tickets-dao/integration/utils"
)

func TestAclAddUser(t *testing.T) {
	runner.Run(t, "add user in acl chaincode", func(t provider.T) {
		ctx := context.Background()
		t.Severity(allure.BLOCKER)
		t.Description("As member of organization add user in acl chaincode and validate that user was added")
		t.Tags("smoke", "acl", "positive")

		var publicKey string

		t.WithNewStep("Generate cryptos for issuer", func(sCtx provider.StepCtx) {
			pkey, err := readPrivateKeyFromFile(filenameByUser(issuerUsername))
			sCtx.Assert().NoError(err)
			publicKey = base58.Encode(pkey.Public().(ed25519.PublicKey))
		})

		t.WithNewStep("Add user by invoking method `addUser` of chaincode `acl` with valid parameters", func(sCtx provider.StepCtx) {
			_, err := utils.Invoke(ctx, os.Getenv(utils.EnvHlfProxyURL),
				os.Getenv(utils.EnvHlfProxyAuthToken),
				"acl", "addUser", publicKey, "test", "issuer", "true")
			sCtx.Assert().NoError(err)
		})

		time.Sleep(utils.BatchTransactionTimeout)
		t.WithNewStep("Check user is created by querying method `checkKeys` of chaincode `acl`", func(sCtx provider.StepCtx) {
			_, err := utils.Query(ctx, os.Getenv(utils.EnvHlfProxyURL),
				os.Getenv(utils.EnvHlfProxyAuthToken), "acl", "checkKeys", publicKey)
			sCtx.Assert().NoError(err)
		})

		t.WithNewStep("Generate cryptos for user", func(sCtx provider.StepCtx) {

			pkey, err := readPrivateKeyFromFile(filenameByUser(defaultUsername))
			sCtx.Assert().NoError(err)
			publicKey = base58.Encode(pkey.Public().(ed25519.PublicKey))

		})

		t.WithNewStep("Add user by invoking method `addUser` of chaincode `acl` with valid parameters", func(sCtx provider.StepCtx) {
			_, err := utils.Invoke(ctx, os.Getenv(utils.EnvHlfProxyURL),
				os.Getenv(utils.EnvHlfProxyAuthToken),
				"acl", "addUser", publicKey, "test", "defaultUser", "true")
			sCtx.Assert().NoError(err)
		})

		time.Sleep(utils.BatchTransactionTimeout)
		t.WithNewStep("Check user is created by querying method `checkKeys` of chaincode `acl`", func(sCtx provider.StepCtx) {
			_, err := utils.Query(ctx, os.Getenv(utils.EnvHlfProxyURL),
				os.Getenv(utils.EnvHlfProxyAuthToken), "acl", "checkKeys", publicKey)
			sCtx.Assert().NoError(err)
		})

	})
}

func TestPrintIssuerAddress(t *testing.T) {
	privateKey, _, err := readKeysFromFile("issuer")
	if err != nil {
		t.Fatalf("failed to read private key: %v", err)
	}

	publicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok {
		t.Fatalf("failed to get public key")
	}

	fmt.Println(utils.GetAddressByPublicKey(publicKey))
}

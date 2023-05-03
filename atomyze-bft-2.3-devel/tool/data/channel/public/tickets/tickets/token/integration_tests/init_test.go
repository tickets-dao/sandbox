package integration

import (
	"context"
	"fmt"
	"github.com/btcsuite/btcutil/base58"
	"os"
	"testing"
	"time"

	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/runner"
	"github.com/tickets-dao/integration/utils"
	"golang.org/x/crypto/ed25519"
)

const issuerUsername = "issuer"
const defaultUsername = "user"

// TestTransfer - create user 'from' and user 'userTo', emit amount to user 'userFrom' and transfer token from 'userFrom' to 'userTo'
func TestInitialize(t *testing.T) {
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

	runner.Run(t, "Emission of `tickets` token", func(t provider.T) {
		t.Severity(allure.BLOCKER)
		t.Description("Testing emitting token, and transferring it from one to another user")
		t.Tags("positive", "transfer")

		var (
			ctx = context.Background()

			issuerPrKey   ed25519.PrivateKey
			issuerPKey    ed25519.PublicKey
			issuerAddress string
		)

		t.WithNewStep("Generate cryptos for users and saving it to `acl` chaincode", func(sCtx provider.StepCtx) {
			sCtx.WithNewStep("Get crypto for issuer from env and saving to `acl` chaincode", func(sCtx provider.StepCtx) {
				var err error

				issuerPrKey, err = readPrivateKeyFromFile(filenameByUser(issuerUsername))
				sCtx.Assert().NoError(err)

				issuerPKey = issuerPrKey.Public().(ed25519.PublicKey)
				issuerAddress, err = utils.GetAddressByPublicKey(issuerPKey)
				sCtx.Assert().NoError(err)
			})

			t.WithNewStep("Check user is created by querying method `checkKeys` of chaincode `acl`", func(sCtx provider.StepCtx) {
				aclResp, err := utils.Query(ctx, os.Getenv(utils.EnvHlfProxyURL),
					os.Getenv(utils.EnvHlfProxyAuthToken), "acl", "checkKeys", base58.Encode(issuerPKey))
				sCtx.Assert().NoError(err)

				if aclResp != nil {
					fmt.Println("acl resp: ", string(aclResp.Payload))
				}
			})
		})

		t.WithNewStep("Emit FIAT token to first user", func(sCtx provider.StepCtx) {
			var (
				signedEmitArgs []string
				err            error
			)
			sCtx.WithNewStep("Sign arguments before emission process", func(sCtx provider.StepCtx) {
				signedEmitArgs, err = utils.Sign(issuerPrKey, issuerPKey, "tickets", "tickets", "initialize", nil)
				sCtx.Assert().NoError(err)
			})

			sCtx.WithNewStep("Invoke fiat chaincode by issuer for token emission", func(sCtx provider.StepCtx) {
				resp, err := utils.Invoke(ctx, os.Getenv(utils.EnvHlfProxyURL),
					os.Getenv(utils.EnvHlfProxyAuthToken),
					"tickets", "initialize", signedEmitArgs...)
				sCtx.Assert().NoError(err)
				if resp != nil {
					fmt.Printf("got response from init: '%s'\n", string(resp.Payload))
				}
			})

			time.Sleep(utils.BatchTransactionTimeout)
			sCtx.WithNewStep("Check balance of first user after emission", func(sCtx provider.StepCtx) {
				resp, err := utils.Query(ctx, os.Getenv(utils.EnvHlfProxyURL),
					os.Getenv(utils.EnvHlfProxyAuthToken),
					"tickets", "industrialBalanceOf", issuerAddress)
				sCtx.Assert().NoError(err)
				if resp != nil {
					fmt.Printf("got issuer balance: %s\n", string(resp.Payload))
				}

				//sCtx.Assert().Equal("\"1\"", string(resp.Payload))
			})
		})
	})
}

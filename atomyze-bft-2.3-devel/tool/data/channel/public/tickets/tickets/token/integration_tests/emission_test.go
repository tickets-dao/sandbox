package integration

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/runner"
	"github.com/tickets-dao/integration/utils"
	"golang.org/x/crypto/ed25519"
)

//go:embed testdata/default_price_categories.json
var defaultPriceCategories []byte

func TestCreateEvent(t *testing.T) {
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

		t.WithNewStep("read keys from file", func(sCtx provider.StepCtx) {
			sCtx.WithNewStep("Get crypto for issuer", func(sCtx provider.StepCtx) {
				var err error

				issuerPrKey, err = readPrivateKeyFromFile(filenameByUser(issuerUsername))
				sCtx.Assert().NoError(err)

				issuerPKey = issuerPrKey.Public().(ed25519.PublicKey)
				issuerAddress, err = utils.GetAddressByPublicKey(issuerPKey)
				sCtx.Assert().NoError(err)
			})
		})

		t.WithNewStep("Create new event", func(sCtx provider.StepCtx) {
			var (
				signedEmitArgs []string
				err            error
			)
			sCtx.WithNewStep("Sign arguments before emission process", func(sCtx provider.StepCtx) {
				signedEmitArgs, err = utils.Sign(
					issuerPrKey,
					issuerPKey,
					"tickets",
					"tickets",
					"emission",
					[]string{string(defaultPriceCategories), "Лебединое озеро", "Москва, Театральная площадь 1", "2023-05-26 15:00:00"},
				)
				sCtx.Assert().NoError(err)
			})

			sCtx.WithNewStep("Invoke tickets chaincode by issuer for setting issuer info", func(sCtx provider.StepCtx) {
				resp, err := utils.Invoke(ctx, os.Getenv(utils.EnvHlfProxyURL),
					os.Getenv(utils.EnvHlfProxyAuthToken),
					"tickets", "emission", signedEmitArgs...)
				sCtx.Assert().NoError(err)
				if resp != nil {
					fmt.Printf("got response from init: '%s'\n", string(resp.Payload))
				}
			})

			time.Sleep(utils.BatchTransactionTimeout)
			sCtx.WithNewStep("Check issuer info", func(sCtx provider.StepCtx) {
				resp, err := utils.Query(ctx, os.Getenv(utils.EnvHlfProxyURL),
					os.Getenv(utils.EnvHlfProxyAuthToken),
					"tickets", "issuerInfo", issuerAddress)
				sCtx.Assert().NoError(err)
				if resp != nil {
					fmt.Printf("got issuer info: %s\n", string(resp.Payload))
				}

				//sCtx.Assert().Equal("\"1\"", string(resp.Payload))
			})
		})
	})
}

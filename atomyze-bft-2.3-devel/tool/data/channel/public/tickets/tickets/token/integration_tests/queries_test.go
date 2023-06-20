package integration

import (
	"context"
	"fmt"
	"github.com/ozontech/allure-go/pkg/allure"
	"github.com/ozontech/allure-go/pkg/framework/provider"
	"github.com/ozontech/allure-go/pkg/framework/runner"
	"github.com/tickets-dao/integration/utils"
	"golang.org/x/crypto/ed25519"
	"os"
	"testing"
)

func TestMyTickets(t *testing.T) {
	runner.Run(t, "add user in acl chaincode", func(t provider.T) {
		ctx := context.Background()
		t.Severity(allure.BLOCKER)
		t.Description("As member of organization add user in acl chaincode and validate that user was added")
		t.Tags("smoke", "acl", "positive")

		var pkey ed25519.PrivateKey
		var pubKey ed25519.PublicKey
		var err error

		t.WithNewStep("Generate cryptos for issuer", func(sCtx provider.StepCtx) {
			pkey, err = readPrivateKeyFromFile(filenameByUser(defaultUsername))
			sCtx.Assert().NoError(err)
			pubKey = pkey.Public().(ed25519.PublicKey)
			//issuerAddress, err = utils.GetAddressByPublicKey(pubKey)
			sCtx.Assert().NoError(err)
		})

		t.WithNewStep("Add user by invoking method `addUser` of chaincode `acl` with valid parameters", func(sCtx provider.StepCtx) {
			var (
				signedEmitArgs []string
			)
			sCtx.WithNewStep("Sign arguments before emission process", func(sCtx provider.StepCtx) {
				signedEmitArgs, err = utils.Sign(pkey, pubKey, "tickets", "tickets", "myTickets", nil)
				sCtx.Assert().NoError(err)
			})

			resp, err := utils.Query(ctx, os.Getenv(utils.EnvHlfProxyURL),
				os.Getenv(utils.EnvHlfProxyAuthToken),
				"tickets", "myTickets", signedEmitArgs...)
			sCtx.Assert().NoError(err)
			if err == nil {
				fmt.Println(string(resp.Payload))
			}
		})
	})
}

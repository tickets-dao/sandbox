package token

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tickets-dao/foundation/v3/core"
	ma "github.com/tickets-dao/foundation/v3/mock"
	"github.com/tickets-dao/foundation/v3/proto"
)

func TestIndustrialToken_TxIndustrialBuyBack(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()
	user := mock.NewWallet()

	it := &IT{
		IndustrialToken{
			Name:            "Industrial Token",
			Symbol:          "IT",
			Decimals:        8,
			UnderlyingAsset: "Palladium",
			config:          &proto.Industrial{},
		},
	}

	mock.NewChainCode("it", it, &core.ContractOptions{}, issuer.Address())

	issuer.OtfNbInvoke("it", "initialize")

	issuer.SignedInvoke("it", "setRate", "buyBack", "CC", "200000000")
	issuer.AddAllowedBalance("it", "CC", 200)

	issuer.SignedInvoke("it", "transferIndustrial", user.Address(), "203010", "100", "")

	if err := user.RawSignedInvokeWithErrorReturned("it", "industrialBuyBack", "203010", "0", "CC"); err != nil {
		assert.Equal(t, "amount should be more than zero", err.Error())
	}

	err := user.RawSignedInvokeWithErrorReturned("it", "industrialBuyBack", "203010", "100", "CC")
	assert.NoError(t, err)
}

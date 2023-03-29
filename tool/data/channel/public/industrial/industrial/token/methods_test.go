package token

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tickets-dao/foundation/v3/core"
	ma "github.com/tickets-dao/foundation/v3/mock"
)

func TestIndustrialToken_SetLimits(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()

	it := &IT{
		IndustrialToken{
			Name:            "Industrial Token",
			Symbol:          "IT",
			Decimals:        8,
			UnderlyingAsset: "Palladium",
		},
	}

	mock.NewChainCode("it", it, &core.ContractOptions{}, issuer.Address())

	issuer.OtfNbInvoke("it", "initialize")

	issuer.SignedInvoke("it", "setRate", "distribute", "", "1")

	if err := issuer.RawSignedInvokeWithErrorReturned("it", "setLimits", "makarone", "", "1", "3"); err != nil {
		assert.Equal(t, "unknown DealType. Rate for deal type makarone and currency  was not set", err.Error())
	}

	if err := issuer.RawSignedInvokeWithErrorReturned("it", "setLimits", "distribute", "fish", "1", "3"); err != nil {
		assert.Equal(t, "unknown currency. Rate for deal type distribute and currency fish was not set", err.Error())
	}

	if err := issuer.RawSignedInvokeWithErrorReturned("it", "setLimits", "distribute", "", "10", "3"); err != nil {
		assert.Equal(t, "min limit is greater than max limit", err.Error())
	}

	err := issuer.RawSignedInvokeWithErrorReturned("it", "setLimits", "distribute", "", "1", "0")
	assert.NoError(t, err)
}

func TestIndustrialToken_SetRate(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()
	outsider := mock.NewWallet()

	it := &IT{
		IndustrialToken{
			Name:            "Industrial Token",
			Symbol:          "IT",
			Decimals:        8,
			UnderlyingAsset: "Palladium",
		},
	}

	mock.NewChainCode("it", it, &core.ContractOptions{}, issuer.Address())

	issuer.OtfNbInvoke("it", "initialize")

	if err := outsider.RawSignedInvokeWithErrorReturned("it", "setRate", "distribute", "", "1"); err != nil {
		assert.Equal(t, "unauthorized", err.Error())
	}
	if err := issuer.RawSignedInvokeWithErrorReturned("it", "setRate", "distribute", "", "0"); err != nil {
		assert.Equal(t, "trying to set rate = 0", err.Error())
	}
	if err := issuer.RawSignedInvokeWithErrorReturned("it", "setRate", "distribute", "IT", "3"); err != nil {
		assert.Equal(t, "currency is equals token: it is impossible", err.Error())
	}
	err := issuer.RawSignedInvokeWithErrorReturned("it", "setRate", "distribute", "", "1")
	assert.NoError(t, err)

	rawMD := issuer.Invoke("it", "metadata")
	md := &metadata{}

	assert.NoError(t, json.Unmarshal([]byte(rawMD), md))

	rates := md.Rates
	assert.Len(t, rates, 1)

	issuer.SignedInvoke("it", "deleteRate", "distribute", "")

	rawMD = issuer.Invoke("it", "metadata")
	md = &metadata{}

	assert.NoError(t, json.Unmarshal([]byte(rawMD), md))

	rates = md.Rates
	assert.Len(t, rates, 0)
}

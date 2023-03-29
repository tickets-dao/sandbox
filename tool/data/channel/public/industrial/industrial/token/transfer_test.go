package token

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tickets-dao/foundation/v3/core"
	ma "github.com/tickets-dao/foundation/v3/mock"
)

func Test_Transfer_WithFee_BuyBack(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()
	feeAddressSetter := mock.NewWallet()
	feeSetter := mock.NewWallet()
	feeAggregator := mock.NewWallet()
	user := mock.NewWallet()

	it := &IT{
		IndustrialToken{
			Name:            "Industrial Token",
			Symbol:          "GF27ILN061",
			Decimals:        8,
			UnderlyingAsset: "Cobalt",
		},
	}

	mock.NewChainCode("it", it, &core.ContractOptions{}, issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())

	issuer.OtfNbInvoke("it", "initialize")

	feeSetter.SignedInvoke("it", "setFee", "GF27ILN061", "500000", "1", "0")

	predict := Predict{}
	rawResp := issuer.Invoke("it", "predictFee", "100")

	err := json.Unmarshal([]byte(rawResp), &predict)
	assert.NoError(t, err)

	fmt.Println("Invoke response: ", predict.Fee)

	err = issuer.RawSignedInvokeWithErrorReturned("it", "transferIndustrial", user.Address(), "203012", "100", "")
	assert.EqualError(t, err, "fee address is not set")

	feeAddressSetter.SignedInvoke("it", "setFeeAddress", feeAggregator.Address())
	issuer.SignedInvoke("it", "transferIndustrial", user.Address(), "203012", "100", "")

	issuer.IndustrialBalanceShouldBe("it", "203012", 49999999999899)
	user.IndustrialBalanceShouldBe("it", "203012", 100)
	feeAggregator.IndustrialBalanceShouldBe("it", "203012", 1)

	issuer.SignedInvoke("it", "setRate", "buyBack", "CC", "200000000")
	issuer.AddAllowedBalance("it", "CC", 200)

	user.SignedInvoke("it", "industrialBuyBack", "203012", "100", "CC")
	issuer.IndustrialBalanceShouldBe("it", "203012", 49999999999999)
	issuer.AllowedBalanceShouldBe("it", "CC", 0)
	user.AllowedBalanceShouldBe("it", "CC", 200)
}

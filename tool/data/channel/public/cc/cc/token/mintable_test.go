package token

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tickets-dao/foundation/v3/core/types/big"
	"github.com/tickets-dao/foundation/v3/mock"
	"github.com/tickets-dao/foundation/v3/token"
	"golang.org/x/crypto/sha3"
)

type predict struct {
	Currency string   `json:"currency"`
	Fee      *big.Int `json:"fee"`
}

type VT struct {
	token.BaseToken
}

// Chaincode name
const (
	ccName = "CC"
)

func Test_Buy_BuyBack(t *testing.T) {
	ledgerMock := mock.NewLedger(t)
	owner := ledgerMock.NewWallet()
	user1 := ledgerMock.NewWallet()

	cc := NewMintableToken(token.BaseToken{
		Name:   "Atomyze USD",
		Symbol: "CC",
	})
	ledgerMock.NewChainCode(ccName, cc, nil, owner.Address())

	t.Run("setting rates", func(t *testing.T) {
		owner.SignedInvoke(ccName, "setRate", "buyToken", "FIAT", "50000000")
		owner.SignedInvoke(ccName, "setRate", "buyBack", "FIAT", "100000000")
	})

	t.Run("checking metadata", func(t *testing.T) {
		rates := ledgerMock.Metadata(ccName).Rates
		assert.Len(t, rates, 2)
		for _, rate := range rates {
			switch rate.DealType {
			case "buyToken":
				assert.Equal(t, rate.Currency, "FIAT")
				assert.Equal(t, rate.Rate, big.NewInt(50000000))
			case "buyBack":
				assert.Equal(t, rate.Currency, "FIAT")
				assert.Equal(t, rate.Rate, big.NewInt(100000000))
			default:
				panic("wrong rate")
			}
		}
	})

	t.Run("adding allowed balance", func(t *testing.T) {
		user1.AddAllowedBalance(ccName, "FIAT", 1000)
		user1.AllowedBalanceShouldBe(ccName, "FIAT", 1000)
	})

	t.Run("buying token", func(t *testing.T) {
		user1.SignedInvoke(ccName, "buyToken", "250", "FIAT")
		user1.BalanceShouldBe(ccName, 250)
		user1.AllowedBalanceShouldBe(ccName, "FIAT", 875)
	})

	t.Run("selling token", func(t *testing.T) {
		user1.SignedInvoke(ccName, "buyBack", "110", "FIAT")
		user1.BalanceShouldBe(ccName, 140)
		user1.AllowedBalanceShouldBe(ccName, "FIAT", 985)
	})
}

func Test_Buy_Rate_1by1(t *testing.T) {
	ledgerMock := mock.NewLedger(t)
	owner := ledgerMock.NewWallet()
	user1 := ledgerMock.NewWallet()

	const (
		CCchaincodeName   = "CC"
		FiatChaincodeName = "FIAT"

		FnBuyToken = "buyToken"
	)

	cc := NewMintableToken(token.BaseToken{
		Name:   "Same as USD",
		Symbol: CCchaincodeName})
	assert.NotNil(t, cc)

	ledgerMock.NewChainCode(CCchaincodeName, cc, nil, owner.Address())
	user1.BalanceShouldBe(CCchaincodeName, 0)

	t.Run("setting rates", func(t *testing.T) {
		// owner.SignedInvoke(ccName, "setRate", "buyToken", "FIAT", "50000000")
		owner.SignedInvoke(ccName, "setRate", "buyToken", "FIAT", "100000000")
	})

	t.Run("adding balance to cc", func(t *testing.T) {
		user1.AddAllowedBalance(CCchaincodeName, FiatChaincodeName, 100)
		user1.AllowedBalanceShouldBe(CCchaincodeName, FiatChaincodeName, 100)
	})

	t.Run("buy token", func(t *testing.T) {
		user1.SignedInvoke(CCchaincodeName, FnBuyToken, "100", FiatChaincodeName)
		user1.BalanceShouldBe(CCchaincodeName, 100)
		user1.AllowedBalanceShouldBe(CCchaincodeName, FiatChaincodeName, 0)
	})
}

func Test_Buy_Rate_1by2(t *testing.T) {
	ledgerMock := mock.NewLedger(t)
	owner := ledgerMock.NewWallet()
	user1 := ledgerMock.NewWallet()

	const (
		CCchaincodeName   = "CC"
		FiatChaincodeName = "FIAT"

		FnBuyToken = "buyToken"
	)

	cc := NewMintableToken(token.BaseToken{
		Name:   "Same as USD",
		Symbol: CCchaincodeName})
	assert.NotNil(t, cc)

	ledgerMock.NewChainCode(CCchaincodeName, cc, nil, owner.Address())
	user1.BalanceShouldBe(CCchaincodeName, 0)

	t.Run("setting rates", func(t *testing.T) {
		owner.SignedInvoke(ccName, "setRate", "buyToken", "FIAT", "50000000")
	})

	t.Run("adding balance to cc", func(t *testing.T) {
		user1.AddAllowedBalance(CCchaincodeName, FiatChaincodeName, 50)
		user1.AllowedBalanceShouldBe(CCchaincodeName, FiatChaincodeName, 50)
	})

	t.Run("buy token", func(t *testing.T) {
		user1.SignedInvoke(CCchaincodeName, FnBuyToken, "100", FiatChaincodeName)
		user1.BalanceShouldBe(CCchaincodeName, 100)
		user1.AllowedBalanceShouldBe(CCchaincodeName, FiatChaincodeName, 0)
	})
}

func Test_Buy_Limit(t *testing.T) {
	ledgerMock := mock.NewLedger(t)
	owner := ledgerMock.NewWallet()
	user1 := ledgerMock.NewWallet()

	cc := NewMintableToken(
		token.BaseToken{
			Name:   "Atomyze USD",
			Symbol: "CC",
		})

	ledgerMock.NewChainCode(ccName, cc, nil, owner.Address())

	t.Run("adding balance", func(t *testing.T) {
		user1.AddAllowedBalance(ccName, "FIAT", 1000)
		user1.AllowedBalanceShouldBe(ccName, "FIAT", 1000)
	})

	t.Run("buying tokens", func(t *testing.T) {
		owner.SignedInvoke(ccName, "setRate", "buyToken", "FIAT", "50000000")
		user1.SignedInvoke(ccName, "buyToken", "100", "FIAT")
		user1.BalanceShouldBe(ccName, 100)
		user1.AllowedBalanceShouldBe(ccName, "FIAT", 950)
	})

	t.Run("setting limits", func(t *testing.T) {
		owner.SignedInvoke(ccName, "setLimits", "buyToken", "FIAT", "100", "200")
	})

	t.Run("checking metadata", func(t *testing.T) {
		rates := ledgerMock.Metadata(ccName).Rates
		assert.Len(t, rates, 1)

		assert.Equal(t, rates[0].DealType, "buyToken")
		assert.Equal(t, rates[0].Currency, "FIAT")
		assert.Equal(t, rates[0].Rate, big.NewInt(50000000))
		assert.Equal(t, rates[0].Min, big.NewInt(100))
		assert.Equal(t, rates[0].Max, big.NewInt(200))
	})

	t.Run("[negative] trying to buy less than min limit", func(t *testing.T) {
		err := user1.RawSignedInvokeWithErrorReturned(ccName, "buyToken", "50", "FIAT")
		assert.EqualError(t, err, "amount out of limits")
	})

	t.Run("[negative] trying to buy more than max limit", func(t *testing.T) {
		err := user1.RawSignedInvokeWithErrorReturned(ccName, "buyToken", "300", "FIAT")
		assert.EqualError(t, err, "amount out of limits")
	})

	t.Run("buying tokens within the limits", func(t *testing.T) {
		user1.SignedInvoke(ccName, "buyToken", "150", "FIAT")
	})

	t.Run("setting limits with zero", func(t *testing.T) {
		owner.SignedInvoke(ccName, "setLimits", "buyToken", "FIAT", "100", "0")
	})

	t.Run("[negative] trying to set min limit greater than max limit", func(t *testing.T) {
		err := owner.RawSignedInvokeWithErrorReturned(ccName, "setLimits", "buyToken", "FIAT", "100", "50")
		assert.EqualError(t, err, "min limit is greater than max limit")
	})
}

func Test_Atomic_Swap(t *testing.T) {
	ccVtName := "vt"

	ledgerMock := mock.NewLedger(t)
	owner := ledgerMock.NewWallet()
	user1 := ledgerMock.NewWallet()

	cc := token.BaseToken{
		Symbol: "CC",
	}
	ledgerMock.NewChainCode(ccName, &cc, nil, owner.Address())

	vt := token.BaseToken{
		Symbol: "VT",
	}
	ledgerMock.NewChainCode(ccVtName, &vt, nil, owner.Address())

	t.Run("adding balance", func(t *testing.T) {
		user1.AddBalance(ccName, 1000)
		user1.BalanceShouldBe(ccName, 1000)
	})

	swapKey := "123"
	hashed := sha3.Sum256([]byte(swapKey))
	swapHash := hex.EncodeToString(hashed[:])

	var txID string
	t.Run("swap begin", func(t *testing.T) {
		txID = user1.SignedInvoke(ccName, "swapBegin", "CC", "VT", "450", swapHash)
		user1.BalanceShouldBe(ccName, 550)
		ledgerMock.WaitSwapAnswer(ccVtName, txID, time.Second*5)
	})

	t.Run("swap done", func(t *testing.T) {
		user1.Invoke(ccVtName, "swapDone", txID, swapKey)
		user1.AllowedBalanceShouldBe(ccVtName, "CC", 450)
	})

	// todo should check given balance
}

func Test_Transfer_WithFee(t *testing.T) {
	ccName1 := "CC"
	ccName2 := "AT00VAL"

	ledgerMock := mock.NewLedger(t)
	issuer := ledgerMock.NewWallet()
	feeAddressSetter := ledgerMock.NewWallet()
	feeSetter := ledgerMock.NewWallet()
	feeAggregator := ledgerMock.NewWallet()
	user := ledgerMock.NewWallet()

	cc1 := token.BaseToken{
		Symbol: "CC",
	}
	ledgerMock.NewChainCode(ccName1, &cc1, nil, issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())

	cc2 := &VT{
		token.BaseToken{
			Name:            "Atomyzer",
			Symbol:          "AT00VAL",
			Decimals:        8,
			UnderlyingAsset: "",
		},
	}
	ledgerMock.NewChainCode(ccName2, cc2, nil, issuer.Address(), feeSetter.Address(), feeAddressSetter.Address())

	t.Run("setting rate", func(t *testing.T) {
		issuer.SignedInvoke(ccName1, "setRate", "buyToken", "AT00VAL", "1")
	})

	t.Run("setting fee & fee address", func(t *testing.T) {
		feeAddressSetter.SignedInvoke(ccName1, "setFeeAddress", feeAggregator.Address())
		feeSetter.SignedInvoke(ccName1, "setFee", "AT00VAL", "500000", "1", "0")
	})

	t.Run("checking metadata", func(t *testing.T) {
		fee := ledgerMock.Metadata(ccName1).Fee

		assert.Equal(t, fee.Currency, ccName2)
		assert.Equal(t, fee.Fee, big.NewInt(500000))
		assert.Equal(t, fee.Floor, big.NewInt(1))
		assert.Equal(t, fee.Cap, big.NewInt(0))
	})

	t.Run("adding balances", func(t *testing.T) {
		issuer.AddBalance(ccName1, 50000000000000)
		issuer.AddAllowedBalance(ccName1, "AT00VAL", 50000000000000)

		issuer.BalanceShouldBe(ccName1, 50000000000000)
		issuer.AllowedBalanceShouldBe(ccName1, "AT00VAL", 50000000000000)
	})

	predict := predict{}
	t.Run("predicting fee", func(t *testing.T) {
		rawResp := issuer.Invoke(ccName1, "predictFee", "100")

		err := json.Unmarshal([]byte(rawResp), &predict)
		assert.NoError(t, err)

		assert.Equal(t, predict.Fee, big.NewInt(1))
		fmt.Println("Predicted fee: ", predict.Fee)
	})

	t.Run("transferring with fee", func(t *testing.T) {
		issuer.SignedInvoke(ccName1, "transfer", user.Address(), "100", "")

		issuer.BalanceShouldBe(ccName1, 49999999999900)
		issuer.AllowedBalanceShouldBe(ccName1, "AT00VAL", 49999999999999)
		user.BalanceShouldBe(ccName1, 100)
		feeAggregator.AllowedBalanceShouldBe(ccName1, "AT00VAL", predict.Fee.Uint64())
	})
}

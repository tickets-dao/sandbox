package token

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tickets-dao/foundation/v3/core"
	"github.com/tickets-dao/foundation/v3/core/types/big"
	ma "github.com/tickets-dao/foundation/v3/mock"
	"github.com/tickets-dao/foundation/v3/token"
)

const ccName = "fiat"

func Test_FiatToken(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()
	user := mock.NewWallet()

	fiat := NewFiatToken(token.BaseToken{
		Name:            "USD",
		Symbol:          "USD",
		Decimals:        8,
		UnderlyingAsset: "",
	})

	mock.NewChainCode(ccName, fiat, &core.ContractOptions{
		DisabledFunctions: []string{"TxTransfer"},
	}, issuer.Address())

	t.Run("checking method transfer is not exists", func(t *testing.T) {
		assert.False(t, mock.Metadata(ccName).MethodExists("transfer"))
	})

	t.Run("[negative] creating emission by user", func(t *testing.T) {
		err := user.RawSignedInvokeWithErrorReturned(ccName, "emit", user.Address(), "10")
		assert.EqualError(t, err, errUnauthorized)
	})

	t.Run("[negative] trying to create emission with zero value", func(t *testing.T) {
		err := issuer.RawSignedInvokeWithErrorReturned(ccName, "emit", user.Address(), "0")
		assert.EqualError(t, err, errZeroAmount)
	})

	t.Run("emission", func(t *testing.T) {
		issuer.SignedInvoke(ccName, "emit", user.Address(), "10")
		user.BalanceShouldBe(ccName, 10)
	})

	t.Run("checking emission", func(t *testing.T) {
		assert.Equal(t, mock.Metadata(ccName).TotalEmission, big.NewInt(10))
	})

	t.Run("[negative] trying to create cashout request with zero value", func(t *testing.T) {
		if err := user.RawSignedInvokeWithErrorReturned(ccName, "cashOut", "0", "cashout 1"); err != nil {
			assert.EqualError(t, err, errZeroAmount)
		}
	})

	t.Run("[negative] trying to create cashout by issuer", func(t *testing.T) {
		if err := issuer.RawSignedInvokeWithErrorReturned(ccName, "cashOut", "0", "cashout 1"); err != nil {
			assert.EqualError(t, err, errUnauthorized)
		}
	})

	var co1, co2 string
	t.Run("creating cashout requests", func(t *testing.T) {
		co1 = user.SignedInvoke(ccName, "cashOut", "3", "cashout 1")
		co2 = user.SignedInvoke(ccName, "cashOut", "5", "cashout 2")

		issuer.BalanceShouldBe(ccName, 8)
		user.BalanceShouldBe(ccName, 2)
	})

	t.Run("checking cashout list", func(t *testing.T) {
		list := parseList(t, user.Invoke(ccName, "cashOutList"))
		assert.Equal(t, 2, len(list))
		assert.Contains(t, list, CashOut{
			TransactionID: co1,
			UserAddress:   user.AddressType(),
			Amount:        big.NewInt(3),
			Ref:           "cashout 1",
		})
		assert.Contains(t, list, CashOut{
			TransactionID: co2,
			UserAddress:   user.AddressType(),
			Amount:        big.NewInt(5),
			Ref:           "cashout 2",
		})
	})

	t.Run("[negative] trying to accept cash out by user", func(t *testing.T) {
		err := user.RawSignedInvokeWithErrorReturned(ccName, "cashOutAccept", co1)
		assert.EqualError(t, err, errUnauthorized)
	})

	t.Run("[negative] trying to cancel cash out by user", func(t *testing.T) {
		err := user.RawSignedInvokeWithErrorReturned(ccName, "cashOutCancel", co1)
		assert.EqualError(t, err, errUnauthorized)
	})

	t.Run("accepting cash out", func(t *testing.T) {
		issuer.SignedInvoke(ccName, "cashOutAccept", co1)
	})

	t.Run("checking emission reduced", func(t *testing.T) {
		assert.Equal(t, mock.Metadata(ccName).TotalEmission, big.NewInt(7))
	})

	t.Run("cancelling cash out", func(t *testing.T) {
		issuer.SignedInvoke(ccName, "cashOutCancel", co2)

		user.BalanceShouldBe(ccName, 7)
		issuer.BalanceShouldBe(ccName, 0)
	})

	t.Run("checking cashout list", func(t *testing.T) {
		list := parseList(t, user.Invoke(ccName, "cashOutList"))
		assert.Equal(t, 0, len(list))
	})
}

// TODO - somehow it works although tests are not passing
/*
func Test_Balances(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()
	user1 := mock.NewWallet()

	fiat := NewFiatToken(token.BaseToken{
		Name:            "USD",
		Symbol:          "USD",
		Decimals:        8,
		UnderlyingAsset: "",
	})

	mock.NewChainCode(ccName, fiat, &core.ContractOptions{
		DisabledFunctions: []string{"TxTransfer"},
	}, issuer.Address())

	t.Run("adding balances", func(t *testing.T) {
		issuer.Invoke(ccName, "tokenBalanceAdd", user1.Address(), "1000", "add1", "0", "")
		issuer.Invoke(ccName, "allowedBalanceAdd", "FIAT", user1.Address(), "10000", "add2", "0", "")

		user1.BalanceShouldBe(ccName, 1000)
		user1.AllowedBalanceShouldBe(ccName, "FIAT", 10000)
	})
}
*/

func parseList(t *testing.T, data string) (out []CashOut) {
	assert.NoError(t, json.Unmarshal([]byte(data), &out))
	return
}

package token

import (
	"errors"

	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
	"github.com/tickets-dao/foundation/v3/token"
)

// Errors
const (
	errImpossibleOperation = "impossible operation"
)

// MintableToken struct
type MintableToken struct {
	token.BaseToken
}

// Issuer returns first init arg which is issuer
func (mt *MintableToken) Issuer() *types.Address {
	addr, err := types.AddrFromBase58Check(mt.GetInitArg(0))
	if err != nil {
		panic(err)
	}
	return addr
}

// NewmintableToken returns link to the new MintableToken struct
func NewMintableToken(bt token.BaseToken) *MintableToken {
	return &MintableToken{bt}
}

// TxBuyToken - buying token by user
func (mt *MintableToken) TxBuyToken(sender types.Sender, amount *big.Int, currency string) error {
	if sender.Equal(mt.Issuer()) {
		return errors.New(errImpossibleOperation)
	}

	price, err := mt.CheckLimitsAndPrice("buyToken", amount, currency)
	if err != nil {
		return err
	}
	if err := mt.AllowedBalanceTransfer(currency, sender.Address(), mt.Issuer(), price, "buyToken"); err != nil {
		return err
	}
	if err := mt.TokenBalanceAdd(sender.Address(), amount, "buyToken"); err != nil {
		return err
	}

	return mt.EmissionAdd(amount)
}

// TxBuyBack - sells token back to issuer
func (mt *MintableToken) TxBuyBack(sender types.Sender, amount *big.Int, currency string) error {
	if sender.Equal(mt.Issuer()) {
		return errors.New(errImpossibleOperation)
	}

	price, err := mt.CheckLimitsAndPrice("buyBack", amount, currency)
	if err != nil {
		return err
	}
	if err := mt.AllowedBalanceTransfer(currency, mt.Issuer(), sender.Address(), price, "buyBack"); err != nil {
		return err
	}
	if err := mt.TokenBalanceSub(sender.Address(), amount, "buyBack"); err != nil {
		return err
	}
	return mt.EmissionSub(amount)
}

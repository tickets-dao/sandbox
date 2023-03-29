package token

import (
	"encoding/json"
	"errors"

	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
	"github.com/tickets-dao/foundation/v3/token"
)

// FiatWithTrading - base struct
type FiatWithTrading struct {
	token.BaseToken
}

// CashOut - base struct
type CashOut struct {
	TransactionID string         `json:"transactionId"`
	UserAddress   *types.Address `json:"userAddress"`
	Amount        *big.Int       `json:"amount"`
	Ref           string         `json:"ref"`
}

const cashOutKey = "cashOut"

// Errors
const (
	errCashOutNotFound = "cash out with this key not found"
	errUnauthorized    = "unauthorized"
	errZeroAmount      = "amount should be more than zero"
)

// NewFiatToken creates fiat token
func NewFiatToken(bt token.BaseToken) *FiatWithTrading {
	return &FiatWithTrading{bt}
}

// Issuer returns first init arg which is issuer
func (ft *FiatWithTrading) Issuer() *types.Address {
	addr, err := types.AddrFromBase58Check(ft.GetInitArg(0))
	if err != nil {
		panic(err)
	}
	return addr
}

// TxEmit - emits fiat token
func (ft *FiatWithTrading) TxEmit(sender types.Sender, address *types.Address, amount *big.Int) error {
	if !sender.Equal(ft.Issuer()) {
		return errors.New(errUnauthorized)
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		return errors.New(errZeroAmount)
	}

	if err := ft.TokenBalanceAdd(address, amount, "txEmit"); err != nil {
		return err
	}
	return ft.EmissionAdd(amount)
}

// TxCashOut - sells token from user to emitent
func (ft *FiatWithTrading) TxCashOut(sender *types.Sender, amount *big.Int, ref string) error {
	if sender.Equal(ft.Issuer()) {
		return errors.New(errUnauthorized)
	}

	stub := ft.GetStub()
	txID := stub.GetTxID()

	if amount.Cmp(big.NewInt(0)) == 0 {
		return errors.New(errZeroAmount)
	}

	key, err := stub.CreateCompositeKey(cashOutKey, []string{txID})
	if err != nil {
		return err
	}

	jsonCashOut, err := json.Marshal(CashOut{
		TransactionID: txID,
		UserAddress:   sender.Address(),
		Amount:        amount,
		Ref:           ref,
	})
	if err != nil {
		return err
	}

	if err := stub.PutState(key, jsonCashOut); err != nil {
		return err
	}

	return ft.TokenBalanceTransfer(sender.Address(), ft.Issuer(), amount, "cashOut")
}

// TxCashOutAccept - accepts cashout
func (ft *FiatWithTrading) TxCashOutAccept(sender types.Sender, coID string) error {
	if !sender.Equal(ft.Issuer()) {
		return errors.New(errUnauthorized)
	}

	stub := ft.GetStub()

	key, err := stub.CreateCompositeKey(cashOutKey, []string{coID})
	if err != nil {
		return err
	}

	rawCashOut, err := stub.GetState(key)
	if err != nil {
		return err
	}

	if len(rawCashOut) <= 0 {
		return errors.New(errCashOutNotFound)
	}

	var cashOut CashOut
	err = json.Unmarshal([]byte(rawCashOut), &cashOut)
	if err != nil {
		return err
	}

	// delete cashout from state
	if err := stub.DelState(key); err != nil {
		return err
	}

	if err := ft.TokenBalanceSub(sender.Address(), cashOut.Amount, "cashOutAccept"); err != nil {
		return err
	}

	return ft.EmissionSub(cashOut.Amount)
}

// TxCashOutCancel - cancels cashout
func (ft *FiatWithTrading) TxCashOutCancel(sender types.Sender, coID string) error {
	if !sender.Equal(ft.Issuer()) {
		return errors.New(errUnauthorized)
	}

	stub := ft.GetStub()

	key, err := stub.CreateCompositeKey(cashOutKey, []string{coID})
	if err != nil {
		return err
	}

	rawCashOut, err := stub.GetState(key)
	if err != nil {
		return err
	}

	if len(rawCashOut) <= 0 {
		return errors.New(errCashOutNotFound)
	}

	var cashOut CashOut
	err = json.Unmarshal([]byte(rawCashOut), &cashOut)
	if err != nil {
		return err
	}

	// delete cashout from state
	if err := stub.DelState(key); err != nil {
		return err
	}

	return ft.TokenBalanceTransfer(sender.Address(), cashOut.UserAddress, cashOut.Amount, "cashOutCancel")
}

// QueryCashOutList - returns list of cashouts
func (ft *FiatWithTrading) QueryCashOutList() ([]CashOut, error) {
	stub := ft.GetStub()

	iter, err := stub.GetStateByPartialCompositeKey(cashOutKey, []string{})
	if err != nil {
		return nil, err
	}

	defer iter.Close()

	var result []CashOut

	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return nil, err
		}

		var cashOut CashOut
		err = json.Unmarshal(res.Value, &cashOut)
		if err != nil {
			return nil, err
		}

		result = append(result, cashOut)
	}

	return result, nil
}

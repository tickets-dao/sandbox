package token

import (
	"encoding/json"
	"fmt"

	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
)

const (
	tradingOrderKey = "trading"
)

// Order struct
type Order struct {
	From   types.Address `json:"from"`
	To     types.Address `json:"to"`
	Suffix string        `json:"suffix"`
	Amount *big.Int      `json:"amount"`
	Ref    string        `json:"ref"`
}

// TxOrderCreate - creates trading order
func (fwt *FiatWithTrading) TxOrderCreate(sender types.Sender, to types.Address, suffix string, amount *big.Int, ref string) (string, error) {
	if err := fwt.TokenBalanceSub(sender.Address(), amount, "create order"); err != nil {
		return "", err
	}
	txID := fwt.GetStub().GetTxID()
	o := Order{
		From:   sender.Address(),
		To:     to,
		Suffix: suffix,
		Amount: amount,
		Ref:    ref,
	}
	data, err := json.Marshal(o)
	if err != nil {
		return "", err
	}
	key, err := fwt.GetStub().CreateCompositeKey(tradingOrderKey, []string{txID})
	if err != nil {
		return "", err
	}
	return txID, fwt.GetStub().PutState(key, data)
}

// OrderWithID - order struct with ID included
type OrderWithID struct {
	ID string `json:"id"`
	Order
}

// OrderList - orders list struct
type OrderList struct {
	My    []OrderWithID `json:"my_list"`
	Other []OrderWithID `json:"other_list"`
}

// QueryOrderList - returns OrderList
func (fwt *FiatWithTrading) QueryOrderList(me types.Address) (OrderList, error) {
	list := OrderList{}
	iter, err := fwt.GetStub().GetStateByPartialCompositeKey(tradingOrderKey, []string{})
	if err != nil {
		return OrderList{}, err
	}
	defer iter.Close()

	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return OrderList{}, err
		}
		_, parts, err := fwt.GetStub().SplitCompositeKey(kv.Key)
		if err != nil {
			return OrderList{}, err
		}
		key := parts[0]
		data, err := fwt.GetStub().GetState(kv.Key)
		if err != nil {
			return OrderList{}, err
		}
		o := Order{}
		if err := json.Unmarshal(data, &o); err != nil {
			return OrderList{}, err
		}
		if o.From.Equal(me) {
			list.My = append(list.My, OrderWithID{key, o})
		} else {
			list.Other = append(list.Other, OrderWithID{key, o})
		}
	}
	return list, nil
}

// TxOrderCancel - cancels specified order
func (fwt *FiatWithTrading) TxOrderCancel(sender types.Sender, id string) error {
	if !sender.Equal(fwt.Issuer()) {
		return fmt.Errorf(errUnauthorized)
	}
	key, err := fwt.GetStub().CreateCompositeKey(tradingOrderKey, []string{id})
	if err != nil {
		return err
	}
	data, err := fwt.GetStub().GetState(key)
	if err != nil {
		return err
	}
	o := &Order{}
	if err := json.Unmarshal(data, o); err != nil {
		return err
	}
	if err := fwt.TokenBalanceAdd(o.From, o.Amount, "cancel order"); err != nil {
		return err
	}
	return fwt.GetStub().DelState(key)
}

// TxOrderFill - fills specified order
func (fwt *FiatWithTrading) TxOrderFill(sender types.Sender, id string) error {
	if !sender.Equal(fwt.Issuer()) {
		return fmt.Errorf(errUnauthorized)
	}

	key, err := fwt.GetStub().CreateCompositeKey(tradingOrderKey, []string{id})
	if err != nil {
		return err
	}
	data, err := fwt.GetStub().GetState(key)
	if err != nil {
		return err
	}
	o := &Order{}
	if err := json.Unmarshal(data, o); err != nil {
		return err
	}
	if err := fwt.TokenBalanceAdd(o.To, o.Amount, "fill order"); err != nil {
		return err
	}
	return fwt.GetStub().DelState(key)
}

package token

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tickets-dao/foundation/v3/core/types"
)

// RedeemRequest base struct
type RedeemRequest struct {
	TransactionID string        `json:"transactionId"`
	UserAddress   types.Address `json:"userAddress"`
	GroupName     string        `json:"groupName"`
	Amount        *big.Int      `json:"amounts"`
	Ref           string        `json:"ref"`
}

const redeemRequestKey = "it_redeem_req"

// TxCreateRedeemRequest creates redeem request
func (it *IndustrialToken) TxCreateRedeemRequest(sender types.Sender, groupName string, amount *big.Int, ref string) error {
	stub := it.GetStub()
	txID := stub.GetTxID()

	if amount.Cmp(big.NewInt(0)) == 0 {
		return errors.New("amount should be more than zero")
	}

	key, err := stub.CreateCompositeKey(redeemRequestKey, []string{txID})
	if err != nil {
		return err
	}

	if err := it.loadConfigUnlessLoaded(); err != nil {
		return err
	}
	if !it.config.Initialized {
		return errors.New("token is not initialized")
	}
	notFound := true
	for _, group := range it.config.Groups {
		if group.Id == groupName {
			notFound = false
			break
		}
	}
	if notFound {
		return fmt.Errorf("token group %s not found", groupName)
	}

	jsonRequest, err := json.Marshal(RedeemRequest{
		TransactionID: txID,
		UserAddress:   sender.Address(),
		GroupName:     groupName,
		Amount:        amount,
		Ref:           ref,
	})
	if err != nil {
		return err
	}

	if err := stub.PutState(key, jsonRequest); err != nil {
		return err
	}

	return it.IndustrialBalanceLock(groupName, sender.Address(), amount)
}

// QueryRedeemRequestsList returns list of redemption requests
func (it *IndustrialToken) QueryRedeemRequestsList() ([]RedeemRequest, error) {
	stub := it.GetStub()

	iter, err := stub.GetStateByPartialCompositeKey(redeemRequestKey, []string{})
	if err != nil {
		return nil, err
	}

	defer iter.Close()

	var result []RedeemRequest

	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return nil, err
		}

		var req RedeemRequest
		err = json.Unmarshal(res.Value, &req)
		if err != nil {
			return nil, err
		}

		result = append(result, req)
	}

	return result, nil
}

// TxAcceptRedeemRequest - accepts request for tokens redemption
func (it *IndustrialToken) TxAcceptRedeemRequest(sender types.Sender, requestID string, amount *big.Int, ref string) error {
	if !sender.Equal(it.Issuer()) {
		return errors.New("unauthorized")
	}

	bigIntZero := new(big.Int).SetInt64(0)
	// Check limits
	rate, _, err := it.GetRateAndLimits("redeem", "")
	if err != nil {
		return err
	}
	max := new(big.Int).SetBytes(rate.Max)
	min := new(big.Int).SetBytes(rate.Min)
	if amount.Cmp(min) < 0 || (max.Cmp(bigIntZero) > 0 && amount.Cmp(max) > 0) {
		return errors.New("incorrect amount")
	}

	stub := it.GetStub()

	key, err := stub.CreateCompositeKey(redeemRequestKey, []string{requestID})
	if err != nil {
		return err
	}

	rawRequest, err := stub.GetState(key)
	if err != nil {
		return err
	}

	if len(rawRequest) == 0 {
		return errors.New("request with this key not found")
	}

	var req RedeemRequest
	err = json.Unmarshal([]byte(rawRequest), &req)
	if err != nil {
		return err
	}

	// delete request from state
	if err := stub.DelState(key); err != nil {
		return err
	}

	if amount.Cmp(req.Amount) == 1 {
		return errors.New("wrong amount to redeem")
	}

	if err := it.IndustrialBalanceBurnLocked(it.Symbol+"_"+req.GroupName, req.UserAddress, amount, "tokens redemption"); err != nil {
		return err
	}

	returnAmount := new(big.Int).Sub(req.Amount, amount)

	if amount.Cmp(bigIntZero) > 0 {
		it.changeEmissionInGroup(req.GroupName, amount)
	}

	if returnAmount.Cmp(bigIntZero) > 0 {
		// returning to user not accepted amount of tokens
		return it.IndustrialBalanceUnLock(it.Symbol+"_"+req.GroupName, req.UserAddress, returnAmount)
	}

	return nil
}

// TxDenyRedeemRequest - denys request for tokens redemption
func (it *IndustrialToken) TxDenyRedeemRequest(sender types.Sender, requestID string) error {
	if !sender.Equal(it.Issuer()) {
		return errors.New("unauthorized")
	}

	stub := it.GetStub()

	key, err := stub.CreateCompositeKey(redeemRequestKey, []string{requestID})
	if err != nil {
		return err
	}

	rawRequest, err := stub.GetState(key)
	if err != nil {
		return err
	}

	if len(rawRequest) == 0 {
		return errors.New("request with this key not found")
	}

	// delete request from state
	if err := stub.DelState(key); err != nil {
		return err
	}

	var req RedeemRequest
	err = json.Unmarshal([]byte(rawRequest), &req)
	if err != nil {
		return err
	}

	// returning to user not accepted amount of tokens
	return it.IndustrialBalanceUnLock(it.Symbol+"_"+req.GroupName, req.UserAddress, req.Amount)
}

func (it *IndustrialToken) changeEmissionInGroup(groupName string, amount *big.Int) error {
	if err := it.loadConfigUnlessLoaded(); err != nil {
		return err
	}
	if !it.config.Initialized {
		return errors.New("token is not initialized")
	}
	notFound := true
	for _, group := range it.config.Groups {
		if group.Id == groupName {
			notFound = false

			if group.Emission == nil {
				group.Emission = new(big.Int).Bytes()
			}

			if new(big.Int).SetBytes(group.Emission).Cmp(amount) < 0 {
				return errors.New("emission can't become negative")
			}

			group.Emission = new(big.Int).Sub(new(big.Int).SetBytes(group.Emission), amount).Bytes()

			return it.saveConfig()
		}
	}
	if notFound {
		return fmt.Errorf("token group %s not found", groupName)
	}

	return nil
}

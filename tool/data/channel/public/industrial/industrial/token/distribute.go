package token

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
)

// DistribRequest base struct
type DistribRequest struct {
	TransactionID string              `json:"transactionId"`
	UserAddress   *types.Address      `json:"userAddress"`
	GroupsAmounts map[string]*big.Int `json:"groupsAmounts"`
	Ref           string              `json:"ref"`
}

const distribRequestKey = "it_distrib_req"

// TxCreateDistribRequest creates distribution request
func (it *IndustrialToken) TxCreateDistribRequest(sender types.Sender, args, ref string) error {
	stub := it.GetStub()
	txID := stub.GetTxID()

	key, err := stub.CreateCompositeKey(distribRequestKey, []string{txID})
	if err != nil {
		return err
	}

	argsArray := strings.Split(args, ",")
	if len(argsArray)%2 != 0 {
		return errors.New("wrong number of arguments")
	}

	// get limits
	rate, _, err := it.GetRateAndLimits("distribute", "")
	if err != nil {
		return err
	}
	if !it.config.Initialized {
		return errors.New("token is not initialized")
	}

	groupsAmounts := map[string]*big.Int{}
	groupName := ""
	for argNum, arg := range argsArray {
		if (argNum+1)%2 == 0 {
			groupAmount, ok := new(big.Int).SetString(arg, 10)
			if !ok {
				return errors.New("incorrect amount")
			}

			if !rate.InLimit(groupAmount) {
				return errors.New("amount out of limits")
			}

			oldVal, ok := groupsAmounts[groupName]
			if !ok {
				oldVal = big.NewInt(0)
			}
			groupsAmounts[groupName] = new(big.Int).Add(oldVal, groupAmount)
			continue
		} else {
			fGroupNameFound := false

			for _, group := range it.config.Groups {
				if arg == group.Id {
					groupName = group.Id
					fGroupNameFound = true
					break
				}
			}

			if !fGroupNameFound {
				return fmt.Errorf("token group %s not found", arg)
			}
		}
	}

	jsonRequest, err := json.Marshal(DistribRequest{
		TransactionID: txID,
		UserAddress:   sender.Address(),
		GroupsAmounts: groupsAmounts,
		Ref:           ref,
	})

	if err != nil {
		return err
	}

	if err := stub.PutState(key, jsonRequest); err != nil {
		return err
	}

	return nil
}

// QueryDistribRequestsList returns list of distribution requests
func (it *IndustrialToken) QueryDistribRequestsList() ([]DistribRequest, error) {
	stub := it.GetStub()

	iter, err := stub.GetStateByPartialCompositeKey(distribRequestKey, []string{})
	if err != nil {
		return nil, err
	}

	defer iter.Close()

	var result []DistribRequest

	for iter.HasNext() {
		res, err := iter.Next()
		if err != nil {
			return nil, err
		}

		var req DistribRequest
		err = json.Unmarshal(res.Value, &req)
		if err != nil {
			return nil, err
		}

		result = append(result, req)
	}

	return result, nil
}

// TxAcceptDistribRequest - accepts request for tokens distribution
func (it *IndustrialToken) TxAcceptDistribRequest(sender types.Sender, requestID, ref string) error {
	if !sender.Equal(it.Issuer()) {
		return errors.New("unauthorized")
	}

	stub := it.GetStub()

	key, err := stub.CreateCompositeKey(distribRequestKey, []string{requestID})
	if err != nil {
		return err
	}

	rawRequest, err := stub.GetState(key)
	if err != nil {
		return err
	}

	if len(rawRequest) <= 0 {
		return errors.New("request with this key not found")
	}

	var req DistribRequest
	err = json.Unmarshal([]byte(rawRequest), &req)
	if err != nil {
		return err
	}

	// delete request from state
	if err := stub.DelState(key); err != nil {
		return err
	}

	for group, amount := range req.GroupsAmounts {
		if err := it.IndustrialBalanceTransfer(it.Symbol+"_"+group, sender.Address(), req.UserAddress, amount, req.Ref); err != nil {
			return err
		}
	}

	return nil
}

// TxDenyDistribRequest - denys request for tokens distribution
func (it *IndustrialToken) TxDenyDistribRequest(sender types.Sender, requestID string) error {
	if !sender.Equal(it.Issuer()) {
		return errors.New("unauthorized")
	}

	stub := it.GetStub()

	key, err := stub.CreateCompositeKey(distribRequestKey, []string{requestID})
	if err != nil {
		return err
	}

	rawRequest, err := stub.GetState(key)
	if err != nil {
		return err
	}

	if len(rawRequest) <= 0 {
		return errors.New("request with this key not found")
	}

	// delete request from state
	if err := stub.DelState(key); err != nil {
		return err
	}

	return nil
}

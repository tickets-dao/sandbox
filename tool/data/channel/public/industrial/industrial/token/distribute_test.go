package token

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tickets-dao/foundation/v3/core"
	ma "github.com/tickets-dao/foundation/v3/mock"
	"github.com/tickets-dao/foundation/v3/proto"
)

func parseDistribList(t *testing.T, data string) (out []DistribRequest) {
	assert.NoError(t, json.Unmarshal([]byte(data), &out))
	return
}

func Test_DistribRequest(t *testing.T) {
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

	issuer.SignedInvoke("it", "setRate", "distribute", "", "1")
	issuer.SignedInvoke("it", "setLimits", "distribute", "", "1", "10")

	req1 := user.SignedInvoke("it", "createDistribRequest", "203010,1", "ref1")
	req2 := user.SignedInvoke("it", "createDistribRequest", "203009,1,203009,2,203010,2", "ref2")

	list := parseDistribList(t, user.Invoke("it", "distribRequestsList"))
	assert.Equal(t, 2, len(list))

	groupsAmounts1 := map[string]*big.Int{}
	groupsAmounts2 := map[string]*big.Int{}

	groupsAmounts1["203010"] = big.NewInt(1)
	groupsAmounts2["203009"] = big.NewInt(3)
	groupsAmounts2["203010"] = big.NewInt(2)

	assert.Contains(t, list, DistribRequest{
		TransactionID: req1,
		UserAddress:   user.AddressType(),
		GroupsAmounts: groupsAmounts1,
		Ref:           "ref1",
	})
	assert.Contains(t, list, DistribRequest{
		TransactionID: req2,
		UserAddress:   user.AddressType(),
		GroupsAmounts: groupsAmounts2,
		Ref:           "ref2",
	})

	issuer.SignedInvoke("it", "denyDistribRequest", req1)
	issuer.SignedInvoke("it", "acceptDistribRequest", req2, "ref3")

	user.IndustrialBalanceShouldBe("it", "203009", 3)
	user.IndustrialBalanceShouldBe("it", "203010", 2)
}

func Test_DistribRequestNotInitialize(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()
	user := mock.NewWallet()

	it := &IndustrialToken{
		Name:            "Industrial Token",
		Symbol:          "IT",
		Decimals:        8,
		UnderlyingAsset: "Palladium",
		config:          &proto.Industrial{},
	}

	mock.NewChainCode("it", it, &core.ContractOptions{}, issuer.Address())

	issuer.SignedInvoke("it", "setRate", "distribute", "", "1")
	issuer.SignedInvoke("it", "setLimits", "distribute", "", "1", "10")

	err := user.RawSignedInvokeWithErrorReturned("it", "createDistribRequest", "203010,1", "ref1")
	assert.Equal(t, "token is not initialized", err.Error())
}

func Test_DistribRequestLimits(t *testing.T) {
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

	issuer.SignedInvoke("it", "setRate", "distribute", "", "1")

	// check max limit
	issuer.SignedInvoke("it", "setLimits", "distribute", "", "1", "3")
	err := user.RawSignedInvokeWithErrorReturned("it", "createDistribRequest", "203010,10", "ref1")
	assert.Equal(t, "amount out of limits", err.Error())

	// check min limit
	issuer.SignedInvoke("it", "setLimits", "distribute", "", "3", "10")
	err = user.RawSignedInvokeWithErrorReturned("it", "createDistribRequest", "203010,1", "ref1")
	assert.Equal(t, "amount out of limits", err.Error())

	// check max unlim
	issuer.SignedInvoke("it", "setLimits", "distribute", "", "1", "0")
	err = user.RawSignedInvokeWithErrorReturned("it", "createDistribRequest", "203010,100", "ref1")
	assert.NoError(t, err)

	// check full unlim
	issuer.SignedInvoke("it", "setLimits", "distribute", "", "0", "0")
	err = user.RawSignedInvokeWithErrorReturned("it", "createDistribRequest", "203010,10", "ref1")
	assert.NoError(t, err)
}

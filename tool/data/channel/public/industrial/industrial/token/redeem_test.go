package token

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tickets-dao/foundation/v3/core"
	ma "github.com/tickets-dao/foundation/v3/mock"
)

func Test_RedeemRequest(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()
	user := mock.NewWallet()

	it := &IT{
		IndustrialToken{
			Name:            "Industrial Token",
			Symbol:          "GF27ILN061",
			Decimals:        8,
			UnderlyingAsset: "Cobalt",
		},
	}

	mock.NewChainCode("it", it, &core.ContractOptions{}, issuer.Address())

	issuer.OtfNbInvoke("it", "initialize")

	issuer.SignedInvoke("it", "transferIndustrial", user.Address(), "203101", "100", "")

	issuer.IndustrialBalanceShouldBe("it", "203101", 49999999999900)
	user.IndustrialBalanceShouldBe("it", "203101", 100)

	req1 := user.SignedInvoke("it", "createRedeemRequest", "203101", "1", "ref1")
	req2 := user.SignedInvoke("it", "createRedeemRequest", "203101", "2", "ref2")

	user.IndustrialBalanceShouldBe("it", "203101", 97)

	bigOne, _ := new(big.Int).SetString("1", 10)
	bigTwo, _ := new(big.Int).SetString("2", 10)

	list := parseRedeemList(t, user.Invoke("it", "redeemRequestsList"))
	assert.Equal(t, 2, len(list))

	assert.Contains(t, list, RedeemRequest{
		TransactionID: req1,
		UserAddress:   user.AddressType(),
		GroupName:     "203101",
		Amount:        bigOne,
		Ref:           "ref1",
	})
	assert.Contains(t, list, RedeemRequest{
		TransactionID: req2,
		UserAddress:   user.AddressType(),
		GroupName:     "203101",
		Amount:        bigTwo,
		Ref:           "ref2",
	})

	issuer.SignedInvoke("it", "denyRedeemRequest", req1)
	user.IndustrialBalanceShouldBe("it", "203101", 98)

	issuer.SignedInvoke("it", "acceptRedeemRequest", req2, "1", "ref3")

	issuer.IndustrialBalanceShouldBe("it", "203101", 49999999999900)
	user.IndustrialBalanceShouldBe("it", "203101", 99)

	timeOld, err := time.Parse(timeFormat, "01.01.2030 01:00:00")
	assert.NoError(t, err)

	md := mock.IndustrialMetadata("it")
	assert.Contains(t, md.Groups, ma.MetadataGroup{
		Name:         "203101",
		Amount:       new(big.Int).SetUint64(49999999999999),
		MaturityDate: timeOld,
		Note:         "Test note",
	})

}

func parseRedeemList(t *testing.T, data string) (out []RedeemRequest) {
	assert.NoError(t, json.Unmarshal([]byte(data), &out))
	return
}

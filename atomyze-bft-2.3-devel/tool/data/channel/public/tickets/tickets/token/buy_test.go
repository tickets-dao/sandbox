package token

import (
	"fmt"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
	"testing"

	"github.com/tickets-dao/foundation/v3/core"
	ma "github.com/tickets-dao/foundation/v3/mock"
)

func TestIndustrialToken_Buy(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()
	user := mock.NewWallet()

	it := &Contract{}

	mock.NewChainCode("it", it, &core.ContractOptions{}, issuer.Address())

	issuer.OtfNbInvoke("it", "initialize")

	stub := mock.GetStub("it")
	prices, err := stub.GetState(joinStateKey(issuer.Address(), pricesMapStateSubKey))
	if err != nil {
		t.Fatalf("failed to get prices map: %v", err)
	}

	fmt.Println("got prices: ", prices)

	user.AddAllowedBalance("it", rubCurrency, 5000)
	res, response, swaps := user.RawSignedInvoke("it", "buy", "parter", "1", "1", "2")
	fmt.Println(res, response, swaps)

	if response.Error != "" {
		t.Fatalf("failed to buy ticket: %v", err)
	}

	userAllowedBalance := AllowedBalanceGetAll(stub, user.AddressType())
	userTickets := IndustrialBalanceGet(stub, user.AddressType())

	fmt.Println("user rubles: ", userAllowedBalance)
	fmt.Println("user ticekts: ", userTickets)

	fmt.Println("invoke result: ", user.Invoke("it", "allowedBalanceOf", user.Address(), rubCurrency))

	user.AllowedBalanceShouldBe("it", rubCurrency, 4000)
	issuer.AllowedBalanceShouldBe("it", rubCurrency, 1000)

	user.IndustrialBalanceShouldBe("it", joinStateKey(issuer.Address(), "parter", "1", "1", "2"), 1)
	issuer.IndustrialBalanceShouldBe("it", joinStateKey(issuer.Address(), "parter", "1", "1", "2"), 0)

	// fmt.Println(mock.GetStub("it").GetState(joinStateKey(issuer.Address(), pricesMapStateSubKey)))

}

func (con *Contract) mustGetAllowedBalance(t *testing.T, address *types.Address) *big.Int {
	allowedBalance, err := con.AllowedBalanceGet(rubCurrency, address)
	if err != nil {
		t.Fatalf("failed to get rub allowed balance at '%s': %v", address.String(), err)
	}

	return allowedBalance
}

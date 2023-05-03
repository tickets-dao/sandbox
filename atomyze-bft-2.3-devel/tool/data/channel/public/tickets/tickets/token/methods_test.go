package token

import (
	"fmt"
	"testing"

	"github.com/tickets-dao/foundation/v3/core"
	ma "github.com/tickets-dao/foundation/v3/mock"
)

func TestIndustrialToken_Init(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()

	it := &Contract{}

	mock.NewChainCode("it", it, &core.ContractOptions{}, issuer.Address())

	issuer.OtfNbInvoke("it", "initialize")

	fmt.Println(mock.GetStub("it").GetState(joinStateKey(issuer.Address(), pricesMapStateSubKey)))
}

func TestIndustrialToken_QueryTickets(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()

	it := &Contract{}

	mock.NewChainCode("it", it, &core.ContractOptions{}, issuer.Address())

	issuer.OtfNbInvoke("it", "initialize")

	fmt.Println(mock.GetStub("it").GetState(joinStateKey(issuer.Address(), pricesMapStateSubKey)))

	fmt.Println(issuer.Invoke("it", "ticketsByCategory", "parter"))
	fmt.Println("test")
}

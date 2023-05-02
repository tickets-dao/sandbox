package token

import (
	"testing"

	"github.com/tickets-dao/foundation/v3/core"
	ma "github.com/tickets-dao/foundation/v3/mock"
)

func TestIndustrialToken_Prepare(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()

	it := &Contract{}

	mock.NewChainCode("it", it, &core.ContractOptions{}, issuer.Address())

	issuer.OtfNbInvoke("it", "initialize")

	// fmt.Println(mock.GetStub("it").GetState(joinStateKey(issuer.Address(), pricesMapStateSubKey)))

}

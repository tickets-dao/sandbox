package token

import (
	_ "embed"
	"fmt"
	"github.com/tickets-dao/foundation/v3/core"
	ma "github.com/tickets-dao/foundation/v3/mock"
	"testing"
)

func TestTickets_AddTicketers(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()
	ticketer := mock.NewWallet()

	it := &Contract{}

	mock.NewChainCode("it", it, &core.ContractOptions{}, issuer.Address())

	fmt.Println(issuer.OtfNbInvoke("it", "initV2"))

	result1, result2 := issuer.OtfNbInvoke(
		"it",
		"emission",
		string(defaultPriceCategoriesBytes),
		"Лебединое озеро 1",
		"Москва, центр",
		"2023-05-26 15:00:00",
	)

	fmt.Println(result1, result2)

	fmt.Println(issuer.SignedInvoke("it", "addTicketer", ticketer.Address()))

}

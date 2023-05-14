package token

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/tickets-dao/foundation/v3/core"
	ma "github.com/tickets-dao/foundation/v3/mock"
)

//go:embed integration_tests/testdata/default_price_categories.json
var defaultPriceCategoriesBytes []byte

func TestTickets_Emission(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()

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

	fmt.Println("issuer industrial balance: ", issuer.Invoke("it", "industrialBalanceOf", issuer.Address()))

	fmt.Println("issuer events: ", issuer.Invoke("it", "eventsByIssuer", issuer.Address()))

	fmt.Println("events by id", issuer.Invoke("it", "eventsByIDs", fmt.Sprintf(`["%s::1"]`, issuer.Address())))

}

func TestTickets_Check_Issuer_InfoAfterInit(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()

	it := &Contract{}

	mock.NewChainCode("it", it, &core.ContractOptions{}, issuer.Address())

	issuer.OtfNbInvoke("it", "initV2")

	result := issuer.Invoke("it", "allIssuersInfo")

	var expectedAllIssuersInfoString = fmt.Sprintf("[{\"parent\":\"%s\",\"next_event_id\":1,\"name\":\"Большой театр\"}]", issuer.Address())
	if result != expectedAllIssuersInfoString {
		t.Errorf("expected all issuers info '%s', got '%s'", expectedAllIssuersInfoString, result)
	}
}

func TestTickets_MarkUserAsIssuerAndGetAllIssuerInfo(t *testing.T) {
	mock := ma.NewLedger(t)
	issuer := mock.NewWallet()
	issuerCandidate := mock.NewWallet()

	it := &Contract{}

	mock.NewChainCode("it", it, &core.ContractOptions{}, issuer.Address())

	issuer.OtfNbInvoke("it", "initV2")
	result1, result2 := issuer.OtfNbInvoke("it", "markUserAsIssuer", issuerCandidate.Address(), "Малый театр")
	fmt.Println(result1, result2)

	result := issuer.Invoke("it", "allIssuersInfo")

	expectedAllIssuerInfo := []IssuerInfo{
		{
			Parent:      issuer.AddressType(),
			NextEventID: 1,
			Name:        "Большой театр",
			ID:          issuer.AddressType(),
		},
		{
			Parent:      issuer.AddressType(),
			NextEventID: 1,
			Name:        "Малый театр",
			ID:          issuerCandidate.AddressType(),
		},
	}

	expectedAllIssuerInfoBytes, _ := json.Marshal(expectedAllIssuerInfo)

	if result != string(expectedAllIssuerInfoBytes) {
		t.Errorf("expected all issuers info '%s',\n got '%s'", string(expectedAllIssuerInfoBytes), result)
	}
}

func TestPrintDefaultSeatsBytes(t *testing.T) {
	bytes, err := json.Marshal(priceCategoriesDefault)
	if err != nil {
		t.Errorf("failed to marshal defaul price categories: %v", err)
	}

	fmt.Println(string(bytes))
}

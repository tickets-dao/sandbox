package token

import (
	"crypto/md5"
	_ "embed"
	"encoding/hex"
	"fmt"
	"github.com/tickets-dao/foundation/v3/core"
	ma "github.com/tickets-dao/foundation/v3/mock"
	"math/rand"
	"testing"
	"time"
)

func TestTickets_PrepareAndBurnTicket(t *testing.T) {
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

	issuer.RawSignedInvokeWithErrorReturned(
		"it",
		"addTicketer",
		ticketer.Address(),
	)

	time.Sleep(time.Second)
	privateKey := generateRandomString(20)
	burningHash := md5.Sum(privateKey)

	result, txResp := issuer.OtfNbInvoke(
		"it",
		"prepare",
		fmt.Sprintf(`%s::1`, issuer.Address()),
		"parter",
		"1",
		"1",
		hex.EncodeToString(burningHash[:]),
	)

	fmt.Println(result, txResp)

	result, txResp = ticketer.OtfNbInvoke(
		"it",
		"burn",
		fmt.Sprintf(`%s::1`, issuer.Address()),
		"parter",
		"1",
		"1",
		string(privateKey),
	)

	fmt.Println(result, txResp)
}

const symbols = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func generateRandomString(length int) []byte {
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = symbols[rand.Intn(len(symbols))]
	}

	return result
}

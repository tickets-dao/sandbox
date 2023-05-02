package token

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
)

func (con *Contract) TxBurn(sender *types.Sender, categoryName string, sector, row, number int, burningPrivateKey string) (BurnEvent, error) {
	issuerAddress := con.Issuer().String()

	ticketKey := joinStateKey(
		issuerAddress, categoryName, strconv.Itoa(sector), strconv.Itoa(row), strconv.Itoa(number),
	)

	ticketBytes, err := con.GetStub().GetState(ticketKey)
	if err != nil {
		return BurnEvent{}, fmt.Errorf("failed to get ticket info from state: %v", err)
	}

	var ticket Ticket
	err = json.Unmarshal(ticketBytes, &ticket)
	if err != nil {
		return BurnEvent{}, fmt.Errorf("failed to unmarshal ticket from '%s': %v", string(ticketBytes), err)
	}

	burningHash := md5.Sum([]byte(burningPrivateKey))

	if ticket.BurningHash != string(burningHash[:]) {
		return BurnEvent{}, fmt.Errorf("bad burning key, got hash '%s' instead of '%s'", string(burningHash[:]), ticket.BurningHash)
	}

	err = con.IndustrialBalanceBurnLocked(ticketKey, types.AddrFromBytes([]byte(ticket.Owner)), new(big.Int).SetInt64(1), "event entrance")
	if err != nil {
		return BurnEvent{}, fmt.Errorf("failed to burn ticket: %v", err)
	}

	return BurnEvent{
		Owner:       ticket.Owner,
		Ticketer:    sender.Address().String(),
		Ticker:      ticketKey,
		BurningHash: ticket.BurningHash,
	}, nil
}

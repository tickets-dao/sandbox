package token

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
)

func (con *Contract) NBTxBurn(sender *types.Sender, eventID, categoryName string, row, number int, burningPrivateKey string) (BurnEvent, error) {

	ticketKey := createTicketID(eventID, categoryName, row, number)

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

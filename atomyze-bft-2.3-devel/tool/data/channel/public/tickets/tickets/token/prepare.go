package token

import (
	"encoding/json"
	"fmt"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
)

func (con *Contract) NBTxPrepare(sender *types.Sender, eventID, categoryName string, row, number int, newBurningHash string) (Ticket, error) {

	ticketKey := createTicketID(eventID, categoryName, row, number)

	balances, err := con.IndustrialBalanceGet(sender.Address())
	if err != nil {
		return Ticket{}, fmt.Errorf("failed to get industrial balances of sender '%s': %v", sender.Address(), err)
	}

	_, ok := balances[ticketKey]
	if !ok {
		return Ticket{}, fmt.Errorf("unathorized for ticket '%s'", ticketKey)
	}

	if err = con.IndustrialBalanceLock(ticketKey, sender.Address(), big.NewInt(1)); err != nil {
		return Ticket{}, fmt.Errorf("failed to lock ticket: %v", err)
	}

	ticket := Ticket{
		BurningHash: newBurningHash,
		Owner:       sender.Address().String(),
		Category:    categoryName,
		Row:         row,
		Number:      number,
		EventID:     eventID,
	}

	ticketBytes, err := json.Marshal(ticket)
	if err != nil {
		return Ticket{}, fmt.Errorf("failed to marshal ticket: %v", err)
	}

	err = con.GetStub().PutState(ticketKey, ticketBytes)
	if err != nil {
		return Ticket{}, fmt.Errorf("failed to update ticket: %v", err)
	}

	return ticket, nil
}

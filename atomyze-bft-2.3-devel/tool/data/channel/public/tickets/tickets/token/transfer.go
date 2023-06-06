package token

import (
	"encoding/json"
	"fmt"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
)

func (con *Contract) TxTransferTo(
	sender *types.Sender,
	address *types.Address,
	eventID, categoryName string,
	row, number int,
) (TransferEvent, error) {
	return con.transferTicket(sender, address, eventID, categoryName, row, number)
}

func (con *Contract) transferTicket(
	sender *types.Sender,
	address *types.Address,
	eventID, categoryName string,
	row, number int,
) (TransferEvent, error) {
	ticketID := createTicketID(eventID, categoryName, row, number)
	lg.Infof("transferring ticket '%s' from '%s' to '%s'", ticketID, sender.Address(), address)

	ticket := Ticket{
		BurningHash: "", // delete previous saved hash
		Owner:       sender.Address().String(),
		Category:    categoryName,
		Row:         row,
		Number:      number,
		EventID:     eventID,
	}
	ticketBytes, err := json.Marshal(ticket)
	if err != nil {
		return TransferEvent{}, fmt.Errorf("failed to marshal ticket: %v", err)
	}

	if err = con.GetStub().PutState(ticketID, ticketBytes); err != nil {
		return TransferEvent{}, fmt.Errorf("failed to update tickets info: %v", err)
	}

	err = con.IndustrialBalanceTransfer(ticketID, sender.Address(), address, big.NewInt(1), "transfer")
	if err != nil {
		return TransferEvent{}, fmt.Errorf("failed to transfer ticket: %v", err)
	}

	return TransferEvent{
		From:   sender.Address().String(),
		To:     address.String(),
		Price:  0,
		Ticker: ticketID,
	}, nil
}

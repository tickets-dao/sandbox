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
	ticketID := createTicketID(eventID, categoryName, row, number)
	lg.Infof("TxTransferTo '%s' ticketID '%s'", address, ticketID)
	return con.transferTicket(sender, address, ticketID)
}

func (con *Contract) transferTicket(
	sender *types.Sender,
	address *types.Address,
	ticketID string,
) (TransferEvent, error) {
	lg.Infof("transferring ticket '%s' from '%s' to '%s'", ticketID, sender.Address(), address)

	ticket, err := con.getTicket(ticketID)
	if err != nil {
		err = fmt.Errorf("failed to get ticket: %v", err)
	}

	ticket.BurningHash = ""
	ticket.Owner = address.String()

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

	transferEvent := TransferEvent{
		From:   sender.Address().String(),
		To:     address.String(),
		Price:  0,
		Ticker: ticketID,
	}

	lg.Infof("transferred ticket %s: %#v", ticketID, transferEvent)

	return transferEvent, nil
}

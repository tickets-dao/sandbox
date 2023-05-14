package token

import (
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
	lg.Infof("transfering ticket '%s' from '%s' to '%s'", ticketID, sender.Address(), address)

	err := con.IndustrialBalanceTransfer(ticketID, sender.Address(), address, big.NewInt(1), "transfer")
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

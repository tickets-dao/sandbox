package token

import (
	"fmt"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
)

var priceCategoriesDefault = []PriceCategory{
	{
		Name:  "parter",
		Seats: 5,
		Rows:  5,
		Price: big.NewInt(1000),
	},
	{
		Name:  "бельэтаж",
		Seats: 10,
		Rows:  3,
		Price: big.NewInt(500),
	},
}

const contractInitializedStateKey = "contract::initialized"

func (con *Contract) NBTxInitV2(sender *types.Sender) error {
	lg.Infof("this is init v2 for sender '%s'\n", sender.Address().String())

	isInitialized, err := con.GetStub().GetState(contractInitializedStateKey)
	if err != nil {
		return fmt.Errorf("failed to get contract init state: %v", err)
	}

	if isInitialized != nil {
		return fmt.Errorf("contract is already inited: %s", string(isInitialized))
	}

	err = con.saveIssuerInfo(sender.Address(), IssuerInfo{
		Parent:      sender.Address(),
		NextEventID: 1,
		Name:        "Большой театр",
		ID:          sender.Address(),
	})
	if err != nil {
		return fmt.Errorf("failed to save issuer info: %v", err)
	}

	if err = con.GetStub().PutState(contractInitializedStateKey, []byte("true")); err != nil {
		return fmt.Errorf("failed to set contract as initialized: %v", err)
	}

	return nil
}

package token

import (
	"encoding/json"
	"fmt"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
)

var priceCategories = []PriceCategory{
	{
		Name: "parter",
		Seats: []Seat{
			{
				Sector: 1,
				Row:    1,
				Number: 1,
			},
			{
				Sector: 1,
				Row:    1,
				Number: 2,
			},

			{
				Sector: 1,
				Row:    1,
				Number: 3,
			},
			{
				Sector: 1,
				Row:    1,
				Number: 4,
			},
			{
				Sector: 1,
				Row:    2,
				Number: 1,
			},
			{
				Sector: 1,
				Row:    2,
				Number: 2,
			},
		},
		Price: big.NewInt(1000),
	},
}

// NBTxInitialize - initializes chaincode
func (con *Contract) NBTxInitialize(sender *types.Sender) error {
	lg.Infof("this is nbtx initialize for sender %s, issuer: '%s'\n", sender.Address().String(), con.Issuer())

	return con.CustomInitialize(priceCategories)
}

// CustomInitialize - initial tickets0 generation
func (con *Contract) CustomInitialize(priceCategories []PriceCategory) error {
	lg.Infof("this is custom init, going to start, issuer: '%s'\n", con.Issuer())

	lg.Infof("price categories: %+v\n", priceCategories)

	categoriesMap := make(map[string]*big.Int, len(priceCategories))
	for _, category := range priceCategories {
		if _, ok := categoriesMap[category.Name]; ok {
			return fmt.Errorf("category '%s' is used more than once", category.Name)
		}

		categoriesMap[category.Name] = category.Price
	}

	issuerAddress := con.Issuer().String()

	categoriesMapBytes, err := json.Marshal(categoriesMap)
	if err != nil {
		return err
	}

	lg.Infof("categories map: '%s'\n", string(categoriesMapBytes))

	stub := con.GetStub()
	if err = stub.PutState(joinStateKey(issuerAddress, pricesMapStateSubKey), categoriesMapBytes); err != nil {
		return fmt.Errorf("failed to save categories map state: %v", err)
	}

	if err = stub.PutState(joinStateKey(issuerAddress, ticketersStateSubKey), []byte("[]")); err != nil {
		return fmt.Errorf("failed to save ticketers addreses: %v", err)
	}

	if err = stub.PutState(joinStateKey(issuerAddress, issuerBalanceStateSubKey), []byte("0")); err != nil {
		return fmt.Errorf("failed to save ticketers addreses: %v", err)
	}

	if err = stub.PutState(joinStateKey(issuerAddress, buyBackRateStateSubKey), []byte("1")); err != nil {
		return fmt.Errorf("failed to save ticketers addreses: %v", err)
	}

	issuerInfoBytes, _ := json.Marshal(IssuerInfo{Parent: con.Issuer(), NextEventID: 1})
	if err = stub.PutState(issuerAddress, issuerInfoBytes); err != nil {
		return fmt.Errorf("failed to put issuer info: %v", err)
	}

	// билеты, которые нужно выпустить, соответствует ключу в стейте
	ticketKeys := make([]string, 0, len(priceCategories))

	for _, category := range priceCategories {
		for _, seat := range category.Seats {
			ticketID := con.createTicketID(category.Name, seat.Sector, seat.Row, seat.Number)
			ticketKeys = append(ticketKeys, ticketID)
		}
	}

	for _, ticketID := range ticketKeys {
		lg.Infof("got ticketID '%s'", ticketID)
		err = con.IndustrialBalanceAdd("ticket_"+ticketID, con.Issuer(), new(big.Int).SetInt64(1), "initial emission")
		if err != nil {
			return fmt.Errorf("failed to emit ticket %s: %v", ticketID, err)
		}
	}

	lg.Infof("all done, returning nil error\n")

	return nil
}

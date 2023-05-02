package token

import (
	"encoding/json"
	"fmt"
	"time"

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
	lg.Infof("this is nbtx initialize for sender %s\n", sender.Address().String())
	if con.issuer != nil {
		lg.Warningf("contract already initialised, going to exit")
		return nil
	}

	con.issuer = sender.Address()

	lg.Infof("going to start custom init")

	return con.CustomInitialize(priceCategories, "Лебединое озеро", "Театральная площадь, 1", time.Date(2023, 5, 16, 19, 00, 00, 00, time.Local))
}

// CustomInitialize - initial tickets0 generation
func (con *Contract) CustomInitialize(priceCategories []PriceCategory, eventName, eventAddress string, eventDate time.Time) error {
	lg.Infof("this is custom init, going to start, metadata: %+v\n", contractMetadata)
	if contractMetadata != nil {
		lg.Warningf("contract metadata is not nil (%+v), returning", contractMetadata)
		return nil
	}

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

	if err = con.GetStub().PutState(joinStateKey(issuerAddress, pricesMapStateSubKey), categoriesMapBytes); err != nil {
		return fmt.Errorf("failed to save categories map state: %v", err)
	}

	if err = con.GetStub().PutState(joinStateKey(issuerAddress, ticketersStateSubKey), []byte("[]")); err != nil {
		return fmt.Errorf("failed to save ticketers addreses: %v", err)
	}

	if err = con.GetStub().PutState(joinStateKey(issuerAddress, issuerBalanceStateSubKey), []byte("0")); err != nil {
		return fmt.Errorf("failed to save ticketers addreses: %v", err)
	}

	if err = con.GetStub().PutState(joinStateKey(issuerAddress, buyBackRateStateSubKey), []byte("1")); err != nil {
		return fmt.Errorf("failed to save ticketers addreses: %v", err)
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

	contractMetadata = &Metadata{
		EventStart:   eventDate,
		EventName:    eventName,
		EventAddress: eventAddress,
		Issuer:       con.Issuer(),
		Verifiers:    []*types.Address{},
	}

	lg.Infof("all done, returning nil error\n")

	return nil
}

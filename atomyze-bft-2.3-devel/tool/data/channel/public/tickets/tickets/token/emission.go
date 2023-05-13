package token

import (
	"encoding/json"
	"fmt"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
	"strconv"
)

// NBTxEmission - create tickets emission
func (con *Contract) NBTxEmission(sender *types.Sender, priceCategoriesString string) error {
	lg.Infof("this is new emission for sender %s\n", sender.Address().String())

	issuerInfo, err := con.getIssuerInfo(sender.Address())
	if err != nil {
		return fmt.Errorf("failed to get issuerInfo")
	}

	eventID := joinStateKey(sender.Address().String(), strconv.Itoa(issuerInfo.NextEventID))
	issuerInfo.NextEventID++

	if err = con.saveIssuerInfo(sender.Address(), issuerInfo); err != nil {
		return fmt.Errorf("failed to save issuer info: %v", err)
	}

	// TODO save event info as a composite key
	// should be used for getting list of event per issuer

	lg.Infof("price categories: %s\n", priceCategoriesString)

	var priceCategories PriceCategories
	if err = json.Unmarshal([]byte(priceCategoriesString), &priceCategories); err != nil {
		return fmt.Errorf("failed to unmarshal price categories: %v", err)
	}

	if err = con.saveCategoriesMap(eventID, priceCategories); err != nil {
		return err
	}

	stub := con.GetStub()

	if err = stub.PutState(joinStateKey(eventID, ticketersStateSubKey), []byte("[]")); err != nil {
		return fmt.Errorf("failed to save ticketers addreses: %v", err)
	}

	if err = stub.PutState(joinStateKey(eventID, issuerBalanceStateSubKey), []byte("0")); err != nil {
		return fmt.Errorf("failed to save ticketers addreses: %v", err)
	}

	if err = stub.PutState(joinStateKey(eventID, buyBackRateStateSubKey), []byte("1")); err != nil {
		return fmt.Errorf("failed to save ticketers addreses: %v", err)
	}

	if err = con.emitTickets(sender.Address(), eventID, priceCategories); err != nil {
		return err
	}

	lg.Infof("all done, returning nil error\n")

	return nil
}

func (con *Contract) saveCategoriesMap(eventID string, categories PriceCategories) error {
	categoriesMap := make(map[string]*big.Int, len(categories))
	for _, category := range categories {
		if _, ok := categoriesMap[category.Name]; ok {
			return fmt.Errorf("category '%s' is used more than once", category.Name)
		}

		categoriesMap[category.Name] = category.Price
	}

	categoriesMapBytes, err := json.Marshal(categoriesMap)
	if err != nil {
		return fmt.Errorf("failed to marshal categories map: %v", err)
	}

	lg.Infof("categories map: '%s'\n", string(categoriesMapBytes))

	stub := con.GetStub()
	if err = stub.PutState(joinStateKey(eventID, pricesMapStateSubKey), categoriesMapBytes); err != nil {
		return fmt.Errorf("failed to save categories map state: %v", err)
	}

	return nil
}

func (con *Contract) emitTickets(issuer *types.Address, eventID string, categories PriceCategories) error {
	// билеты, которые нужно выпустить, соответствует ключу в стейте
	ticketKeys := make([]string, 0, len(categories))
	var err error

	for _, category := range categories {
		for _, seat := range category.Seats {
			ticketID := con.createTicketID(eventID, category.Name, seat.Sector, seat.Row, seat.Number)
			ticketKeys = append(ticketKeys, ticketID)
		}
	}

	for _, ticketID := range ticketKeys {
		lg.Infof("got ticketID '%s'", ticketID)
		err = con.IndustrialBalanceAdd(ticketID, issuer, new(big.Int).SetInt64(1), "initial emission")
		if err != nil {
			return fmt.Errorf("failed to emit ticket %s: %v", ticketID, err)
		}
	}

	return nil
}

package token

import (
	"fmt"
	"github.com/tickets-dao/chaincode/datastructures/set"
	"github.com/tickets-dao/foundation/v3/core/types"
	"sort"
)

func (con *Contract) TxAddTicketer(sender *types.Sender, newTicketer *types.Address) error {
	lg.Infof("starting to add new ticketer '%s' for issuer '%s'", newTicketer, sender.Address())
	issuerInfo, err := con.getIssuerInfo(sender.Address())
	if err != nil {
		return err
	}

	ticketerSet := set.FromSlice(issuerInfo.Ticketers)
	if ticketerSet.Contains(newTicketer.String()) {
		return fmt.Errorf("ticketer '%s' is already ticketer", newTicketer)
	}

	ticketerSet.Add(newTicketer.String())

	ticketersSlice := ticketerSet.ToSlice()
	sort.Slice(ticketersSlice, func(i, j int) bool {
		return ticketersSlice[i] <= ticketersSlice[j]
	})

	issuerInfo.Ticketers = ticketersSlice

	if err = con.saveIssuerInfo(sender.Address(), issuerInfo); err != nil {
		return err
	}

	lg.Infof("added new ticketer, total %d ticketers", len(ticketersSlice))

	return nil
}

func (con *Contract) TxDeleteTicketer(sender *types.Sender, newTicketer *types.Address) error {
	lg.Infof("starting to delete new ticketer '%s' for issuer '%s'", newTicketer, sender.Address())
	issuerInfo, err := con.getIssuerInfo(sender.Address())
	if err != nil {
		return err
	}

	ticketerSet := set.FromSlice(issuerInfo.Ticketers)
	if ticketerSet.Contains(newTicketer.String()) {
		return fmt.Errorf("ticketer '%s' is already ticketer", newTicketer)
	}

	ticketerSet.Delete(newTicketer.String())

	ticketersSlice := ticketerSet.ToSlice()
	sort.Strings(ticketersSlice)

	issuerInfo.Ticketers = ticketersSlice

	if err = con.saveIssuerInfo(sender.Address(), issuerInfo); err != nil {
		return err
	}

	lg.Infof("deleted new ticketer, total %d ticketers", len(ticketersSlice))

	return nil
}

func (con *Contract) QueryTicketers(issuer *types.Address) ([]string, error) {
	lg.Infof("getting ticketers for issuer '%s'", issuer)
	issuerInfo, err := con.getIssuerInfo(issuer)
	if err != nil {
		return nil, err
	}

	sort.Strings(issuerInfo.Ticketers)

	lg.Infof("got %d ticketers", len(issuerInfo.Ticketers))

	return issuerInfo.Ticketers, nil
}

func (con *Contract) checkTicketer(issuer, maybeTicketer *types.Address) error {
	lg.Infof("checking that '%s' is ticketer for issuer '%s'", maybeTicketer, issuer)
	issuerInfo, err := con.getIssuerInfo(issuer)
	if err != nil {
		return err
	}

	ticketerSet := set.FromSlice(issuerInfo.Ticketers)
	if !ticketerSet.Contains(maybeTicketer.String()) {
		return fmt.Errorf("address '%s' is not ticketer", maybeTicketer)
	}

	return nil
}

package token

import (
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
)

func (fwt *FiatWithTrading) TxTokenBalanceAdd(addr *types.Address, amount *big.Int, reason string, reasonId uint32, correlatedTxId string) error {
	return fwt.TokenBalanceAdd(addr, amount, reason)
}

func (fwt *FiatWithTrading) TxTokenBalanceSub(addr *types.Address, amount *big.Int, reason string, reasonId uint32, correlatedTxId string) error {
	return fwt.TokenBalanceSub(addr, amount, reason)
}

func (fwt *FiatWithTrading) TxAllowedBalanceAdd(token string, addr *types.Address, amount *big.Int, reason string, reasonId uint32, correlatedTxId string) error {
	return fwt.AllowedBalanceAdd(token, addr, amount, reason)
}

func (fwt *FiatWithTrading) TxAllowedBalanceSub(token string, addr *types.Address, amount *big.Int, reason string, reasonId uint32, correlatedTxId string) error {
	return fwt.AllowedBalanceSub(token, addr, amount, reason)
}

package proto

import "github.com/tickets-dao/foundation/v3/core/types/big"

func (x *TokenRate) InLimit(amount *big.Int) bool {
	maxLimit := new(big.Int).SetBytes(x.Max)
	minLimit := new(big.Int).SetBytes(x.Min)

	return amount.Cmp(minLimit) >= 0 && (maxLimit.Cmp(big.NewInt(0)) == 0 || amount.Cmp(maxLimit) <= 0)
}

func (x *TokenRate) CalcPrice(amount *big.Int, rateDecimal uint64) *big.Int {
	return new(big.Int).Div(
		new(big.Int).Mul(
			amount, new(big.Int).SetBytes(x.Rate),
		),
		new(big.Int).Exp(
			new(big.Int).SetUint64(10), //nolint:gomnd
			new(big.Int).SetUint64(rateDecimal), nil,
		),
	)
}

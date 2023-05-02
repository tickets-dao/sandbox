package proto

import (
	"encoding/hex"
	"encoding/json"

	"github.com/btcsuite/btcutil/base58"
)

type addressDump struct {
	UserID       string `json:"user_id,omitempty"` //nolint:tagliatelle
	Address      string `json:"address,omitempty"`
	IsIndustrial bool   `json:"is_industrial,omitempty"` //nolint:tagliatelle
	IsMultisig   bool   `json:"is_multisig,omitempty"`   //nolint:tagliatelle
}

type pendingTxDump struct {
	Method     string       `json:"method"`
	Sender     *addressDump `json:"sender"`
	Args       []string     `json:"args"`
	CreatorSKI string       `json:"creator_ski"` //nolint:tagliatelle
	Timestamp  int64
	Nonce      uint64
}

func (x *PendingTx) DumpJSON() []byte {
	var sender *addressDump
	if x.Sender != nil {
		sender = &addressDump{
			UserID:       x.Sender.UserID,
			Address:      base58.CheckEncode(x.Sender.Address[1:], x.Sender.Address[0]),
			IsIndustrial: x.Sender.IsIndustrial,
			IsMultisig:   x.Sender.IsMultisig,
		}
	}

	data, err := json.MarshalIndent(&pendingTxDump{
		Method:     x.Method,
		Sender:     sender,
		Args:       x.Args,
		CreatorSKI: hex.EncodeToString(x.CreatorSKI),
		Timestamp:  x.Timestamp,
		Nonce:      x.Nonce,
	}, "", "  ")
	if err != nil {
		panic(err)
	}
	return data
}

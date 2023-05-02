package mock

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/btcsuite/btcutil/base58"
	pb "github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/tickets-dao/foundation/v3/core"
	"github.com/tickets-dao/foundation/v3/core/types"
	"github.com/tickets-dao/foundation/v3/core/types/big"
	"github.com/tickets-dao/foundation/v3/mock/stub"
	"github.com/tickets-dao/foundation/v3/proto"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"
)

const batchRobotCert = "0a0a61746f6d797a654d535012d7062d2d2d2d2d424547494e2043455254494649434154452d2d2d2d2d0a4d494943536a434341664367417749424167495241496b514e37444f456b6836686f52425057633157495577436759494b6f5a497a6a304541774977675963780a437a414a42674e5642415954416c56544d524d77455159445651514945777044595778705a6d3979626d6c684d525977464159445651514845773154595734670a526e4a68626d4e7063324e764d534d77495159445651514b45787068644739746558706c4c6e56686443356b624851755958527662586c365a53356a6144456d0a4d4351474131554541784d64593245755958527662586c365a533531595851755a4778304c6d463062323135656d5575593267774868634e4d6a41784d44457a0a4d4467314e6a41775768634e4d7a41784d4445784d4467314e6a4177576a42324d517377435159445651514745774a56557a45544d4245474131554543424d4b0a5132467361575a76636d3570595445574d4251474131554542784d4e5532467549455a795957356a61584e6a627a45504d4130474131554543784d47593278700a5a5735304d536b774a7759445651514444434256633256794d554268644739746558706c4c6e56686443356b624851755958527662586c365a53356a6144425a0a4d424d4742797147534d34394167454743437147534d3439417745484130494142427266315057484d51674d736e786263465a346f3579774b476e677830594e0a504b6270494335423761446f6a46747932576e4871416b5656723270697853502b4668497634434c634935633162473963365a375738616a5454424c4d4134470a41315564447745422f775145417749486744414d42674e5648524d4241663845416a41414d437347413155644977516b4d434b4149464b2f5335356c6f4865700a6137384441363173364e6f7433727a4367436f435356386f71462b37585172344d416f4743437147534d343942414d43413067414d4555434951436e6870476d0a58515664754b632b634266554d6b31494a6835354444726b3335436d436c4d657041533353674967596b634d6e5a6b385a42727179796953544d6466526248740a5a32506837364e656d536b62345651706230553d0a2d2d2d2d2d454e442043455254494649434154452d2d2d2d2d0a" //nolint:gofumpt
const userCert = `MIICSTCCAe+gAwIBAgIQW3KyKC2acfVxSNneRkHZPjAKBggqhkjOPQQDAjCBhzEL
MAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNhbiBG
cmFuY2lzY28xIzAhBgNVBAoTGmF0b215emUudWF0LmRsdC5hdG9teXplLmNoMSYw
JAYDVQQDEx1jYS5hdG9teXplLnVhdC5kbHQuYXRvbXl6ZS5jaDAeFw0yMDEwMTMw
ODU2MDBaFw0zMDEwMTEwODU2MDBaMHYxCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpD
YWxpZm9ybmlhMRYwFAYDVQQHEw1TYW4gRnJhbmNpc2NvMQ8wDQYDVQQLEwZjbGll
bnQxKTAnBgNVBAMMIFVzZXI5QGF0b215emUudWF0LmRsdC5hdG9teXplLmNoMFkw
EwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEp5H9GVCTmUnVo8dHBTCT7cHmK4xn2X+Y
jJEsrbhodUt9GjUx04uOo05uRWhOI+O4fi0EEu+RSkx98hFUapWfRqNNMEswDgYD
VR0PAQH/BAQDAgeAMAwGA1UdEwEB/wQCMAAwKwYDVR0jBCQwIoAgUr9LnmWgd6lr
vwMDrWzo2i3evMKAKgJJXyioX7tdCvgwCgYIKoZIzj0EAwIDSAAwRQIhAPUozDTR
MOS4WBh87DbsJjI8gIuXPGXwoFXDQQhc2gz0AiAz9jt95z3MKnwj0dWPhjnzAGP8
8PrsVxYtGp6/TnpiPQ==` //nolint:gofumpt
const shouldNotBeHereMsg = "shouldn't be here"

type Wallet struct {
	ledger *Ledger
	pKey   ed25519.PublicKey
	sKey   ed25519.PrivateKey
	addr   string
}

// ChangeKeys change private key, then public key will be derived and changed too
func (w *Wallet) ChangeKeys(sKey ed25519.PrivateKey) error {
	w.sKey = sKey
	var ok bool
	w.pKey, ok = sKey.Public().(ed25519.PublicKey)
	if !ok {
		return errors.New("failed to derive public key from secret")
	}
	return nil
}

func (w *Wallet) Address() string {
	return w.addr
}

func (w *Wallet) PubKey() []byte {
	return w.pKey
}

func (w *Wallet) SecretKey() []byte {
	return w.sKey
}

func (w *Wallet) SetPubKey(pk ed25519.PublicKey) {
	w.pKey = pk
}

func (w *Wallet) AddressType() *types.Address {
	value, ver, err := base58.CheckDecode(w.addr)
	if err != nil {
		panic(err)
	}
	return &types.Address{Address: append([]byte{ver}, value...)[:32]}
}

func (w *Wallet) addBalance(stub *stub.Stub, amount *big.Int, balanceType core.StateKey, path ...string) {
	prefix := hex.EncodeToString([]byte{byte(balanceType)})
	key, err := stub.CreateCompositeKey(prefix, append([]string{w.Address()}, path...))
	assert.NoError(w.ledger.t, err)
	data := stub.State[key]
	balance := new(big.Int).SetBytes(data)
	newBalance := new(big.Int).Add(balance, amount)
	_ = stub.PutBalanceToState(key, newBalance)
}

func (w *Wallet) CheckGivenBalanceShouldBe(ch string, token string, expectedBalance uint64) {
	st := w.ledger.stubs[ch]
	prefix := hex.EncodeToString([]byte{byte(core.StateKeyGivenBalance)})
	key, err := st.CreateCompositeKey(prefix, []string{token})
	assert.NoError(w.ledger.t, err)
	bytes := st.State[key]
	if bytes == nil && expectedBalance == 0 {
		return
	}
	actualBalanceInt := new(big.Int).SetBytes(bytes)
	expectedBalanceInt := new(big.Int).SetUint64(expectedBalance)
	assert.Equal(w.ledger.t, expectedBalanceInt, actualBalanceInt)
}

func (w *Wallet) AddBalance(ch string, amount uint64) {
	w.addBalance(w.ledger.stubs[ch], new(big.Int).SetUint64(amount), core.StateKeyTokenBalance)
}

func (w *Wallet) AddAllowedBalance(ch string, token string, amount uint64) {
	w.addBalance(w.ledger.stubs[ch], new(big.Int).SetUint64(amount), core.StateKeyAllowedBalance, token)
}

func (w *Wallet) AddGivenBalance(ch string, givenBalanceChannel string, amount uint64) {
	st := w.ledger.stubs[ch]
	prefix := hex.EncodeToString([]byte{byte(core.StateKeyGivenBalance)})
	key, err := st.CreateCompositeKey(prefix, []string{givenBalanceChannel})
	assert.NoError(w.ledger.t, err)
	newBalance := new(big.Int).SetUint64(amount)
	_ = st.PutBalanceToState(key, newBalance)
}

func (w *Wallet) AddTokenBalance(ch string, token string, amount uint64) {
	parts := strings.Split(token, "_")
	w.addBalance(w.ledger.stubs[ch], new(big.Int).SetUint64(amount), core.StateKeyTokenBalance, parts[len(parts)-1])
}

func (w *Wallet) BalanceShouldBe(ch string, expected uint64) {
	assert.Equal(w.ledger.t, "\""+strconv.FormatUint(expected, 10)+"\"", w.Invoke(ch, "balanceOf", w.Address())) //nolint:gomnd
}

func (w *Wallet) AllowedBalanceShouldBe(ch string, token string, expected uint64) {
	assert.Equal(w.ledger.t, "\""+strconv.FormatUint(expected, 10)+"\"", w.Invoke(ch, "allowedBalanceOf", w.Address(), token)) //nolint:gomnd
}

func (w *Wallet) OtfBalanceShouldBe(ch string, token string, expected uint64) {
	assert.Equal(w.ledger.t, "\""+strconv.FormatUint(expected, 10)+"\"", w.Invoke(ch, "getBalance", w.Address(), token)) //nolint:gomnd
}

func (w *Wallet) IndustrialBalanceShouldBe(ch, group string, expected uint64) {
	var balances map[string]string
	res := w.Invoke(ch, "industrialBalanceOf", w.Address())
	assert.NoError(w.ledger.t, json.Unmarshal([]byte(res), &balances))

	if balance, ok := balances[group]; ok {
		assert.Equal(w.ledger.t, strconv.FormatUint(expected, 10), balance) //nolint:gomnd
		return
	}
	if expected == 0 {
		return
	}
	assert.Fail(w.ledger.t, "group not found")
}

func (w *Wallet) GroupBalanceShouldBe(ch, group string, expected uint64) {
	var balances map[string]string
	res := w.Invoke(ch, "groupBalanceOf", w.Address())
	assert.NoError(w.ledger.t, json.Unmarshal([]byte(res), &balances))

	if balance, ok := balances[group]; ok {
		assert.Equal(w.ledger.t, strconv.FormatUint(expected, 10), balance) //nolint:gomnd
		return
	}
	if expected == 0 {
		return
	}
	assert.Fail(w.ledger.t, "group not found")
}

func (w *Wallet) Invoke(ch string, fn string, args ...string) string {
	return w.ledger.doInvoke(ch, txIDGen(), fn, args...)
}

func (w *Wallet) InvokeReturnsTxID(ch string, fn string, args ...string) string {
	txID := txIDGen()
	w.ledger.doInvoke(ch, txID, fn, args...)
	return txID
}

func (w *Wallet) InvokeWithError(ch string, fn string, args ...string) error {
	return w.ledger.doInvokeWithErrorReturned(ch, txIDGen(), fn, args...)
}

func (w *Wallet) SignArgs(ch string, fn string, args ...string) []string {
	resp, _ := w.sign(fn, ch, args...)
	return resp
}

func (w *Wallet) BatchedInvoke(ch string, fn string, args ...string) (string, TxResponse) {
	txID := txIDGen()
	w.ledger.doInvoke(ch, txID, fn, args...)

	id, err := hex.DecodeString(txID)
	assert.NoError(w.ledger.t, err)
	data, err := pb.Marshal(&proto.Batch{TxIDs: [][]byte{id}})
	assert.NoError(w.ledger.t, err)

	cert, err := hex.DecodeString(batchRobotCert)
	assert.NoError(w.ledger.t, err)
	w.ledger.stubs[ch].SetCreator(cert)
	res := w.Invoke(ch, "batchExecute", string(data))
	out := &proto.BatchResponse{}
	assert.NoError(w.ledger.t, pb.Unmarshal([]byte(res), out))

	e := <-w.ledger.stubs[ch].ChaincodeEventsChannel
	if e.EventName == batchExecute {
		events := &proto.BatchEvent{}
		assert.NoError(w.ledger.t, pb.Unmarshal(e.Payload, events))
		for _, ev := range events.Events {
			if hex.EncodeToString(ev.Id) == txID {
				evts := make(map[string][]byte)
				for _, evt := range ev.Events {
					evts[evt.Name] = evt.Value
				}
				er := ""
				if ev.Error != nil {
					er = ev.Error.Error
				}
				return txID, TxResponse{
					Method: ev.Method,
					Error:  er,
					Result: string(ev.Result),
					Events: evts,
				}
			}
		}
	}
	assert.Fail(w.ledger.t, shouldNotBeHereMsg)
	return txID, TxResponse{}
}

func (w *Wallet) sign(fn string, ch string, args ...string) ([]string, string) {
	time.Sleep(time.Millisecond * 5)                              //nolint:gomnd
	nonce := strconv.FormatInt(time.Now().UnixNano()/1000000, 10) //nolint:gomnd
	result := append(append([]string{fn, "", ch, ch}, args...), nonce, base58.Encode(w.pKey))
	message := sha3.Sum256([]byte(strings.Join(result, "")))
	return append(result[1:], base58.Encode(ed25519.Sign(w.sKey, message[:]))), hex.EncodeToString(message[:])
}

type BatchTxResponse map[string]*proto.TxResponse

func (w *Wallet) DoBatch(ch string, txID ...string) BatchTxResponse {
	b := &proto.Batch{}
	for _, id := range txID {
		x, err := hex.DecodeString(id)
		assert.NoError(w.ledger.t, err)
		b.TxIDs = append(b.TxIDs, x)
	}
	data, err := pb.Marshal(b)
	assert.NoError(w.ledger.t, err)

	cert, err := hex.DecodeString(batchRobotCert)
	assert.NoError(w.ledger.t, err)
	w.ledger.stubs[ch].SetCreator(cert)
	res := w.Invoke(ch, "batchExecute", string(data))
	out := &proto.BatchResponse{}
	assert.NoError(w.ledger.t, pb.Unmarshal([]byte(res), out))

	result := make(BatchTxResponse)
	for _, resp := range out.TxResponses {
		if resp != nil {
			result[hex.EncodeToString(resp.Id)] = resp
		}
	}
	return result
}

func (br BatchTxResponse) TxHasNoError(t *testing.T, txID ...string) {
	for _, id := range txID {
		res, ok := br[id]
		assert.True(t, ok, "tx %s doesn't exist in batch response", id)
		if !ok {
			return
		}
		msg := ""
		if res.Error != nil {
			msg = res.Error.Error
		}
		assert.Nil(t, res.Error, msg)
	}
}

func (w *Wallet) RawSignedInvoke(ch string, fn string, args ...string) (string, TxResponse, []*proto.Swap) {
	invoke, response, swaps, _ := w.RawSignedMultiSwapInvoke(ch, fn, args...)
	return invoke, response, swaps
}

func (w *Wallet) Ledger() *Ledger {
	return w.ledger
}

func (w *Wallet) RawSignedMultiSwapInvoke(ch string, fn string, args ...string) (string, TxResponse, []*proto.Swap, []*proto.MultiSwap) {
	txID := txIDGen()
	args, _ = w.sign(fn, ch, args...)
	cert, err := base64.StdEncoding.DecodeString(userCert)
	assert.NoError(w.ledger.t, err)
	_ = w.ledger.stubs[ch].SetCreatorCert("atomyzeMSP", cert)
	w.ledger.doInvoke(ch, txID, fn, args...)

	id, err := hex.DecodeString(txID)
	assert.NoError(w.ledger.t, err)
	data, err := pb.Marshal(&proto.Batch{TxIDs: [][]byte{id}})
	assert.NoError(w.ledger.t, err)

	cert, err = hex.DecodeString(batchRobotCert)
	assert.NoError(w.ledger.t, err)
	w.ledger.stubs[ch].SetCreator(cert)
	res := w.Invoke(ch, "batchExecute", string(data))
	out := &proto.BatchResponse{}
	assert.NoError(w.ledger.t, pb.Unmarshal([]byte(res), out))

	e := <-w.ledger.stubs[ch].ChaincodeEventsChannel
	if e.EventName == "batchExecute" {
		events := &proto.BatchEvent{}
		assert.NoError(w.ledger.t, pb.Unmarshal(e.Payload, events))
		for _, ev := range events.Events {
			if hex.EncodeToString(ev.Id) == txID {
				evts := make(map[string][]byte)
				for _, evt := range ev.Events {
					evts[evt.Name] = evt.Value
				}
				er := ""
				if ev.Error != nil {
					er = ev.Error.Error
				}
				return txID, TxResponse{
					Method: ev.Method,
					Error:  er,
					Result: string(ev.Result),
					Events: evts,
				}, out.CreatedSwaps, out.CreatedMultiSwap
			}
		}
	}
	assert.Fail(w.ledger.t, shouldNotBeHereMsg)
	return txID, TxResponse{}, out.CreatedSwaps, out.CreatedMultiSwap
}

func (w *Wallet) RawSignedInvokeWithErrorReturned(ch string, fn string, args ...string) error {
	txID := txIDGen()
	args, _ = w.sign(fn, ch, args...)
	cert, err := base64.StdEncoding.DecodeString(userCert)
	assert.NoError(w.ledger.t, err)
	_ = w.ledger.stubs[ch].SetCreatorCert("atomyzeMSP", cert)
	err = w.ledger.doInvokeWithErrorReturned(ch, txID, fn, args...)
	if err != nil {
		return err
	}

	id, err := hex.DecodeString(txID)
	if err != nil {
		return err
	}
	data, err := pb.Marshal(&proto.Batch{TxIDs: [][]byte{id}})
	if err != nil {
		return err
	}

	cert, err = hex.DecodeString(batchRobotCert)
	if err != nil {
		return err
	}
	w.ledger.stubs[ch].SetCreator(cert)
	res := w.Invoke(ch, "batchExecute", string(data))
	out := &proto.BatchResponse{}
	err = pb.Unmarshal([]byte(res), out)
	if err != nil {
		return err
	}

	e := <-w.ledger.stubs[ch].ChaincodeEventsChannel
	if e.EventName == "batchExecute" {
		events := &proto.BatchEvent{}
		err = pb.Unmarshal(e.Payload, events)
		if err != nil {
			return err
		}
		for _, ev := range events.Events {
			if hex.EncodeToString(ev.Id) == txID {
				evts := make(map[string][]byte)
				for _, evt := range ev.Events {
					evts[evt.Name] = evt.Value
				}
				if ev.Error != nil {
					return errors.New(ev.Error.Error)
				}
				return nil
			}
		}
	}
	assert.Fail(w.ledger.t, shouldNotBeHereMsg)
	return nil
}

func (w *Wallet) SignedInvoke(ch string, fn string, args ...string) string {
	txID, res, swaps := w.RawSignedInvoke(ch, fn, args...)
	assert.Equal(w.ledger.t, "", res.Error)
	for _, swap := range swaps {
		x := proto.Batch{Swaps: []*proto.Swap{{
			Id:      swap.Id,
			Creator: []byte("0000"),
			Owner:   swap.Owner,
			Token:   swap.Token,
			Amount:  swap.Amount,
			From:    swap.From,
			To:      swap.To,
			Hash:    swap.Hash,
			Timeout: swap.Timeout,
		}}}
		data, err := pb.Marshal(&x)
		assert.NoError(w.ledger.t, err)
		cert, err := hex.DecodeString(batchRobotCert)
		assert.NoError(w.ledger.t, err)
		w.ledger.stubs[strings.ToLower(swap.To)].SetCreator(cert)
		w.Invoke(strings.ToLower(swap.To), "batchExecute", string(data))
	}
	return txID
}

func (w *Wallet) SignedMultiSwapsInvoke(ch string, fn string, args ...string) string {
	txID, res, _, multiSwaps := w.RawSignedMultiSwapInvoke(ch, fn, args...)
	assert.Equal(w.ledger.t, "", res.Error)
	for _, swap := range multiSwaps {
		x := proto.Batch{
			MultiSwaps: []*proto.MultiSwap{
				{
					Id:      swap.Id,
					Creator: []byte("0000"),
					Owner:   swap.Owner,
					Token:   swap.Token,
					Assets:  swap.Assets,
					From:    swap.From,
					To:      swap.To,
					Hash:    swap.Hash,
					Timeout: swap.Timeout,
				},
			},
		}
		data, err := pb.Marshal(&x)
		assert.NoError(w.ledger.t, err)
		cert, err := hex.DecodeString(batchRobotCert)
		assert.NoError(w.ledger.t, err)
		w.ledger.stubs[swap.To].SetCreator(cert)
		w.Invoke(swap.To, "batchExecute", string(data))
	}
	return txID
}

func (w *Wallet) OtfNbInvoke(ch string, fn string, args ...string) (string, string) {
	txID := txIDGen()
	message, hash := w.sign(fn, ch, args...)
	cert, err := base64.StdEncoding.DecodeString(userCert)
	assert.NoError(w.ledger.t, err)
	_ = w.ledger.stubs[ch].SetCreatorCert("atomyzeMSP", cert)
	w.ledger.doInvoke(ch, txID, fn, message...)

	nested, err := pb.Marshal(&proto.Nested{Args: append([]string{fn}, message...)})
	assert.NoError(w.ledger.t, err)

	return base58.Encode(nested), hash
}

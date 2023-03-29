package token

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"reflect"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/btcsuite/btcutil/base58"
	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
	pb "github.com/tickets-dao/foundation/v3/proto"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"
)

const (
	pkPrefix      = "pk"
	addressPrefix = "address"
	accInfoPrefix = "accountinfo"
	noncePrefix   = "nonce"
)

type AddrsWithPagination struct {
	Addrs    []string
	Bookmark string
}

type (
	initArgs struct {
		AdminSKI        []byte
		ValidatorsCount int
		Validators      []string
	}
	ACL struct {
		allowedMspId string
		init         *initArgs
	}
	ccfunc func(stub shim.ChaincodeStubInterface, args []string) peer.Response
)

func New(allowedMspId string) *ACL {
	return &ACL{allowedMspId: allowedMspId}
}

func (c *ACL) Init(stub shim.ChaincodeStubInterface) peer.Response {
	creator, err := stub.GetCreator()
	if err != nil {
		return shim.Error(err.Error())
	}
	var identity msp.SerializedIdentity
	if err := proto.Unmarshal(creator, &identity); err != nil {
		return shim.Error(err.Error())
	}
	if identity.Mspid != c.allowedMspId {
		fmt.Println("XX", identity.Mspid, c.allowedMspId, identity, len(creator))
		return shim.Error("incorrect MSP Id")
	}
	b, _ := pem.Decode(identity.IdBytes)
	parsed, err := x509.ParseCertificate(b.Bytes)
	if err != nil {
		return shim.Error(err.Error())
	}
	ouIsOk := false
	for _, ou := range parsed.Subject.OrganizationalUnit {
		if strings.ToLower(ou) == "admin" {
			ouIsOk = true
		}
	}
	if !ouIsOk {
		return shim.Error("incorrect sender's OU")
	}

	args := stub.GetStringArgs()
	if len(args) < 2 {
		return shim.Error("arguments should be at least 2")
	}
	adminSKI, err := hex.DecodeString(args[0])
	if err != nil {
		return shim.Error(err.Error())
	}
	validatorsCount, err := strconv.Atoi(args[1])
	if err != nil {
		return shim.Error(err.Error())
	}
	var data bytes.Buffer
	if err := gob.NewEncoder(&data).Encode(initArgs{
		AdminSKI:        adminSKI,
		ValidatorsCount: validatorsCount,
		Validators:      args[2:],
	}); err != nil {
		return shim.Error(err.Error())
	}
	if err := stub.PutState("__init", data.Bytes()); err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}

type Account struct {
	Address string   `json:"address"`
	Balance *big.Int `json:"balance"`
}

func (c *ACL) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("panic invoke\n" + string(debug.Stack()))
		}
	}()
	fn, args := stub.GetFunctionAndParameters()
	if c.init == nil {
		data, err := stub.GetState("__init")
		if err != nil {
			return shim.Error(err.Error())
		}
		if data == nil {
			return shim.Error("ACL chaincode not initialized, please invoke Init with init args first")
		}
		init := &initArgs{}
		if err := gob.NewDecoder(bytes.NewReader(data)).Decode(init); err != nil {
			return shim.Error(err.Error())
		}
		c.init = init
	}
	methods := make(map[string]ccfunc)
	t := reflect.TypeOf(c)
	var ok bool
	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)
		if method.Name != "Init" && method.Name != "Invoke" {
			name := toLowerFirstLetter(method.Name)
			if methods[name], ok = reflect.ValueOf(c).MethodByName(method.Name).Interface().(func(shim.ChaincodeStubInterface, []string) peer.Response); !ok {
				return shim.Error(fmt.Sprintf("Chaincode initialization failure: cc method %s does not satisfy signature func(stub shim.ChaincodeStubInterface, args []string) peer.Response", method.Name))
			}
		}
	}

	if ccinvoke, ok := methods[fn]; !ok {
		return shim.Error("unknown method")
	} else {
		return ccinvoke(stub, args)
	}
}

func (c *ACL) AddUser(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	argsNum := len(args)
	if argsNum < 4 {
		return shim.Error(fmt.Sprintf("incorrect number of arguments: %d, but this method expects: public key, KYC hash, user ID, industrial attribute ('true' or 'false')", argsNum))
	}

	if err := c.checkCert(stub); err != nil {
		return shim.Error(fmt.Sprintf("unauthorized: %s", err.Error()))
	}

	encodedBase58PublicKey := args[0]
	kycHash := args[1]
	userId := args[2]
	isIndustrial := args[3] == "true"

	decodedPublicKey, err := decodeBase58PublicKey(encodedBase58PublicKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(kycHash) == 0 {
		return shim.Error("empty kyc hash")
	}
	if len(userId) == 0 {
		return shim.Error("empty userId")
	}

	hashed := sha3.Sum256(decodedPublicKey)
	pkeys := hex.EncodeToString(hashed[:])
	addr := base58.CheckEncode(hashed[1:], hashed[0])
	pkToAddrCompositeKey, err := stub.CreateCompositeKey(addressPrefix, []string{pkeys})
	if err != nil {
		return shim.Error(err.Error())
	}

	addrAlreadyInLedgerBytes, err := stub.GetState(pkToAddrCompositeKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(addrAlreadyInLedgerBytes) != 0 {
		addrAlreadyInLedger := &pb.SignedAddress{}
		err = proto.Unmarshal(addrAlreadyInLedgerBytes, addrAlreadyInLedger)
		if err != nil {
			return shim.Error(err.Error())
		}
		return shim.Error(fmt.Sprintf("The address %s associated with key %s already exists", addrAlreadyInLedger.Address.AddrString(), pkeys))
	}

	addrToPkCompositeKey, err := stub.CreateCompositeKey(pkPrefix, []string{addr})
	if err != nil {
		return shim.Error(err.Error())
	}

	addrMsg, err := proto.Marshal(&pb.SignedAddress{Address: &pb.Address{
		UserID:       userId,
		Address:      hashed[:],
		IsIndustrial: isIndustrial,
		IsMultisig:   false,
	}})
	if err != nil {
		return shim.Error(err.Error())
	}

	if err := stub.PutState(pkToAddrCompositeKey, addrMsg); err != nil {
		return shim.Error(err.Error())
	}

	// save address -> pubkey hash mapping
	if err := stub.PutState(addrToPkCompositeKey, []byte(pkeys)); err != nil {
		return shim.Error(err.Error())
	}

	infoMsg, err := proto.Marshal(&pb.AccountInfo{KycHash: kycHash})
	if err != nil {
		return shim.Error(err.Error())
	}

	ckey, err := stub.CreateCompositeKey(accInfoPrefix, []string{addr})
	if err != nil {
		return shim.Error(err.Error())
	}
	if err := stub.PutState(ckey, infoMsg); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// AddMultisig creates multisignature address which operates when N of M signatures is present
// arg[0] - N number of signature policy (number of sufficient signatures), M part is derived from number of pubkeys
// arg[1] - nonce
// other arguments are the public keys and signatures of all participants in the multi-wallet
// and signatures confirming the agreement of all participants with the signature policy
func (c *ACL) AddMultisig(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	argsNum := len(args)
	if argsNum < 4 {
		return shim.Error(fmt.Sprintf("incorrect number of arguments: %d, but this method expects: address, N (signatures required), nonce, public keys, signatures", argsNum))
	}

	if err := c.checkCert(stub); err != nil {
		return shim.Error(fmt.Sprintf("unauthorized: %s", err.Error()))
	}

	N, err := strconv.Atoi(args[0])
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to parse N, error: %s", err.Error()))
	}
	nonce := args[1]
	PksAndSignatures := args[2:]

	pks := PksAndSignatures[:len(PksAndSignatures)/2]
	signatures := PksAndSignatures[len(PksAndSignatures)/2:]

	// check all members signed
	if len(pks) != len(signatures) {
		return shim.Error(fmt.Sprintf("the number of signatures (%d) does not match the number of public keys (%d)", len(signatures), len(pks)))
	}

	pksNumber := len(pks)
	signaturesNumber := len(signatures)
	// number of pks should be equal to number of signatures
	if pksNumber != signaturesNumber {
		return shim.Error(fmt.Sprintf("multisig signature policy can't be created, number of public keys (%d) does not match number of signatures (%d)", pksNumber, signaturesNumber))
	}
	// N shouldn't be greater then number of public keys (M part of signature policy)
	if N > pksNumber {
		return shim.Error(fmt.Sprintf("N (%d) is greater then M (number of pubkeys, %d)", N, pksNumber))
	}

	message := sha3.Sum256([]byte(strings.Join(append([]string{"addMultisig", args[0], args[1]}, pks...), "")))

	for _, pk := range pks {
		// check the presence of multisig members in the black and gray list
		if err := checkBlocked(stub, pk); err != nil {
			return shim.Error(err.Error())
		}
	}

	if err := checkKeysArr(pks); err != nil {
		return shim.Error(fmt.Sprintf("%s, input: '%v'", err.Error(), pks))
	}
	hashedHexKeys, err := keyStringToSortedHashedHex(pks)
	if err != nil {
		return shim.Error(fmt.Sprintf("%s, input: '%s'", err.Error(), args[3]))
	}
	var pksDecodedOriginalOrder [][]byte

	for _, encodedBase58PublicKey := range pks {
		decodedPublicKey, err := decodeBase58PublicKey(encodedBase58PublicKey)
		if err != nil {
			return shim.Error(err.Error())
		}
		pksDecodedOriginalOrder = append(pksDecodedOriginalOrder, decodedPublicKey)
	}

	// derive address from hash of sorted base58-(DE)coded pubkeys
	keysArrSorted, err := DecodeAndSort(strings.Join(pks, "/"))
	if err != nil {
		return shim.Error(err.Error())
	}
	hashedPksSortedOrder := sha3.Sum256(bytes.Join(keysArrSorted, []byte("")))
	addr := base58.CheckEncode(hashedPksSortedOrder[1:], hashedPksSortedOrder[0])

	if err := checkNonce(stub, addr, nonce); err != nil {
		return shim.Error(err.Error())
	}

	if err := checkNOutMSigned(len(pksDecodedOriginalOrder), message[:], pksDecodedOriginalOrder, signatures); err != nil {
		return shim.Error(err.Error())
	}

	// check multisig address doesn't already exist
	pkToAddrCompositeKey, err := stub.CreateCompositeKey(addressPrefix, []string{hashedHexKeys})
	if err != nil {
		return shim.Error(err.Error())
	}

	addrAlreadyInLedgerBytes, err := stub.GetState(pkToAddrCompositeKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	addrAlreadyInLedger := &pb.SignedAddress{}
	err = proto.Unmarshal(addrAlreadyInLedgerBytes, addrAlreadyInLedger)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(addrAlreadyInLedgerBytes) != 0 {
		return shim.Error(fmt.Sprintf("The address %s associated with key %s already exists", addrAlreadyInLedger.Address.AddrString(), hashedHexKeys))
	}

	addrToPkCompositeKey, err := stub.CreateCompositeKey(pkPrefix, []string{addr})
	if err != nil {
		return shim.Error(err.Error())
	}

	var pksDecodedOrigOrder [][]byte
	for _, encodedBase58PublicKey := range pks {
		decodedPublicKey, err := decodeBase58PublicKey(encodedBase58PublicKey)
		if err != nil {
			return shim.Error(err.Error())
		}
		pksDecodedOrigOrder = append(pksDecodedOrigOrder, decodedPublicKey)
	}

	signedAddr, err := proto.Marshal(&pb.SignedAddress{
		Address: &pb.Address{
			UserID:       "",
			Address:      hashedPksSortedOrder[:],
			IsIndustrial: false,
			IsMultisig:   true,
		},
		SignedTx: append(append(append([]string{"addMultisig"}, args[0:2]...), pks...), signatures...),
		SignaturePolicy: &pb.SignaturePolicy{
			N:       uint32(N),
			PubKeys: pksDecodedOrigOrder,
		},
	})
	if err != nil {
		return shim.Error(err.Error())
	}

	// save multisig pk -> addr mapping
	if err := stub.PutState(pkToAddrCompositeKey, signedAddr); err != nil {
		return shim.Error(err.Error())
	}

	// save multisig address -> pk mapping
	if err := stub.PutState(addrToPkCompositeKey, []byte(hashedHexKeys)); err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}

// AddToList sets address to graylist or blacklist
// arg[0] - address
// arg[1] - "gray" of "black"
func (c *ACL) AddToList(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	argsNum := len(args)
	if argsNum < 2 {
		return shim.Error(fmt.Sprintf("incorrect number of arguments: %d, but this method expects: address, attribute ('gray' or 'black')", argsNum))
	}

	if err := c.checkCert(stub); err != nil {
		return shim.Error(fmt.Sprintf("unauthorized: %s", err.Error()))
	}

	if len(args[0]) == 0 {
		return shim.Error("empty address")
	}

	addrArg := args[0]
	if args[1] != "gray" && args[1] != "black" {
		return shim.Error("marker not specified (black or gray list)")
	}

	color := args[1]

	if err := changeListStatus(stub, addrArg, color, true); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// DelFromList removes address from graylist or blacklist
// arg[0] - address
// arg[1] - "gray" of "black"
func (c *ACL) DelFromList(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	argsNum := len(args)
	if argsNum < 2 {
		return shim.Error(fmt.Sprintf("incorrect number of arguments: %d, but this method expects: address, attribute ('gray' or 'black')", argsNum))
	}

	if err := c.checkCert(stub); err != nil {
		return shim.Error(fmt.Sprintf("unauthorized: %s", err.Error()))
	}

	if len(args[0]) == 0 {
		return shim.Error("empty pub key")
	}

	addrArg := args[0]
	if args[1] != "gray" && args[1] != "black" {
		return shim.Error("marker not specified (black or white list)")
	}

	color := args[1]

	if err := changeListStatus(stub, addrArg, color, false); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func changeListStatus(stub shim.ChaincodeStubInterface, addr, color string, status bool) error {
	ckey, err := stub.CreateCompositeKey(accInfoPrefix, []string{addr})
	if err != nil {
		return err
	}
	info, err := getAccountInfo(stub, addr)
	if err != nil {
		return err
	}

	switch color {
	case "gray":
		info.GrayListed = status
	case "black":
		info.BlackListed = status
	}

	infoMarshaled, err := proto.Marshal(info)
	if err != nil {
		return err
	}

	return stub.PutState(ckey, infoMarshaled)
}

func (c *ACL) CheckKeys(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	argsNum := len(args)
	if argsNum < 1 {
		return shim.Error(fmt.Sprintf("incorrect number of arguments: %d, but this method expects: N pubkeys", argsNum))
	}

	if len(args[0]) == 0 {
		return shim.Error("empty pub keys")
	}

	strKeys := strings.Split(args[0], "/")
	if err := checkKeysArr(strKeys); err != nil {
		return shim.Error(fmt.Sprintf("%s, input: '%s'", err.Error(), args[0]))
	}
	pkeys, err := keyStringToSortedHashedHex(strKeys)
	if err != nil {
		return shim.Error(fmt.Sprintf("%s, input: '%s'", err.Error(), args[3]))
	}

	addr, err := getAddressByHashedKeys(stub, pkeys)
	if err != nil {
		return shim.Error(err.Error())
	}

	var info *pb.AccountInfo
	if len(strKeys) == 1 {
		info, err = getAccountInfo(stub, addr.Address.AddrString())
		if err != nil {
			return shim.Error(err.Error())
		}
	} else {
		// for multi keys
		info = &pb.AccountInfo{}
		for _, key := range strKeys {
			strKeys := strings.Split(key, "/")
			if err := checkKeysArr(strKeys); err != nil {
				return shim.Error(fmt.Sprintf("%s, input: '%s'", err.Error(), key))
			}
			pkeys, err := keyStringToSortedHashedHex(strKeys)
			if err != nil {
				return shim.Error(fmt.Sprintf("%s, input: '%s'", err.Error(), args[3]))
			}
			addr, err := getAddressByHashedKeys(stub, pkeys)
			if err != nil {
				return shim.Error(err.Error())
			}
			info, err = getAccountInfo(stub, addr.Address.AddrString())
			if err != nil {
				return shim.Error(err.Error())
			}
			if info.GrayListed {
				info = &pb.AccountInfo{GrayListed: true}
				break
			}
			if info.BlackListed {
				info = &pb.AccountInfo{BlackListed: true}
				break
			}
		}
	}

	result, err := proto.Marshal(&pb.AclResponse{
		Account: info,
		Address: addr,
	})
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(result)
}

func checkBlocked(stub shim.ChaincodeStubInterface, encodedBase58PublicKey string) error {
	decodedPublicKey, err := decodeBase58PublicKey(encodedBase58PublicKey)
	if err != nil {
		return err
	}
	hashed := sha3.Sum256(decodedPublicKey)
	pkeys := hex.EncodeToString(hashed[:])

	pkToAddrCompositeKey, err := stub.CreateCompositeKey(addressPrefix, []string{pkeys})
	if err != nil {
		return err
	}

	keyData, err := stub.GetState(pkToAddrCompositeKey)
	if err != nil {
		return err
	}
	if len(keyData) == 0 {
		return errors.New("not found any records")
	}
	var a pb.SignedAddress
	if err := proto.Unmarshal(keyData, &a); err != nil {
		return err
	}

	var info *pb.AccountInfo
	info, err = getAccountInfo(stub, a.Address.AddrString())
	if err != nil {
		return err
	}

	if info.BlackListed {
		return errors.New(fmt.Sprintf("address %s is blacklisted", a.Address.AddrString()))
	}
	if info.GrayListed {
		return errors.New(fmt.Sprintf("address %s is graylisted", a.Address.AddrString()))
	}

	return nil
}

// CheckAddress checks if the address is graylisted
// returns an error if the address is graylisted or returns pb.Address if not
// args[0] - base58-encoded address
func (c *ACL) CheckAddress(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	argsNum := len(args)
	if argsNum < 1 {
		return shim.Error(fmt.Sprintf("incorrect number of arguments: %d, but this method expects: address", argsNum))
	}

	addrEncoded := args[0]
	if len(addrEncoded) == 0 {
		return shim.Error("empty address")
	}

	addrToPkCompositeKey, err := stub.CreateCompositeKey(pkPrefix, []string{addrEncoded})
	if err != nil {
		return shim.Error(err.Error())
	}

	// check the pubkey hash exists in ACL
	keys, err := stub.GetState(addrToPkCompositeKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(keys) == 0 {
		return shim.Error(fmt.Sprintf("no pub keys for address %s", addrEncoded))
	}

	if err = checkGraylist(stub, addrEncoded); err != nil {
		return shim.Error(err.Error())
	}

	// get pb.SignedAddress
	pkToAddrCompositeKey, err := stub.CreateCompositeKey(addressPrefix, []string{string(keys)})
	if err != nil {
		return shim.Error(err.Error())
	}

	addrProto, err := stub.GetState(pkToAddrCompositeKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(addrProto) == 0 {
		return shim.Error("no such address in the ledger")
	}

	signedAddr := &pb.SignedAddress{}
	if err := proto.Unmarshal(addrProto, signedAddr); err != nil {
		return shim.Error(err.Error())
	}

	// prepare and return pb.Address only (extracted from pb.SignedAddress)
	addrResponse, err := proto.Marshal(signedAddr.Address)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(addrResponse)
}

// Setkyc updates KYC for address.
// arg[0] - address
// arg[1] - KYC hash
// arg[2] - nonce
// arg[3:] - public keys and signatures of validators
func (c *ACL) Setkyc(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	argsNum := len(args)
	if argsNum < 5 {
		return shim.Error(fmt.Sprintf("incorrect number of arguments: %d, but this method expects: address, nonce, KYC hash, public keys, signatures", argsNum))
	}

	if err := c.checkCert(stub); err != nil {
		return shim.Error(fmt.Sprintf("unauthorized: %s", err.Error()))
	}

	if len(args[0]) == 0 {
		return shim.Error("empty address")
	}
	if len(args[1]) == 0 {
		return shim.Error("empty KYC hash string")
	}
	if len(args[2]) == 0 {
		return shim.Error("empty nonce")
	}
	if len(args[3:]) == 0 {
		return shim.Error("no public keys and signatures provided")
	}
	address := args[0]
	newKyc := args[1]
	nonce := args[2]
	PksAndSignatures := args[3:]
	pks := PksAndSignatures[:len(PksAndSignatures)/2]
	signatures := PksAndSignatures[len(PksAndSignatures)/2:]
	message := sha3.Sum256([]byte(strings.Join(append([]string{"setkyc", address, newKyc, nonce}, pks...), "")))

	if err := checkNonce(stub, address, nonce); err != nil {
		return shim.Error(err.Error())
	}

	if err := c.checkValidatorsSigned(message[:], pks, signatures); err != nil {
		return shim.Error(err.Error())
	}

	ckey, err := stub.CreateCompositeKey(accInfoPrefix, []string{address})
	if err != nil {
		return shim.Error(err.Error())
	}
	infoData, err := stub.GetState(ckey)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(infoData) == 0 {
		return shim.Error(fmt.Sprintf("Account info for address %s is empty", address))
	}

	var info pb.AccountInfo
	if err := proto.Unmarshal(infoData, &info); err != nil {
		return shim.Error(err.Error())
	}

	info.KycHash = newKyc

	newAccInfo, err := proto.Marshal(&info)
	if err != nil {
		return shim.Error(err.Error())
	}

	if err := stub.PutState(ckey, newAccInfo); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// GetAccountInfo returns json-serialized account info (KYC hash, graylist and blacklist attributes) for address.
// arg[0] - address
func (c *ACL) GetAccountInfo(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	addrEncoded := args[0]
	if len(addrEncoded) == 0 {
		return shim.Error("empty address")
	}
	accInfo, err := getAccountInfo(stub, addrEncoded)
	if err != nil {
		return shim.Error(err.Error())
	}
	accInfoSerilized, err := json.Marshal(accInfo)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(accInfoSerilized)
}

func getAccountInfo(stub shim.ChaincodeStubInterface, address string) (*pb.AccountInfo, error) {
	ckey, err := stub.CreateCompositeKey(accInfoPrefix, []string{address})
	if err != nil {
		return nil, err
	}
	infoData, err := stub.GetState(ckey)
	if err != nil {
		return nil, err
	}
	if len(infoData) == 0 {
		return nil, fmt.Errorf("no such address in ACL: %s", address)
	}

	var info pb.AccountInfo
	if err := proto.Unmarshal(infoData, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func (c *ACL) GetAddresses(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	argsNum := len(args)
	if argsNum < 2 {
		return shim.Error(fmt.Sprintf("incorrect number of arguments: %d, but this method expects: pagesize, bookmark", argsNum))
	}
	pageSize := args[0]
	bookmark := args[1]
	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil {
		return shim.Error(err.Error())
	}
	iterator, result, err := stub.GetStateByPartialCompositeKeyWithPagination(pkPrefix, []string{}, int32(pageSizeInt), bookmark) // we use addr -> pk mapping here
	if err != nil {
		return shim.Error(err.Error())
	}
	defer iterator.Close()

	var addrs []string
	for iterator.HasNext() {
		kv, err := iterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		_, extractedAddr, err := stub.SplitCompositeKey(kv.Key)
		if err != nil {
			return shim.Error(err.Error())
		}
		addrs = append(addrs, extractedAddr[0])
	}

	serialized, err := json.Marshal(AddrsWithPagination{
		Addrs:    addrs,
		Bookmark: result.Bookmark,
	})
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(serialized)
}

// SetAccountInfo sets account info (KYC hash, graylist and blacklist attributes) for address.
// arg[0] - address
// arg[1] - KYC hash
// arg[2] - is address graylisted? ("true" or "false")
// arg[3] - is address blacklisted? ("true" or "false")
func (c *ACL) SetAccountInfo(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	addrEncoded := args[0]
	if len(addrEncoded) == 0 {
		return shim.Error("empty address")
	}

	kycHash := args[1]

	graylisted := args[2]
	if len(graylisted) == 0 {
		return shim.Error("graylist attribute is not set")
	}

	isGraylisted, err := strconv.ParseBool(graylisted)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to parse graylist attribute, %s", err))
	}

	blacklisted := args[3]
	if len(blacklisted) == 0 {
		return shim.Error("blacklist attribute is not set")
	}

	isBlacklisted, err := strconv.ParseBool(blacklisted)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to parse blacklist attribute, %s", err))
	}

	if err := c.checkCert(stub); err != nil {
		return shim.Error(fmt.Sprintf("unauthorized: %s", err.Error()))
	}

	infoMsg, err := proto.Marshal(&pb.AccountInfo{KycHash: kycHash, GrayListed: isGraylisted, BlackListed: isBlacklisted})
	if err != nil {
		return shim.Error(err.Error())
	}

	ckey, err := stub.CreateCompositeKey(accInfoPrefix, []string{addrEncoded})
	if err != nil {
		return shim.Error(err.Error())
	}
	if err := stub.PutState(ckey, infoMsg); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func checkGraylist(stub shim.ChaincodeStubInterface, addrEncoded string) error {
	accInfoKey, err := stub.CreateCompositeKey(accInfoPrefix, []string{addrEncoded})
	if err != nil {
		return err
	}

	accInfo, err := stub.GetState(accInfoKey)
	if err != nil {
		return err
	}

	var info pb.AccountInfo
	if err := proto.Unmarshal(accInfo, &info); err != nil {
		return err
	}

	if info.GrayListed {
		return fmt.Errorf("address %s is graylisted", addrEncoded)
	}
	return nil
}

// ChangePublicKeyWithBase58Signature changes public key of user
// arg[0] - user's address (base58check)
// arg[1] - reason (string)А
// arg[2] - reason ID (string)
// arg[3] - new key (base58)
// arg[4] - nonce
// arg[5:] - public keys and signatures of validators
func (c *ACL) ChangePublicKeyWithBase58Signature(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	fn, _ := stub.GetFunctionAndParameters()
	argsNum := len(args)
	if argsNum < 10 {
		return shim.Error(fmt.Sprintf("incorrect number of arguments: %d, but this method expects: address, reason, reason ID, new key, nonce, public keys, signatures", argsNum))
	}

	if err := c.checkCert(stub); err != nil {
		return shim.Error(fmt.Sprintf("unauthorized: %s", err.Error()))
	}

	// args[0] is request id
	// requestId := args[0]

	chaincodeName := args[1]
	if chaincodeName != "acl" {
		return shim.Error("incorrect chaincode name")
	}

	channelID := args[2]
	if channelID != stub.GetChannelID() {
		return shim.Error("incorrect channel")
	}

	forAddrOrig := args[3]
	if len(forAddrOrig) == 0 {
		return shim.Error("empty address")
	}
	reason := args[4]
	if len(reason) == 0 {
		return shim.Error("reason not provided")
	}

	if len(args[5]) == 0 {
		return shim.Error("reason ID not provided")
	}
	reasonId, err := strconv.Atoi(args[5])
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to convert reason ID to int, err: %s", err.Error()))
	}

	if len(args[6]) == 0 {
		return shim.Error("empty new key")
	}

	strKeys := strings.Split(args[6], "/")
	if err := checkKeysArr(strKeys); err != nil {
		return shim.Error(fmt.Sprintf("%s, input: '%s'", err.Error(), args[3]))
	}
	newkey, err := keyStringToSortedHashedHex(strKeys)
	if err != nil {
		return shim.Error(fmt.Sprintf("%s, input: '%s'", err.Error(), args[3]))
	}

	nonce := args[7]
	if len(nonce) == 0 {
		return shim.Error("empty nonce")
	}

	pksAndSignatures := args[8:]
	if len(pksAndSignatures) == 0 {
		return shim.Error("no public keys and signatures provided")
	}
	validatorCount := len(pksAndSignatures) / 2
	pks := pksAndSignatures[:validatorCount]
	signatures := pksAndSignatures[validatorCount:]

	message := sha3.Sum256([]byte(fn + strings.Join(args[:8+validatorCount], "")))

	if err := checkNonce(stub, forAddrOrig, nonce); err != nil {
		return shim.Error(err.Error())
	}

	if err := c.checkValidatorsSignedWithBase58Signature(message[:], pks, signatures); err != nil {
		return shim.Error(err.Error())
	}

	addrToPkCompositeKey, err := stub.CreateCompositeKey(pkPrefix, []string{forAddrOrig})
	if err != nil {
		return shim.Error(err.Error())
	}

	// check that we have pub key for such address
	keys, err := stub.GetState(addrToPkCompositeKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(keys) == 0 {
		return shim.Error(fmt.Sprintf("no pub keys for address %s", forAddrOrig))
	}

	// del old pub key -> pb.Address mapping
	pkToAddrCompositeKey, err := stub.CreateCompositeKey(addressPrefix, []string{string(keys)})
	if err != nil {
		return shim.Error(err.Error())
	}
	// firstly get pb.SignedAddress to re-create it later in new mapping
	signedAddrBytes, err := stub.GetState(pkToAddrCompositeKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(signedAddrBytes) == 0 {
		return shim.Error(fmt.Sprintf("no SignedAddress msg for address %s", forAddrOrig))
	}
	signedAddr := &pb.SignedAddress{}
	if err := proto.Unmarshal(signedAddrBytes, signedAddr); err != nil {
		return shim.Error(err.Error())
	}

	// and delete
	err = stub.DelState(pkToAddrCompositeKey)
	if err != nil {
		return shim.Error(err.Error())
	}

	// del old addr -> pub key mapping
	err = stub.DelState(addrToPkCompositeKey)
	if err != nil {
		return shim.Error(err.Error())
	}

	// set new key -> pb.SignedAddress mapping
	newPkToAddrCompositeKey, err := stub.CreateCompositeKey(addressPrefix, []string{newkey})
	if err != nil {
		return shim.Error(err.Error())
	}

	signedAddr.SignedTx = append(append(append([]string{"changePublicKeyWithBase58Signature"}, args[0:5]...), pks...), signatures...)
	signedAddr.Reason = reason
	signedAddr.ReasonId = int32(reasonId)
	addrChangeMsg, err := proto.Marshal(signedAddr)
	if err != nil {
		return shim.Error(err.Error())
	}

	if err := stub.PutState(newPkToAddrCompositeKey, addrChangeMsg); err != nil {
		return shim.Error(err.Error())
	}

	// set new address -> key mapping
	if err := stub.PutState(addrToPkCompositeKey, []byte(newkey)); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// ChangePublicKey changes public key of user
// arg[0] - user's address (base58check)
// arg[1] - reason (string)А
// arg[2] - reason ID (string)
// arg[3] - new key (base58)
// arg[4] - nonce
// arg[5:] - public keys and signatures of validators
func (c *ACL) ChangePublicKey(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	argsNum := len(args)
	if argsNum < 7 {
		return shim.Error(fmt.Sprintf("incorrect number of arguments: %d, but this method expects: address, reason, reason ID, new key, nonce, public keys, signatures", argsNum))
	}

	if err := c.checkCert(stub); err != nil {
		return shim.Error(fmt.Sprintf("unauthorized: %s", err.Error()))
	}

	if len(args[0]) == 0 {
		return shim.Error("empty address")
	}
	if len(args[1]) == 0 {
		return shim.Error("reason not provided")
	}
	if len(args[2]) == 0 {
		return shim.Error("reason ID not provided")
	}
	if len(args[3]) == 0 {
		return shim.Error("empty new key")
	}
	if len(args[4]) == 0 {
		return shim.Error("empty nonce")
	}
	if len(args[5:]) == 0 {
		return shim.Error("no public keys and signatures provided")
	}

	forAddrOrig := args[0]
	reason := args[1]
	reasonId, err := strconv.Atoi(args[2])
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to convert reason ID to int, err: %s", err.Error()))
	}

	strKeys := strings.Split(args[3], "/")
	if err := checkKeysArr(strKeys); err != nil {
		return shim.Error(fmt.Sprintf("%s, input: '%s'", err.Error(), args[3]))
	}
	newkey, err := keyStringToSortedHashedHex(strKeys)
	if err != nil {
		return shim.Error(fmt.Sprintf("%s, input: '%s'", err.Error(), args[3]))
	}

	nonce := args[4]
	PksAndSignatures := args[5:]
	pks := PksAndSignatures[:len(PksAndSignatures)/2]
	signatures := PksAndSignatures[len(PksAndSignatures)/2:]

	// check all members signed
	if len(pks) != len(signatures) {
		return shim.Error(fmt.Sprintf("the number of signatures (%d) does not match the number of public keys (%d)", len(signatures), len(pks)))
	}

	message := sha3.Sum256([]byte(strings.Join(append([]string{"changePublicKey", forAddrOrig, reason, args[2], args[3], nonce}, pks...), "")))

	if err := checkNonce(stub, forAddrOrig, nonce); err != nil {
		return shim.Error(err.Error())
	}

	if err := c.checkValidatorsSigned(message[:], pks, signatures); err != nil {
		return shim.Error(err.Error())
	}

	addrToPkCompositeKey, err := stub.CreateCompositeKey(pkPrefix, []string{forAddrOrig})
	if err != nil {
		return shim.Error(err.Error())
	}

	// check that we have pub key for such address
	keys, err := stub.GetState(addrToPkCompositeKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(keys) == 0 {
		return shim.Error(fmt.Sprintf("no pub keys for address %s", forAddrOrig))
	}

	// del old pub key -> pb.Address mapping
	pkToAddrCompositeKey, err := stub.CreateCompositeKey(addressPrefix, []string{string(keys)})
	if err != nil {
		return shim.Error(err.Error())
	}
	// firstly get pb.SignedAddress to re-create it later in new mapping
	signedAddrBytes, err := stub.GetState(pkToAddrCompositeKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(signedAddrBytes) == 0 {
		return shim.Error(fmt.Sprintf("no SignedAddress msg for address %s", forAddrOrig))
	}
	signedAddr := &pb.SignedAddress{}
	if err := proto.Unmarshal(signedAddrBytes, signedAddr); err != nil {
		return shim.Error(err.Error())
	}

	// and delete
	err = stub.DelState(pkToAddrCompositeKey)
	if err != nil {
		return shim.Error(err.Error())
	}

	// del old addr -> pub key mapping
	err = stub.DelState(addrToPkCompositeKey)
	if err != nil {
		return shim.Error(err.Error())
	}

	// set new key -> pb.SignedAddress mapping
	newPkToAddrCompositeKey, err := stub.CreateCompositeKey(addressPrefix, []string{newkey})
	if err != nil {
		return shim.Error(err.Error())
	}

	signedAddr.SignedTx = append(append(append([]string{"changePublicKey"}, args[0:5]...), pks...), signatures...)
	signedAddr.Reason = reason
	signedAddr.ReasonId = int32(reasonId)
	addrChangeMsg, err := proto.Marshal(signedAddr)
	if err != nil {
		return shim.Error(err.Error())
	}

	if err := stub.PutState(newPkToAddrCompositeKey, addrChangeMsg); err != nil {
		return shim.Error(err.Error())
	}

	// set new address -> key mapping
	if err := stub.PutState(addrToPkCompositeKey, []byte(newkey)); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

// ChangeMultisigPublicKey changes public key of multisig member
// arg[0] - multisig adress (base58check)
// arg[1] - old key (base58)
// arg[2] - new key (base58)
// arg[3] - reason (string)
// arg[4] - reason ID (string)
// arg[5] - nonce
// arg[6:] - public keys and signatures of validators
func (c *ACL) ChangeMultisigPublicKey(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	argsNum := len(args)
	if argsNum < 8 {
		return shim.Error(fmt.Sprintf("incorrect number of arguments: %d, but this method expects: address, old key, new key, reason, reason ID, nonce, public keys, signatures", argsNum))
	}

	if err := c.checkCert(stub); err != nil {
		return shim.Error(fmt.Sprintf("unauthorized: %s", err.Error()))
	}

	multisigAddr := args[0]
	oldKey := args[1]
	encodedBase58NewPublicKey := args[2]
	reason := args[3]
	if len(reason) == 0 {
		return shim.Error("reasonnot provided")
	}
	if len(args[4]) == 0 {
		return shim.Error("reason ID not provided")
	}
	reasonId, err := strconv.Atoi(args[4])
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to convert reason ID to int, err: %s", err.Error()))
	}

	nonce := args[5]
	pksAndSignatures := args[6:]
	if len(multisigAddr) == 0 {
		return shim.Error("empty address")
	}
	if len(oldKey) == 0 {
		return shim.Error("empty old key")
	}
	if len(encodedBase58NewPublicKey) == 0 {
		return shim.Error("empty new key")
	}
	if len(nonce) == 0 {
		return shim.Error("empty nonce")
	}
	if len(pksAndSignatures) == 0 {
		return shim.Error("no public keys and signatures provided")
	}

	pks := pksAndSignatures[:len(pksAndSignatures)/2]
	signatures := pksAndSignatures[len(pksAndSignatures)/2:]

	if err := checkNonce(stub, multisigAddr, nonce); err != nil {
		return shim.Error(err.Error())
	}

	addrToPkCompositeKey, err := stub.CreateCompositeKey(pkPrefix, []string{multisigAddr})
	if err != nil {
		return shim.Error(err.Error())
	}

	// check that we have pub key for such address
	keys, err := stub.GetState(addrToPkCompositeKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(keys) == 0 {
		return shim.Error(fmt.Sprintf("no pub keys for address %s", multisigAddr))
	}

	pkToAddrCompositeKey, err := stub.CreateCompositeKey(addressPrefix, []string{string(keys)})
	if err != nil {
		return shim.Error(err.Error())
	}

	// get pb.SignedAddress
	signedAddrBytes, err := stub.GetState(pkToAddrCompositeKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	if len(signedAddrBytes) == 0 {
		return shim.Error(fmt.Sprintf("no SignedAddress msg for address %s", multisigAddr))
	}
	signedAddr := &pb.SignedAddress{}
	if err := proto.Unmarshal(signedAddrBytes, signedAddr); err != nil {
		return shim.Error(err.Error())
	}

	// update pubkeys list
	var newKeys []string
	for index, pk := range signedAddr.SignaturePolicy.PubKeys {
		if base58.Encode(pk) == oldKey {
			decodedPublicKey, err := decodeBase58PublicKey(encodedBase58NewPublicKey)
			if err != nil {
				return shim.Error(err.Error())
			}
			signedAddr.SignaturePolicy.PubKeys[index] = decodedPublicKey
			newKeys = append(newKeys, encodedBase58NewPublicKey)
		} else {
			newKeys = append(newKeys, base58.Encode(signedAddr.SignaturePolicy.PubKeys[index]))
		}
	}

	newKeysString := strings.Join(newKeys, "/")
	message := append([]string{"changeMultisigPublicKey", multisigAddr, oldKey, newKeysString, reason, args[4], nonce}, pks...)
	hashedMessage := sha3.Sum256([]byte(strings.Join(message, "")))
	if err := c.checkValidatorsSigned(hashedMessage[:], pks, signatures); err != nil {
		return shim.Error(err.Error())
	}

	// ReplaceKeysSignedTx contains strings array ["changeMultisigPublicKey", multisig address, old pk (base58), new pub keys of multisig members (base58), nonce, validators public keys, validators signatures]
	signedAddr.SignaturePolicy.ReplaceKeysSignedTx = append(message, signatures...)

	// add reason
	signedAddr.Reason = reason
	signedAddr.ReasonId = int32(reasonId)

	// and delete
	err = stub.DelState(pkToAddrCompositeKey)
	if err != nil {
		return shim.Error(err.Error())
	}

	// del old addr -> pub key mapping
	err = stub.DelState(addrToPkCompositeKey)
	if err != nil {
		return shim.Error(err.Error())
	}

	addrChangeMsg, err := proto.Marshal(signedAddr)
	if err != nil {
		return shim.Error(err.Error())
	}

	strKeys := strings.Split(newKeysString, "/")
	if err := checkKeysArr(strKeys); err != nil {
		return shim.Error(fmt.Sprintf("%s, input: '%s'", err.Error(), newKeysString))
	}
	hashedHexKeys, err := keyStringToSortedHashedHex(strKeys)
	if err != nil {
		return shim.Error(err.Error())
	}

	// set new key -> pb.SignedAddress mapping
	newPkToAddrCompositeKey, err := stub.CreateCompositeKey(addressPrefix, []string{hashedHexKeys})
	if err != nil {
		return shim.Error(err.Error())
	}
	if err := stub.PutState(newPkToAddrCompositeKey, addrChangeMsg); err != nil {
		return shim.Error(err.Error())
	}

	// set new address -> key mapping
	if err := stub.PutState(addrToPkCompositeKey, []byte(hashedHexKeys)); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func toLowerFirstLetter(in string) string {
	return string(unicode.ToLower(rune(in[0]))) + in[1:]
}

func checkNonce(stub shim.ChaincodeStubInterface, sender, nonceStr string) error {
	key, err := stub.CreateCompositeKey(noncePrefix, []string{sender})

	nonce, ok := new(big.Int).SetString(nonceStr, 10)
	if !ok {
		return errors.New("incorrect nonce")
	}
	data, err := stub.GetState(key)
	if err != nil {
		return err
	}
	existed := new(big.Int).SetBytes(data)
	if existed.Cmp(nonce) >= 0 {
		return errors.New("incorrect nonce")
	}
	return stub.PutState(key, nonce.Bytes())
}

func (c *ACL) checkValidatorsSignedWithBase58Signature(message []byte, pks, signatures []string) error {
	countValidatorsSigned := 0
	if signDublicates := checkDuplicates(signatures); len(signDublicates) > 0 {
		return fmt.Errorf("dublicate validators signatures are not allowed %v", signDublicates)
	}
	if pkDublicates := checkDuplicates(pks); len(pkDublicates) > 0 {
		return fmt.Errorf("dublicate validators public keys are not allowed %v", pkDublicates)
	}

	for i, encodedBase58PublicKey := range pks {
		if !IsValidator(c.init.Validators, encodedBase58PublicKey) {
			return errors.Errorf("pk %s does not belong to any validator", encodedBase58PublicKey)
		}
		countValidatorsSigned++

		// check signature
		decodedSignature := base58.Decode(signatures[i])
		decodedPublicKey, err := decodeBase58PublicKey(encodedBase58PublicKey)
		if err != nil {
			return err
		}
		if !ed25519.Verify(decodedPublicKey, message, decodedSignature) {
			return errors.Errorf("the signature %s does not match the public key %s", signatures[i], encodedBase58PublicKey)
		}
	}

	if countValidatorsSigned < c.init.ValidatorsCount {
		return errors.Errorf("%d of %d signed", countValidatorsSigned, c.init.ValidatorsCount)
	}
	return nil
}

func (c *ACL) checkValidatorsSigned(message []byte, pks, hexSignatures []string) error {
	countValidatorsSigned := 0
	if signDublicates := checkDuplicates(hexSignatures); len(signDublicates) > 0 {
		return fmt.Errorf("dublicate validators signatures are not allowed %v", signDublicates)
	}
	if pkDublicates := checkDuplicates(pks); len(pkDublicates) > 0 {
		return fmt.Errorf("dublicate validators public keys are not allowed %v", pkDublicates)
	}

	for i, encodedBase58PublicKey := range pks {
		if !IsValidator(c.init.Validators, encodedBase58PublicKey) {
			return errors.Errorf("pk %s does not belong to any validator", encodedBase58PublicKey)
		}
		countValidatorsSigned++

		// check signature
		decodedSignature, err := hex.DecodeString(hexSignatures[i])
		if err != nil {
			return err
		}
		decodedPublicKey, err := decodeBase58PublicKey(encodedBase58PublicKey)
		if err != nil {
			return err
		}
		if !ed25519.Verify(decodedPublicKey, message, decodedSignature) {
			// TODO why signature in error in base58 format?
			// in this method args signatures in hex
			return errors.Errorf("the signature %s does not match the public key %s", base58.Encode(decodedSignature), encodedBase58PublicKey)
		}
	}

	if countValidatorsSigned < c.init.ValidatorsCount {
		return errors.Errorf("%d of %d signed", countValidatorsSigned, c.init.ValidatorsCount)
	}
	return nil
}

// checkDuplicates checks string array for duplicates and returns duplicates if exists.
func checkDuplicates(arr []string) (duplicateBuffer []string) {
	itemsMap := make(map[string]struct{})
	for _, item := range arr {
		if _, ok := itemsMap[item]; ok {
			if !stringSliceContains(duplicateBuffer, item) {
				duplicateBuffer = append(duplicateBuffer, item)
			}
		} else {
			itemsMap[item] = struct{}{}
		}
	}
	return
}

func stringSliceContains(arr []string, item string) bool {
	for _, found := range arr {
		if item == found {
			return true
		}
	}
	return false
}

func checkNOutMSigned(n int, message []byte, pks [][]byte, signatures []string) error {
	if signDublicates := checkDuplicates(signatures); len(signDublicates) > 0 {
		return fmt.Errorf("dublicate validators signatures are not allowed %v", signDublicates)
	}

	var strPubKeys []string
	for _, pk := range pks {
		strPubKeys = append(strPubKeys, hex.EncodeToString(pk))
	}

	if pkDublicates := checkDuplicates(strPubKeys); len(pkDublicates) > 0 {
		return fmt.Errorf("dublicate validators public keys are not allowed %v", pkDublicates)
	}

	countSigned := 0
	for i, pk := range pks {
		// check signature
		decodedSignature, err := hex.DecodeString(signatures[i])
		if err != nil {
			return err
		}

		if !ed25519.Verify(pk, message, decodedSignature) {
			return errors.Errorf("the signature %s does not match the public key %s", signatures[i], hex.EncodeToString(pk))
		}
		countSigned++
	}

	if countSigned < n {
		return errors.Errorf("%d of %d signed", countSigned, n)
	}
	return nil
}

// IsValidator checks whether a public key belongs to authorized entities and returns true or false
func IsValidator(authorities []string, pk string) bool {
	// check it was a validator
	for _, authorityPublicKey := range authorities {
		if authorityPublicKey == pk {
			return true
		}
	}
	return false
}

func keyStringToSortedHashedHex(keys []string) (string, error) {
	binKeys := make([][]byte, len(keys))
	for i, encodedBase58PublicKey := range keys {
		publicKey, err := decodeBase58PublicKey(encodedBase58PublicKey)
		if err != nil {
			return "", err
		}
		binKeys[i] = publicKey
	}
	sort.Slice(binKeys, func(i, j int) bool {
		return bytes.Compare(binKeys[i], binKeys[j]) < 0
	})
	hashed := sha3.Sum256(bytes.Join(binKeys, []byte("")))
	return hex.EncodeToString(hashed[:]), nil
}

func DecodeAndSort(item string) ([][]byte, error) {
	strKeys := strings.Split(item, "/")
	binKeys := make([][]byte, len(strKeys))
	for i, encodedBase58PublicKey := range strKeys {
		decodedPublicKey, err := decodeBase58PublicKey(encodedBase58PublicKey)
		if err != nil {
			return nil, err
		}
		binKeys[i] = decodedPublicKey
	}
	sort.Slice(binKeys, func(i, j int) bool {
		return bytes.Compare(binKeys[i], binKeys[j]) < 0
	})
	return binKeys, nil
}

func hashedHex(items [][]byte) string {
	hashed := sha3.Sum256(bytes.Join(items, []byte("")))
	return hex.EncodeToString(hashed[:])
}

func getAddressByHashedKeys(stub shim.ChaincodeStubInterface, keys string) (*pb.SignedAddress, error) {
	pkToAddrCompositeKey, err := stub.CreateCompositeKey(addressPrefix, []string{keys})
	if err != nil {
		return nil, err
	}

	keyData, err := stub.GetState(pkToAddrCompositeKey)
	if err != nil {
		return nil, err
	}
	if len(keyData) == 0 {
		return nil, fmt.Errorf("not found any records")
	}
	var a pb.SignedAddress
	if err := proto.Unmarshal(keyData, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (c *ACL) checkCert(stub shim.ChaincodeStubInterface) error {
	cert, err := stub.GetCreator()
	if err != nil {
		return err
	}
	sId := &msp.SerializedIdentity{}
	if err := proto.Unmarshal(cert, sId); err != nil {
		return fmt.Errorf("could not deserialize a SerializedIdentity, err %s", err)
	}
	b, _ := pem.Decode(sId.IdBytes)
	parsed, err := x509.ParseCertificate(b.Bytes)
	if err != nil {
		return err
	}
	pk := parsed.PublicKey.(*ecdsa.PublicKey)
	hash := sha256.New()
	hash.Write(elliptic.Marshal(pk.Curve, pk.X, pk.Y))
	hashed := sha3.Sum256(cert)
	if !bytes.Equal(hashed[:], c.init.AdminSKI) &&
		!bytes.Equal(hash.Sum(nil), c.init.AdminSKI) {
		return errors.New("unauthorized")
	}
	return nil
}

func decodeBase58PublicKey(encodedBase58PublicKey string) ([]byte, error) {
	if len(encodedBase58PublicKey) == 0 {
		return nil, errors.New("empty pub key")
	}
	decode := base58.Decode(encodedBase58PublicKey)
	if len(decode) == 0 {
		return nil, fmt.Errorf("failed base58 decoding of key %s", encodedBase58PublicKey)
	}
	return decode, nil
}

func checkKeysArr(keysArr []string) error {
	uniqPks := make(map[string]struct{})
	for _, p := range keysArr {
		if p == "" {
			return fmt.Errorf("empty public key detected")
		}
		if _, ok := uniqPks[p]; ok {
			return fmt.Errorf("duplicated public keys")
		}
		uniqPks[p] = struct{}{}
	}
	return nil
}

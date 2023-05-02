package utils

import (
	"strconv"
	"time"
)

const (
	// EnvHlfProxyURL - domain and port for hlf proxy service, example http://localhost:9001 without '/' on the end the string
	EnvHlfProxyURL = "HLF_PROXY_URL"
	// EnvHlfProxyAuthToken - support Basic Auth with auth token
	EnvHlfProxyAuthToken = "HLF_PROXY_AUTH_TOKEN" //nolint:gosec
	// EnvFiatIssuerPrivateKey - issuer private key ed25519 in base58 check
	EnvFiatIssuerPrivateKey = "FIAT_ISSUER_PRIVATE_KEY"
	// BatchTransactionTimeout - common time execution of following process
	// robot - defaultBatchLimits.batchTimeoutLimit
	// Time batch execute by robot
	BatchTransactionTimeout = 3 * time.Second
	// InvokeTimeout sets timeout for invoke method operations
	InvokeTimeout = 10 * time.Second
	// QueryTimeout sets timeout for query method operations
	QueryTimeout = 10 * time.Second
	// QueryTimeout sets timeout for query method operations
	MoreNonceTTL = 11 * time.Second
)

func AsBytes(args ...string) [][]byte {
	bytes := make([][]byte, len(args))
	for i, arg := range args {
		bytes[i] = []byte(arg)
	}
	return bytes
}

func GetNonce() string {
	return strconv.FormatInt(time.Now().UnixMilli(), 10) //nolint:gomnd
}

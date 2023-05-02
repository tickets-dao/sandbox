package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type HlfProxyService struct {
	// url - domain and port for hlf proxy service, example http://localhost:9001 without '/' on the end the string
	url string
	// authToken - support Basic Auth with auth token
	authToken string
}

func NewHlfProxyService(url string, authToken string) *HlfProxyService {
	return &HlfProxyService{
		url:       url,
		authToken: authToken,
	}
}

// Invoke - send invoke request to hlf through hlf proxy service
func (p *HlfProxyService) Invoke(chaincodeID string, fcn string, args ...string) (*Response, error) {
	return p.sendRequest("invoke", chaincodeID, fcn, args...)
}

// Query - send query request to hlf through hlf proxy service
func (p *HlfProxyService) Query(chaincodeID string, fcn string, args ...string) (*Response, error) {
	return p.sendRequest("query", chaincodeID, fcn, args...)
}

//nolint:funlen
func (p *HlfProxyService) sendRequest(requestType string, chaincodeID string, fcn string, args ...string) (*Response, error) {
	fmt.Printf("requestType: %s\n", requestType)
	fmt.Printf("chaincodeID: %s\n", chaincodeID)
	fmt.Printf("fcn: %s\n", fcn)
	fmt.Printf("args: %s\n", args)

	requestData := Request{
		Args:        AsBytes(args...),
		ChaincodeID: chaincodeID,
		Fcn:         fcn,
	}

	requestPayload, err := json.Marshal(requestData)
	if err != nil {
		return nil, err
	}

	httpRequest, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		fmt.Sprintf("%s/%s", p.url, requestType),
		bytes.NewReader(requestPayload),
	)
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Add("authorization", fmt.Sprintf("Basic %s", p.authToken))
	httpRequest.Header.Add("content-type", "application/json")

	httpResponse, err := http.DefaultClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	if httpResponse == nil {
		return nil, errors.New("response not found")
	}

	defer func() {
		err = httpResponse.Body.Close()
		if err != nil {
			fmt.Println(fmt.Errorf("error close body http response - error: %w", err))
		}
	}()

	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return nil, err
	}

	if httpResponse.StatusCode != http.StatusOK {
		responseError := &ResponseError{}
		err = json.Unmarshal(body, responseError)
		if err != nil {
			return nil, err
		}

		return nil, errors.New(responseError.Message)
	}

	fmt.Println(string(body))
	response := &Response{}
	err = json.Unmarshal(body, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

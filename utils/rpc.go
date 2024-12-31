package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type RPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

func batchRequest(url string, requests []RPCRequest) ([]map[string]interface{}, error) {
	jsonData, err := json.Marshal(requests)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response, nil
}

func batchTokenSupply() {
	url := "https://api.mainnet-beta.solana.com" // Solana RPC URL
	requests := []RPCRequest{
		{
			JSONRPC: "2.0",
			Method:  "getTokenSupply",
			Params:  []interface{}{"<mint_address_1>", map[string]string{"commitment": "confirmed"}},
			ID:      1,
		},
		{
			JSONRPC: "2.0",
			Method:  "getTokenSupply",
			Params:  []interface{}{"<mint_address_2>", map[string]string{"commitment": "confirmed"}},
			ID:      2,
		},
	}

	response, err := batchRequest(url, requests)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Batch Request Response:", response)
}

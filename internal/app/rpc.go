package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}
type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func callBitcoinRPC(ctx context.Context, method string, params []any) (json.RawMessage, error) {
	url := strings.TrimSpace(os.Getenv("BTC_RPC_URL"))
	user := os.Getenv("BTC_RPC_USER")
	pass := os.Getenv("BTC_RPC_PASS")
	if url == "" || user == "" || pass == "" {
		return nil, fmt.Errorf("configure BTC_RPC_URL, BTC_RPC_USER and BTC_RPC_PASS")
	}
	payload, err := json.Marshal(rpcRequest{JSONRPC: "1.0", ID: "btc-kit", Method: method, Params: params})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(user, pass)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Bitcoin Core RPC request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Bitcoin Core RPC returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	var decoded rpcResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return nil, fmt.Errorf("decode RPC response: %w", err)
	}
	if decoded.Error != nil {
		return nil, fmt.Errorf("Bitcoin Core RPC error %d: %s", decoded.Error.Code, decoded.Error.Message)
	}
	if len(decoded.Result) == 0 {
		return nil, fmt.Errorf("Bitcoin Core RPC returned an empty result")
	}
	return decoded.Result, nil
}

func RunNodeHealth(ctx context.Context, out io.Writer) error {
	result, err := callBitcoinRPC(ctx, "getblockchaininfo", []any{})
	if err != nil {
		return err
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, result, "", "  "); err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, pretty.String())
	return err
}

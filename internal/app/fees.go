package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const defaultFeeURL = "https://mempool.space/api/v1/fees/recommended"

type FeeEstimates struct {
	FastestFee  float64 `json:"fastestFee"`
	HalfHourFee float64 `json:"halfHourFee"`
	HourFee     float64 `json:"hourFee"`
	EconomyFee  float64 `json:"economyFee"`
	MinimumFee  float64 `json:"minimumFee"`
}

func FetchFees(ctx context.Context) (FeeEstimates, error) {
	url := os.Getenv("BTC_KIT_FEE_URL")
	if url == "" {
		url = defaultFeeURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return FeeEstimates{}, fmt.Errorf("create fee request: %w", err)
	}
	req.Header.Set("User-Agent", "btc-kit/0.1.0")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return FeeEstimates{}, fmt.Errorf("fee provider request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return FeeEstimates{}, fmt.Errorf("fee provider returned HTTP %d", resp.StatusCode)
	}
	var fees FeeEstimates
	decoder := json.NewDecoder(io.LimitReader(resp.Body, (1<<20)+1))
	if err := decoder.Decode(&fees); err != nil {
		return FeeEstimates{}, fmt.Errorf("decode fee response: %w", err)
	}
	if fees.FastestFee <= 0 || fees.MinimumFee <= 0 {
		return FeeEstimates{}, fmt.Errorf("fee provider returned invalid or empty fee estimates")
	}
	return fees, nil
}

func RunFees(ctx context.Context, out interface{ Write([]byte) (int, error) }) error {
	fees, err := FetchFees(ctx)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, "Bitcoin fee estimates (sat/vB)")
	fmt.Fprintf(out, "Fastest:   %.2f\n", fees.FastestFee)
	fmt.Fprintf(out, "30 min:    %.2f\n", fees.HalfHourFee)
	fmt.Fprintf(out, "1 hour:    %.2f\n", fees.HourFee)
	fmt.Fprintf(out, "Economy:   %.2f\n", fees.EconomyFee)
	fmt.Fprintf(out, "Minimum:   %.2f\n", fees.MinimumFee)
	fmt.Fprintln(out, "Source: mempool.space; estimates are not guaranteed confirmation times.")
	return nil
}

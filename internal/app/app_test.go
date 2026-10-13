package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestIsLoopbackAddress(t *testing.T) {
	if !isLoopbackAddress("127.0.0.1:8080") {
		t.Fatal("expected IPv4 loopback")
	}
	if !isLoopbackAddress("[::1]:8080") {
		t.Fatal("expected IPv6 loopback")
	}
	if isLoopbackAddress("0.0.0.0:8080") {
		t.Fatal("wildcard address must not be loopback")
	}
	if isLoopbackAddress("example.com:8080") {
		t.Fatal("hostname must not be assumed loopback")
	}
}

func TestDecodeTransactionRejectsInvalidHex(t *testing.T) {
	var out bytes.Buffer
	if err := DecodeTransaction("not-hex", &out); err == nil {
		t.Fatal("expected invalid hex error")
	}
}

func TestDecodeTransactionRejectsEmpty(t *testing.T) {
	var out bytes.Buffer
	if err := DecodeTransaction("", &out); err == nil {
		t.Fatal("expected empty input error")
	}
}

func TestFetchFeesUsesConfiguredProvider(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"fastestFee":20,"halfHourFee":15,"hourFee":10,"economyFee":5,"minimumFee":1}`))
	}))
	defer srv.Close()
	old := os.Getenv("BTC_KIT_FEE_URL")
	defer os.Setenv("BTC_KIT_FEE_URL", old)
	_ = os.Setenv("BTC_KIT_FEE_URL", srv.URL)
	fees, err := FetchFees(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if fees.FastestFee != 20 || fees.MinimumFee != 1 {
		t.Fatalf("unexpected fees: %+v", fees)
	}
}

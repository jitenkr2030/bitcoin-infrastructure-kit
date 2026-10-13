package app

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func envDuration(name string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	d, err := time.ParseDuration(value)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

func isLoopbackAddress(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func RunServer(ctx context.Context) error {
	addr := strings.TrimSpace(os.Getenv("BTC_KIT_ADDR"))
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	apiKey := os.Getenv("BTC_KIT_API_KEY")
	if !isLoopbackAddress(addr) && len(apiKey) < 32 {
		return errors.New("non-loopback bind requires BTC_KIT_API_KEY of at least 32 characters; use TLS reverse proxy before public deployment")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "service": "btc-kit"})
	})
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		jsonResponse(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("/v1/fees", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if apiKey != "" {
			provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if subtle.ConstantTimeCompare([]byte(provided), []byte(apiKey)) != 1 {
				jsonResponse(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
		}
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		fees, err := FetchFees(ctx)
		if err != nil {
			log.Printf("fee request failed: %v", err)
			jsonResponse(w, http.StatusBadGateway, map[string]string{"error": "fee provider unavailable"})
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=15")
		jsonResponse(w, http.StatusOK, fees)
	})

	server := &http.Server{
		Addr: addr, Handler: securityHeaders(requestLog(mux)),
		ReadHeaderTimeout: envDuration("BTC_KIT_READ_TIMEOUT", 5*time.Second),
		ReadTimeout:       envDuration("BTC_KIT_READ_TIMEOUT", 5*time.Second),
		WriteTimeout:      envDuration("BTC_KIT_WRITE_TIMEOUT", 15*time.Second),
		IdleTimeout:       envDuration("BTC_KIT_IDLE_TIMEOUT", 60*time.Second),
		MaxHeaderBytes:    1 << 20,
	}
	errCh := make(chan error, 1)
	go func() {
		log.Printf("btc-kit API listening on %s", addr)
		log.Printf("routes: GET /health, GET /ready, GET /v1/fees")
		errCh <- server.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), envDuration("BTC_KIT_SHUTDOWN_TIMEOUT", 10*time.Second))
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("server shutdown: %w", err)
		}
		err := <-errCh
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

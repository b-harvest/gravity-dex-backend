package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	apiBaseURL       = "https://pro-api.coinmarketcap.com/"
	apiKeyHeaderName = "X-CMC_PRO_API_KEY"
)

var _ PriceService = (*CoinMarketCapPriceService)(nil)

type Price struct {
	Symbol string
	Price  float64
}

type PriceService interface {
	Prices(ctx context.Context, symbols ...string) ([]Price, error)
}

type CoinMarketCapPriceService struct {
	apiBaseURL *url.URL
	hc         *http.Client
	apiKey     string
}

func NewCoinMarketCapPriceService(apiKey string) (PriceService, error) {
	u, err := url.Parse(apiBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse api base url: %w", err)
	}
	hc := &http.Client{}
	return &CoinMarketCapPriceService{u, hc, apiKey}, nil
}

func (s *CoinMarketCapPriceService) Prices(ctx context.Context, symbols ...string) ([]Price, error) {
	r, err := s.request(ctx, "/v1/cryptocurrency/quotes/latest", url.Values{
		"symbol": {strings.Join(symbols, ",")},
		"aux":    {""},
	})
	if err != nil {
		return nil, err
	}
	var data map[string]struct {
		Quote struct {
			USD struct {
				Price float64 `json:"price"`
			} `json:"USD"`
		} `json:"quote"`
	}
	if err := json.Unmarshal(r.Data, &data); err != nil {
		return nil, fmt.Errorf("unmarshal data: %w", err)
	}
	var ps []Price
	for _, symbol := range symbols {
		d, ok := data[strings.ToUpper(symbol)]
		if !ok {
			return nil, fmt.Errorf("price for symbol %q not found", symbol)
		}
		ps = append(ps, Price{
			Symbol: symbol,
			Price:  d.Quote.USD.Price,
		})
	}
	return ps, nil
}

func (s *CoinMarketCapPriceService) request(ctx context.Context, path string, params url.Values) (*CoinMarketCapResponse, error) {
	u, err := s.apiBaseURL.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("resolve url for path: %w", err)
	}
	u.RawQuery = params.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Accepts", "application/json")
	req.Header.Set(apiKeyHeaderName, s.apiKey)
	resp, err := s.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	defer io.Copy(io.Discard, resp.Body)
	var r CoinMarketCapResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("decode body: %w", err)
	}
	if r.Status.ErrorCode != 0 {
		return &r, &CoinMarketCapError{r.Status.ErrorCode, r.Status.ErrorMessage}
	}
	return &r, nil
}

type CoinMarketCapResponse struct {
	Status struct {
		Timestamp    time.Time `json:"timestamp"`
		ErrorCode    int       `json:"error_code"`
		ErrorMessage string    `json:"error_message"`
		Elapsed      int       `json:"elapsed"`
		CreditCount  int       `json:"credit_count"`
	} `json:"status"`
	Data json.RawMessage `json:"data"`
}

type CoinMarketCapError struct {
	ErrorCode    int
	ErrorMessage string
}

func (e *CoinMarketCapError) Error() string {
	return fmt.Sprintf("%d: %s", e.ErrorCode, e.ErrorMessage)
}

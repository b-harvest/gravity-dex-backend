package price

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
	cmcAPIBaseURL       = "https://pro-api.coinmarketcap.com/"
	cmcAPIKeyHeaderName = "X-CMC_PRO_API_KEY"
)

var _ Service = (*CoinMarketCapService)(nil)

type CoinMarketCapService struct {
	baseURL        *url.URL
	hc             *http.Client
	apiKey         string
	caches         map[string]cache
	updateInterval time.Duration
}

func NewCoinMarketCapService(apiKey string, updateInterval time.Duration) (Service, error) {
	u, err := url.Parse(cmcAPIBaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse api base url: %w", err)
	}
	hc := &http.Client{}
	return &CoinMarketCapService{u, hc, apiKey, make(map[string]cache), updateInterval}, nil
}

func (s *CoinMarketCapService) Prices(ctx context.Context, symbols ...string) (Table, error) {
	symbolsToUpdate := s.symbolsToUpdate(symbols)
	symbolSetToUpdate := make(map[string]struct{})
	for _, symbol := range symbolsToUpdate {
		symbolSetToUpdate[symbol] = struct{}{}
	}
	var data map[string]struct {
		Quote struct {
			USD struct {
				Price float64 `json:"price"`
			} `json:"USD"`
		} `json:"quote"`
	}
	if len(symbolsToUpdate) > 0 {
		r, err := s.request(ctx, "/v1/cryptocurrency/quotes/latest", url.Values{
			"symbol": {strings.Join(symbolsToUpdate, ",")},
			"aux":    {""},
		})
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(r.Data, &data); err != nil {
			return nil, fmt.Errorf("unmarshal data: %w", err)
		}
	}
	now := time.Now()
	t := make(Table)
	for _, symbol := range symbols {
		if _, ok := symbolSetToUpdate[symbol]; ok {
			d, ok := data[strings.ToUpper(symbol)]
			if !ok {
				return nil, fmt.Errorf("price for symbol %q not found", symbol)
			}
			s.caches[strings.ToLower(symbol)] = cache{d.Quote.USD.Price, now}
			t[strings.ToLower(symbol)] = d.Quote.USD.Price
		} else {
			c, ok := s.caches[strings.ToLower(symbol)]
			if !ok { // will never happen!
				return nil, fmt.Errorf("cache for symbol %q not found", symbol)
			}
			t[strings.ToLower(symbol)] = c.price
		}
	}
	return t, nil
}

func (s *CoinMarketCapService) request(ctx context.Context, path string, params url.Values) (*CoinMarketCapResponse, error) {
	u, err := s.baseURL.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("resolve url for path: %w", err)
	}
	u.RawQuery = params.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Accepts", "application/json")
	req.Header.Set(cmcAPIKeyHeaderName, s.apiKey)
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

func (s *CoinMarketCapService) clearCaches() {
	now := time.Now()
	for symbol, c := range s.caches {
		if !c.updatedAt.Add(s.updateInterval).After(now) {
			delete(s.caches, symbol)
		}
	}
}

func (s *CoinMarketCapService) symbolsToUpdate(symbols []string) []string {
	s.clearCaches()
	var res []string
	for _, symbol := range symbols {
		if _, ok := s.caches[strings.ToLower(symbol)]; !ok {
			res = append(res, symbol)
		}
	}
	return res
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

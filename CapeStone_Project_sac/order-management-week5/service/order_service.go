package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	"order-management/model"
	"order-management/repository"

	"github.com/sony/gobreaker"
)

var ErrInvalid = errors.New("invalid input")
var ErrProductUnavailable = errors.New("product service unavailable")

type ProductClient interface {
	Price(context.Context, int64) (float64, error)
}
type HTTPProductClient struct {
	baseURL string
	client  *http.Client
	retries int
	breaker *gobreaker.CircuitBreaker
}

func NewHTTPProductClient(url string, timeout time.Duration, retries int) *HTTPProductClient {
	return &HTTPProductClient{baseURL: url, client: &http.Client{Timeout: timeout}, retries: retries, breaker: gobreaker.NewCircuitBreaker(gobreaker.Settings{Name: "product-service", MaxRequests: 1, Interval: 30 * time.Second, Timeout: 15 * time.Second})}
}
func (p *HTTPProductClient) Price(ctx context.Context, id int64) (float64, error) {
	if p.baseURL == "" {
		return 0, fmt.Errorf("%w: PRODUCT_SERVICE_URL is not configured", ErrProductUnavailable)
	}
	v, err := p.breaker.Execute(func() (interface{}, error) {
		var last error
		for attempt := 0; attempt <= p.retries; attempt++ {
			reqCtx, cancel := context.WithTimeout(ctx, p.client.Timeout)
			req, e := http.NewRequestWithContext(reqCtx, http.MethodGet, fmt.Sprintf("%s/products/%d", p.baseURL, id), nil)
			if e == nil {
				resp, re := p.client.Do(req)
				if re == nil {
					if resp.StatusCode >= 200 && resp.StatusCode < 300 {
						var body struct {
							Price     float64 `json:"price"`
							UnitPrice float64 `json:"unitPrice"`
						}
						e = json.NewDecoder(resp.Body).Decode(&body)
						resp.Body.Close()
						cancel()
						if e == nil {
							price := body.Price
							if price == 0 {
								price = body.UnitPrice
							}
							if price > 0 {
								return price, nil
							}
							e = fmt.Errorf("product service returned invalid price")
						}
					} else {
						resp.Body.Close()
						e = fmt.Errorf("product service returned status %d", resp.StatusCode)
					}
				} else {
					e = re
				}
			}
			cancel()
			last = e
			if attempt < p.retries {
				delay := time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(delay):
				}
			}
		}
		return nil, last
	})
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrProductUnavailable, err)
	}
	return v.(float64), nil
}

type OrderService struct {
	repo     repository.OrderRepository
	products ProductClient
}

func NewOrderService(r repository.OrderRepository, p ProductClient, _ int) *OrderService {
	return &OrderService{repo: r, products: p}
}
func (s *OrderService) Create(ctx context.Context, req model.CreateOrderRequest) (model.Order, error) {
	if req.CustomerID <= 0 || req.ProductID <= 0 || req.Quantity < 1 || req.Quantity > 10000 {
		return model.Order{}, ErrInvalid
	}
	price, err := s.products.Price(ctx, req.ProductID)
	if err != nil {
		return model.Order{}, err
	}
	return s.repo.Create(ctx, model.Order{CustomerID: req.CustomerID, ProductID: req.ProductID, Quantity: req.Quantity, UnitPrice: price, TotalAmount: price * float64(req.Quantity), Status: model.StatusCreated})
}
func (s *OrderService) Get(ctx context.Context, id int64) (model.Order, error) {
	if id <= 0 {
		return model.Order{}, ErrInvalid
	}
	return s.repo.Get(ctx, id)
}
func (s *OrderService) List(ctx context.Context, limit, offset int) ([]model.Order, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, limit, offset)
}
func (s *OrderService) UpdateStatus(ctx context.Context, id int64, status model.OrderStatus) (model.Order, error) {
	if id <= 0 {
		return model.Order{}, ErrInvalid
	}
	if status != model.StatusCreated && status != model.StatusConfirmed && status != model.StatusCancelled {
		return model.Order{}, ErrInvalid
	}
	return s.repo.UpdateStatus(ctx, id, status)
}

package accrual

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/cookiejar"
	"time"

	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
	"github.com/go-resty/resty/v2"
)

type OrderAccrual struct {
	Order   string
	Status  string
	Accrual float32
}

type accrualSystem struct {
	client  *resty.Client
	limit   int
	timeout time.Duration
}

func NewAccrual(addr string, lim int, timeout time.Duration) (*accrualSystem, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("can't create cookie jar, %w", err)
	}
	c := resty.New().SetBaseURL(addr).SetCookieJar(jar)

	return &accrualSystem{
		client:  c,
		limit:   lim,
		timeout: timeout,
	}, nil
}

func (a *accrualSystem) Limit() int {
	return a.limit
}

func (a *accrualSystem) Timeout() time.Duration {
	return a.timeout
}

func (a *accrualSystem) GetOrderAccrual(ctx context.Context, orderNum string) (*OrderAccrual, error) {
	orderAccrual := new(OrderAccrual)

	resp, err := a.client.R().Get("/api/orders/" + orderNum)
	if err != nil {
		logger.Log(ctx).Errorf("order: failed sending request to accrual, %v", err)
		return nil, err
	}

	jsonErr := json.Unmarshal(resp.Body(), &orderAccrual)
	if jsonErr != nil {
		logger.Log(ctx).Errorf("order: failed parsing response from accrual, %w", jsonErr)
		return nil, err
	}

	return orderAccrual, nil
}

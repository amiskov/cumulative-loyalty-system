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

type accrualHTTP struct {
	client  *resty.Client
	limit   int
	timeout time.Duration
}

func NewHTTPAccrual(addr string, lim int, timeout time.Duration) (*accrualHTTP, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("can't create cookie jar, %w", err)
	}
	c := resty.New().SetBaseURL(addr).SetCookieJar(jar)

	return &accrualHTTP{
		client:  c,
		limit:   lim,
		timeout: timeout,
	}, nil
}

func (a *accrualHTTP) Limit() int {
	return a.limit
}

func (a *accrualHTTP) Timeout() time.Duration {
	return a.timeout
}

func (a *accrualHTTP) GetOrderAccrual(ctx context.Context, orderNum string) (*OrderAccrual, error) {
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

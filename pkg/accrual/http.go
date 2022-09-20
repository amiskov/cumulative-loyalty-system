package accrual

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
)

type accrualHTTP struct {
	baseURL      string
	maxAttempts  int
	reqTimeout   time.Duration
	pollInterval time.Duration
	client       *http.Client
}

func NewHTTPClient(addr string, maxAttempts int, reqTimeout time.Duration, pollInterval time.Duration) *accrualHTTP {
	c := http.Client{
		Timeout: reqTimeout,
	}

	return &accrualHTTP{
		client:       &c,
		baseURL:      addr,
		maxAttempts:  maxAttempts,
		reqTimeout:   reqTimeout,
		pollInterval: pollInterval,
	}
}

func (a *accrualHTTP) MaxAttempts() int {
	return a.maxAttempts
}

func (a *accrualHTTP) Interval() time.Duration {
	return a.pollInterval
}

func (a *accrualHTTP) GetOrderAccrual(ctx context.Context, orderNum string) (*OrderAccrual, error) {
	orderAccrual := new(OrderAccrual)

	resp, err := a.client.Get(a.baseURL + "/api/orders/" + orderNum)
	if err != nil {
		logger.Log(ctx).Errorf("order: failed sending request to accrual, %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	jsonErr := json.Unmarshal(body, &orderAccrual)
	if jsonErr != nil {
		logger.Log(ctx).Errorf("order: failed parsing response from accrual, %w", jsonErr)
		return nil, err
	}

	return orderAccrual, nil
}

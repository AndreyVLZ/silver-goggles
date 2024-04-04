package accrual

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/order"
)

const defaultRetryAfter int = 8

type errRetryable struct {
	error
	after int
}

func (e errRetryable) Retry() int { return e.after }

type Repo struct {
	accURL *url.URL
	client *http.Client
}

type resp struct {
	Order   string   `json:"order"`
	Status  string   `json:"status"`
	Accrual *float64 `json:"accrual,omitempty"`
}

func New(accURL *url.URL) *Repo {
	return &Repo{
		client: http.DefaultClient,
		accURL: accURL,
	}
}

func correctStatus(status order.Status) order.Status {
	if status == order.StatusRegistered {
		return order.StatusNew
	}
	return status
}

func (repo *Repo) Load(ctx context.Context, number order.Number) (order.Info, error) {
	url := fmt.Sprintf(
		"http://%s/api/orders/%s",
		strings.TrimPrefix(repo.accURL.String(), "http://"), number.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return order.Info{}, fmt.Errorf("load build req: %w", err)
	}

	res, err := repo.client.Do(req)
	if err != nil {
		return order.Info{}, fmt.Errorf("load exec req: %w", err)
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK: // 200
		var orderResp resp
		if err := json.NewDecoder(res.Body).Decode(&orderResp); err != nil {
			return order.Info{}, fmt.Errorf("load decode body: %w", err)
		}

		return order.NewInfo(
			number,
			correctStatus(order.ParseStatus(orderResp.Status)),
			order.ParseAccrual(orderResp.Accrual),
		), nil

	case http.StatusNoContent: // 204
		return order.NewInfo(
			number,
			order.StatusNew,
			nil,
		), nil // "заказ не зарегистрирован в системе расчёта"

	case http.StatusTooManyRequests: // 429
		retry := defaultRetryAfter
		if retryVal := req.Header.Get("Retry-After"); retryVal != "" {
			var err error
			retry, err = strconv.Atoi(retryVal)
			if err != nil {
				return order.Info{}, err
			}
		}

		return order.Info{}, errRetryable{
			error: fmt.Errorf("превышено количество запросов к сервису"),
			after: retry,
		}

	case http.StatusInternalServerError: // 500
		return order.Info{}, errRetryable{
			error: fmt.Errorf("внутренняя ошибка сервера accrual"),
			after: defaultRetryAfter,
		}

	default:
		return order.Info{}, fmt.Errorf("statusCode не обработан")
	}
}

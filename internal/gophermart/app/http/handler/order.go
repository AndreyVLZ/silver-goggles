package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/order"
	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/user"
)

// ORDER
type saveOrderService interface {
	LoadOrder(context.Context, user.ID, order.Number) error
	Withdraw(context.Context, order.Order) error
}

type orderService interface {
	GetOrders(context.Context, user.ID) ([]order.Order, error)
	Withdrawals(context.Context, user.ID) ([]order.Order, error)
	Balance(context.Context, user.ID) (order.Balance, error)
}

type OrderHandler struct {
	saveService saveOrderService
	getService  orderService
	log         *slog.Logger
}

func NewOrderHandler(saveService saveOrderService, getService orderService, log *slog.Logger) OrderHandler {
	return OrderHandler{saveService: saveService, getService: getService, log: log}
}

type errWithdraw interface {
	IsSuccessful() bool
	error
}

type contextKey struct{}

func GetContextKey() contextKey { return contextKey{} }

func userIDFromReq(req *http.Request) (user.ID, error) {
	ctxKey := GetContextKey()
	v := req.Context().Value(ctxKey)

	userID, ok := v.(user.ID)
	if !ok {
		return user.NillID(), errors.New("err parse userID")
	}

	return userID, nil
}

func parseBodyToString(r io.ReadCloser) (string, error) {
	defer r.Close()
	b, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Загрузка номера заказа
// POST /api/user/orders
func (oh OrderHandler) LoadHandle() http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		userID, err := userIDFromReq(req)
		if err != nil {
			oh.log.Error("parse req", "err", err)
			http.Error(rw, err.Error(), statusInternal)
			return
		}

		numberStr, err := parseBodyToString(req.Body)
		if err != nil {
			oh.log.Error("parse Body", "err", err)
			http.Error(rw, err.Error(), statusInternal)
			return
		}

		number, err := order.ParseNumber(numberStr)
		if err != nil {
			oh.log.Error("parse Number", "err", err)
			http.Error(rw, err.Error(), statusUnprocess)
			return
		}

		if err := oh.saveService.LoadOrder(req.Context(), userID, number); err != nil {
			var errFieldCon errFieldConflict
			if errors.As(err, &errFieldCon) {
				data := errFieldCon.Data()
				if _, ok := data.(order.Number); ok {
					oh.log.Error("srv conflict Number", "err", errFieldCon.Error())
					http.Error(rw, err.Error(), statusUnprocess)
					return
				}

				if _, ok := data.(user.ID); ok {
					oh.log.Error("srv conflict userID", "err", errFieldCon.Error())
					http.Error(rw, errFieldCon.Error(), statusConflict)
					return
				}

				if _, ok := data.(order.ID); ok {
					oh.log.Debug("srv number exist", "msg", errFieldCon.Error())
					rw.WriteHeader(statusOK)
					return
				}

			}

			oh.log.Error("srv", "err", err)
			http.Error(rw, err.Error(), statusInternal)
			return
		}

		rw.WriteHeader(statusAccepted)
	}
}

// Запрос на списание средств
// POST /api/user/balance/withdraw
func (oh OrderHandler) Withdraw() http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		witdrawReq := struct {
			Order string  `json:"order"`
			Sum   float64 `json:"sum"`
		}{}

		if err := parseBody(req.Body, &witdrawReq); err != nil {
			http.Error(rw, err.Error(), statusInternal)
			return
		}

		userID, err := userIDFromReq(req)
		if err != nil {
			http.Error(rw, err.Error(), statusInternal)
			return
		}

		number, err := order.ParseNumber(witdrawReq.Order)
		if err != nil {
			http.Error(rw, err.Error(), statusUnprocess)
			return
		}

		acc := order.NewAccrual(witdrawReq.Sum)

		order := order.New(
			order.NewID(),
			userID,
			order.NewInfo(
				number,
				order.StatusWithdraw,
				acc,
			),
		)

		if err := oh.saveService.Withdraw(req.Context(), order); err != nil {
			var errWithdraw errWithdraw
			if errors.As(err, &errWithdraw) && errWithdraw.IsSuccessful() {
				http.Error(rw, errWithdraw.Error(), statusPaymentReq)
				return
			}

			http.Error(rw, err.Error(), statusInternal)
		}
	}
}

// Получение списка загруженных номеров заказов
// GET /api/user/orders
func (oh OrderHandler) GetOrders() http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		userID, err := userIDFromReq(req)
		if err != nil {
			http.Error(rw, err.Error(), statusInternal)
			return
		}

		orders, err := oh.getService.GetOrders(req.Context(), userID)
		if err != nil {
			http.Error(rw, err.Error(), statusInternal)
			return
		}

		if len(orders) == 0 {
			rw.WriteHeader(statusNoContent)
			return
		}

		type orderResp struct {
			Number  string    `json:"number"`
			Status  string    `json:"status"`
			Accrual *float64  `json:"accrual,omitempty"`
			Upload  time.Time `json:"uploaded_at"`
		}

		ordersResp := make([]orderResp, len(orders))

		fn := func(accPtr *order.Accrual) *float64 {
			if accPtr == nil {
				return nil
			}

			acc := accPtr.ToFloat()
			return &acc
		}

		for i := range orders {
			oi := orders[i].Info()
			ordersResp[i] = orderResp{
				Number:  oi.Number().String(),
				Status:  oi.Status().String(),
				Accrual: fn(oi.Accrual()),
				Upload:  oi.Date(),
			}
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(rw).Encode(ordersResp); err != nil {
			http.Error(rw, err.Error(), statusInternal)
		}
	}
}

// Получение информации о выводе средств
// GET /api/user/withdrawals
func (oh OrderHandler) Withdrawals() http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		userID, err := userIDFromReq(req)
		if err != nil {
			http.Error(rw, err.Error(), statusInternal)
			return
		}

		type orderResp struct {
			Order string    `json:"order"`
			Sum   float64   `json:"sum"`
			Date  time.Time `json:"processed_at"`
		}

		orders, err := oh.getService.Withdrawals(req.Context(), userID)
		if err != nil {
			http.Error(rw, err.Error(), statusInternal)
			return
		}

		if len(orders) == 0 {
			http.Error(rw, "нет списаний", statusNoContent)
			return
		}

		ordersResp := make([]orderResp, len(orders))
		for i := range orders {
			oi := orders[i].Info()
			ordersResp[i] = orderResp{
				Order: oi.Number().String(),
				Sum:   oi.Accrual().ToFloat(),
				Date:  oi.Date(),
			}
		}

		rw.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(rw).Encode(ordersResp); err != nil {
			http.Error(rw, err.Error(), statusInternal)
		}
	}
}

// Получение текущего баланса пользователя
// GET /api/user/balance
func (oh OrderHandler) Balance() http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		userID, err := userIDFromReq(req)
		if err != nil {
			http.Error(rw, err.Error(), statusInternal)
			return
		}

		type balanceResp struct {
			Current   float64 `json:"current"`
			Withdrawn float64 `json:"withdrawn"`
		}

		balance, err := oh.getService.Balance(req.Context(), userID)
		if err != nil {
			http.Error(rw, err.Error(), statusInternal)
			return
		}

		blnResp := balanceResp{
			Current:   balance.Current().ToFloat(),
			Withdrawn: balance.Withdraw().ToFloat(),
		}

		rw.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(rw).Encode(blnResp); err != nil {
			http.Error(rw, err.Error(), statusInternal)
		}
	}
}

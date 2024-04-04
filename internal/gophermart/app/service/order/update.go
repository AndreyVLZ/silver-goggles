package order

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/order"
)

type errRetryable interface {
	error
	Retry() int64
}

type ordersUpdater interface {
	OrdersBatch(context.Context, []order.Status) ([]order.Order, error)
	OrdersUpdate(context.Context, []order.Order) error
}

type uService struct {
	accRepo   accrualRepo
	orderRepo ordersUpdater
	interval  int
	exit      chan (struct{})
	log       *slog.Logger
}

func NewUService(accRepo accrualRepo, orderRepo ordersUpdater, interval int, log *slog.Logger) uService {
	return uService{
		accRepo:   accRepo,
		orderRepo: orderRepo,
		interval:  interval,
		exit:      make(chan struct{}),
		log:       log,
	}
}

func (usrv uService) Name() string { return "uService" }

func (usrv uService) Stop() error {
	usrv.exit <- struct{}{}
	return nil
}

func (usrv uService) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-time.After(time.Duration(usrv.interval) * time.Second):
				if err := usrv.run(ctx); err != nil {
					var errRetry errRetryable
					if errors.As(err, &errRetry) {
						usrv.log.Debug("uService", "retry", errRetry.Retry())
						time.Sleep(time.Second * time.Duration(errRetry.Retry()))
					} else {
						usrv.log.Warn("uService", "err", err)
					}
				}
			case <-usrv.exit:
				cancel()
				return
			}
		}
	}()
	return nil
}

func (usrv uService) run(ctx context.Context) error {
	statuses := []order.Status{
		order.StatusNew,
		order.StatusProcessing,
	}

	orders, err := usrv.orderRepo.OrdersBatch(ctx, statuses)
	if err != nil {
		return err
	}

	if len(orders) == 0 {
		usrv.log.Debug("нет данных")
		return nil
	}

	resOrders, err := usrv.getUpdates(ctx, orders)
	if err != nil {
		return err
	}

	if err := usrv.orderRepo.OrdersUpdate(ctx, resOrders); err != nil {
		return err
	}

	return nil
}

// получение обновленной info для заказов
func (usrv uService) getUpdates(ctx context.Context, orders []order.Order) ([]order.Order, error) {
	resOrders := make([]order.Order, len(orders))
	for _, ordr := range orders {
		oi, err := usrv.accRepo.Load(ctx, ordr.Info().Number())
		if err != nil {
			return nil, err
		}

		if oi.Status().String() == ordr.Info().Status().String() {
			// обновлений нет
			continue
		}

		if err := ordr.UpdateInfo(oi); err != nil {
			return nil, err
		}

		resOrders = append(resOrders, ordr)
	}

	return resOrders, nil
}

package order

import (
	"context"
	"fmt"

	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/order"
	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/user"
)

type orderGetter interface {
	OrdersByStatuses(context.Context, user.ID, []order.Status) ([]order.Order, error)
}

type orderService struct {
	orderRepo orderGetter
}

func NewOrderService(orderRepo orderGetter) orderService {
	return orderService{orderRepo: orderRepo}
}

// GET /api/user/orders
func (srv orderService) GetOrders(ctx context.Context, userID user.ID) ([]order.Order, error) {
	statuses := []order.Status{
		order.StatusNew,
		order.StatusProcessing,
		order.StatusInvalid,
		order.StatusProcessed,
	}

	orders, err := srv.orderRepo.OrdersByStatuses(ctx, userID, statuses)
	if err != nil {
		return nil, fmt.Errorf("orderService GetOrders: %w", err)
	}

	return orders, nil
}

func (srv orderService) Withdrawals(ctx context.Context, userID user.ID) ([]order.Order, error) {
	statuses := []order.Status{order.StatusWithdraw}
	return srv.orderRepo.OrdersByStatuses(ctx, userID, statuses)
}

func (srv orderService) Balance(ctx context.Context, userID user.ID) (order.Balance, error) {
	statuses := []order.Status{order.StatusWithdraw, order.StatusProcessed}
	orders, err := srv.orderRepo.OrdersByStatuses(ctx, userID, statuses)
	if err != nil {
		return order.Balance{}, err
	}

	return getBalance(userID, orders), nil
}

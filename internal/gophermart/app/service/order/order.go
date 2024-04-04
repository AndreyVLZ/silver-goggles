package order

import (
	"context"

	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/order"
	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/user"
)

// repo
type _ interface {
	// USER
	SaveUser(context.Context, user.User) (user.User, error)
	UserByLogin(context.Context, user.Login) (user.User, error)
	// ORDER save
	SaveOrder(context.Context, order.Order) error
	// ORDER get save
	OrdersByStatuses(context.Context, user.ID, []order.Status) ([]order.Order, error)
	// ORDER update
	OrdersBatch(context.Context, []order.Status) ([]order.Order, error)
	OrdersUpdate(context.Context, []order.Order) error
}

type accrualRepo interface {
	Load(context.Context, order.Number) (order.Info, error)
}

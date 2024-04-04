package order

import (
	"context"
	"fmt"

	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/order"
	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/user"
	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/service/internal"
)

type orderSaver interface {
	SaveOrder(context.Context, order.Order) error
	OrdersByStatuses(context.Context, user.ID, []order.Status) ([]order.Order, error)
}

type saveService struct {
	accrualRepo accrualRepo
	orderRepo   orderSaver
}

type errWithdraw struct {
	current  float64
	withdraw float64
	orderAcc float64
}

func (e errWithdraw) IsSuccessful() bool {
	return (e.current - e.withdraw) <= e.orderAcc
}

func (e errWithdraw) Error() string {
	return fmt.Sprintf(
		"баланс [%v], запрос [%v]",
		e.current-e.withdraw, e.orderAcc,
	)
}
func NewSaveService(accrualRepo accrualRepo, orderRepo orderSaver) saveService {
	return saveService{
		accrualRepo: accrualRepo,
		orderRepo:   orderRepo,
	}
}

func (srv saveService) LoadOrder(ctx context.Context, userID user.ID, number order.Number) error {
	if !ValidNumber(int(number)) {
		return internal.ErrFieldConflict{Field: number, ErrStr: "number not valide"}
	}

	orderInfo, err := srv.accrualRepo.Load(ctx, number)
	if err != nil {
		return fmt.Errorf("LoadOrder: %w", err)
	}

	ordr := order.New(order.NewID(), userID, orderInfo)

	return srv.orderRepo.SaveOrder(ctx, ordr)
}

func getBalance(userID user.ID, orders []order.Order) order.Balance {
	var (
		procAcc order.Accrual
		withAcc order.Accrual
	)

	for _, ord := range orders {
		oi := ord.Info()
		switch oi.Status() {
		case order.StatusWithdraw:
			withAcc = withAcc.Add(oi.Accrual())
		case order.StatusProcessed:
			procAcc = withAcc.Add(oi.Accrual())
		}
	}

	total := procAcc - withAcc

	return order.NewBalance(userID, total, withAcc)
}

func (srv saveService) Withdraw(ctx context.Context, ordr order.Order) error {
	statuses := []order.Status{order.StatusWithdraw, order.StatusProcessed}

	orders, err := srv.orderRepo.OrdersByStatuses(ctx, ordr.UserID(), statuses)
	if err != nil {
		return err
	}

	orderAcc := ordr.Info().Accrual()

	balance := getBalance(ordr.UserID(), orders)

	accF := orderAcc.ToFloat()
	if (balance.Current().ToFloat() - balance.Withdraw().ToFloat()) <= accF {
		return errWithdraw{
			current:  balance.Current().ToFloat(),
			withdraw: balance.Withdraw().ToFloat(),
			orderAcc: orderAcc.ToFloat(),
		}
	}

	return srv.orderRepo.SaveOrder(ctx, ordr)
}

func ValidNumber(number int) bool {
	return (number%10+checksum(number/10))%10 == 0
}

func checksum(number int) int {
	var luhn int

	for i := 0; number > 0; i++ {
		cur := number % 10

		if i%2 == 0 {
			cur = cur * 2
			if cur > 9 {
				cur = cur%10 + cur/10
			}
		}

		luhn += cur
		number = number / 10
	}
	return luhn % 10
}

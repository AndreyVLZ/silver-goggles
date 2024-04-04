package order

import "github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/user"

type Balance struct {
	userID    user.ID
	current   Accrual
	withdrawn Accrual
}

func NewBalance(userID user.ID, current, withdrawn Accrual) Balance {
	return Balance{
		userID:    userID,
		current:   current,
		withdrawn: withdrawn,
	}
}

func (b *Balance) Current() Accrual  { return b.current }
func (b *Balance) Withdraw() Accrual { return b.withdrawn }

// func (b *Balance) AddCurrent(acc *Accrual)  { b.current = b.current.Add(acc) }
// func (b *Balance) AddWithdraw(acc *Accrual) { b.withdrawn = b.withdrawn.Add(acc) }

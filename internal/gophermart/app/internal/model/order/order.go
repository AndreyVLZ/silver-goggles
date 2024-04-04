package order

import "github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/user"

type Order struct {
	id        ID
	userID    user.ID
	orderInfo Info
}

func New(id ID, userID user.ID, orderInfo Info) Order {
	return Order{
		id:        id,
		userID:    userID,
		orderInfo: orderInfo,
	}
}

func (o *Order) UpdateInfo(newInfo Info) error {
	infoOrder, err := o.orderInfo.new(newInfo)
	if err != nil {
		return err
	}

	o.orderInfo = infoOrder

	return nil
}

func (o Order) ID() ID          { return o.id }
func (o Order) Info() Info      { return o.orderInfo }
func (o Order) UserID() user.ID { return o.userID }

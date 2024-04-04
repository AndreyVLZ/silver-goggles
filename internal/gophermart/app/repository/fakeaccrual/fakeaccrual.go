package fakeaccrual

import (
	"context"
	"math/rand"
	"slices"
	"time"

	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/order"
)

func randInt(min, max int) int {
	r := rand.New(
		rand.NewSource(
			time.Now().UnixNano(),
		),
	)

	return min + 1 + r.Intn(max-min)
}

func randFloat(min, max float64) float64 {
	r := rand.New(
		rand.NewSource(
			time.Now().UnixNano(),
		),
	)
	return min + r.Float64()*(max-min)
}

func finalStatuses() []order.Status {
	return []order.Status{
		//order.StatusInvalid,
		order.StatusProcessed,
	}
}

type fakeAccrual struct{}

func (fa fakeAccrual) Load(_ context.Context, number order.Number) (order.Info, error) {
	var accrual *order.Accrual

	status := order.Status(randInt(int(order.StatusRegistered), int(order.StatusProcessed)))

	if slices.Contains(finalStatuses(), status) {
		accrual = order.NewAccrual(randFloat(1, 100))
	}

	return order.NewInfo(number, status, accrual), nil
}

func (fa fakeAccrual) Name() string { return "fake Accraul" }

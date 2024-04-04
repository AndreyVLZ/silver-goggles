package order

type Accrual uint64

func NewAccrual(acc float64) *Accrual {
	a := Accrual(acc * 100)
	return &a
}

func ParseAccrual(accFloat *float64) *Accrual {
	if accFloat == nil {
		return nil
	}

	acc := NewAccrual(*accFloat)
	return acc
}

func (acc Accrual) Add(accr *Accrual) Accrual {
	if accr != nil {
		acc += *accr
	}

	return acc
}

func (acc Accrual) ToFloat() float64 { return float64(acc) / 100 }

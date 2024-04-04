package order

import "time"

type Info struct {
	number   Number
	status   Status
	accrual  *Accrual
	uploaded time.Time
}

func ParseInfo(numberStr string, statusStr string, acc *float64, date time.Time) (Info, error) {
	number, err := ParseNumber(numberStr)
	if err != nil {
		return Info{}, err
	}

	return Info{
		number:   number,
		status:   ParseStatus(statusStr),
		accrual:  ParseAccrual(acc),
		uploaded: date,
	}, nil
}

func NewInfo(number Number, status Status, acc *Accrual) Info {
	return Info{
		number:   number,
		status:   status,
		accrual:  acc,
		uploaded: time.Now(),
	}
}

func (i Info) new(newInfo Info) (Info, error) {
	return Info{
		number:   newInfo.number,
		status:   newInfo.status,
		accrual:  newInfo.accrual,
		uploaded: newInfo.uploaded,
	}, nil
}

func (i Info) Accrual() *Accrual { return i.accrual }
func (i Info) Status() Status    { return i.status }
func (i Info) Number() Number    { return i.number }
func (i Info) Date() time.Time   { return i.uploaded }

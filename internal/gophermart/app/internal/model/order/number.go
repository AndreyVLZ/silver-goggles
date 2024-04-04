package order

import "strconv"

type Number uint64

func ParseNumber(numStr string) (Number, error) {
	n, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, err
	}

	return Number(n), nil
}

func (n Number) String() string { return strconv.Itoa(int(n)) }

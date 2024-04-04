package hash

import "github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/user"

type hashe struct {
	fnHash   func([]byte) ([]byte, error)
	fnEquals func([]byte, []byte) error
}

func NewHashe(fnHash func([]byte) ([]byte, error), fnEquals func([]byte, []byte) error) hashe {
	return hashe{fnHash: fnHash, fnEquals: fnEquals}
}

func (hf hashe) Hash(pass user.Password) (string, error) {
	hashPass, err := hf.fnHash([]byte(pass))
	if err != nil {
		return "", err
	}
	return string(hashPass), nil
}

func (hf hashe) Equals(hashPass string, pass user.Password) bool {
	if err := hf.fnEquals([]byte(hashPass), []byte(pass)); err != nil {
		return false
	}
	return true
}

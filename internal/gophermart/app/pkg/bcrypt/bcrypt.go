package bcrypt

import "golang.org/x/crypto/bcrypt"

func Hash(pass []byte) ([]byte, error) {
	return bcrypt.GenerateFromPassword(pass, bcrypt.DefaultCost)

}

func Compare(hashPass, pass []byte) error {
	return bcrypt.CompareHashAndPassword(hashPass, pass)
}

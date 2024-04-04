package user

import (
	"errors"
	"fmt"
)

type Login string
type Password string

type User struct {
	id    ID
	login Login
	hash  string
}

func (u User) Valid() error {
	if u.login == "" {
		return errors.New("login empty")
	}

	if u.id == NillID() {
		return errors.New("id empty")
	}

	if u.hash == "" {
		return errors.New("hash empty")
	}

	return nil
}

func Parse(idStr string, login string, hash string) (User, error) {
	id, err := ParseID(idStr)
	if err != nil {
		return User{}, fmt.Errorf("New: %w", err)
	}

	return User{
		id:    id,
		login: Login(login),
		hash:  hash,
	}, nil
}

func New(login Login, hash string) User {
	return User{
		id:    NewID(),
		login: login,
		hash:  hash,
	}
}

func (u User) ID() ID        { return u.id }
func (u User) Login() string { return string(u.login) }
func (u User) Hash() string  { return string(u.hash) }

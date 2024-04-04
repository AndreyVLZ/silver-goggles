package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/user"
	"github.com/stretchr/testify/assert"
)

type fakeHasher struct {
	err     error
	hashVal string
}

func (fh fakeHasher) Hash(user.Password) (string, error) {
	if fh.err != nil {
		return "", fh.err
	}
	return fh.hashVal, nil
}

func (fh fakeHasher) Equals(string, user.Password) bool { return false }

type fakeUserRepo struct {
	err error
}

func (fRepo fakeUserRepo) SaveUser(_ context.Context, u user.User) (user.User, error) {
	if fRepo.err != nil {
		return user.User{}, fRepo.err
	}
	return u, nil
}

func (fRepo fakeUserRepo) UserByLogin(context.Context, user.Login) (user.User, error) {
	return user.User{}, nil
}

func TestRegister(t *testing.T) {
	ctx := context.TODO()

	type testCase struct {
		name     string
		login    user.Login
		pass     user.Password
		userRes  user.User
		hash     string
		userRepo userRepo
		hasher   hasher
		isErr    bool
	}

	tc := []testCase{
		{
			name:     "#1 not error",
			login:    user.Login("Login"),
			pass:     user.Password("pass"),
			hash:     "hash",
			userRepo: fakeUserRepo{},
			hasher:   fakeHasher{hashVal: "hash"},
			isErr:    false,
		},
		{
			name:     "#2 hash error",
			login:    user.Login("Login"),
			pass:     user.Password("pass"),
			userRepo: fakeUserRepo{},
			hasher:   fakeHasher{err: errors.New("hashe error")},
			isErr:    true,
		},
	}

	for _, test := range tc {
		t.Run(test.name, func(t *testing.T) {
			authUser := user.NewAuthUser(test.login, test.pass)
			authSrv := NewAuthService(test.userRepo, test.hasher)

			userRes, err := authSrv.Register(ctx, authUser)
			if test.isErr {
				if err == nil {
					t.Errorf("err not equals [%v]!=[%v]", err, test.isErr)
				}
				return
			}
			assert.Equal(t, userRes.Login(), string(test.login))
			assert.Equal(t, userRes.Hash(), test.hash)
		})
	}
}

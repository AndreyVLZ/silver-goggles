package auth

import (
	"context"
	"fmt"

	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/user"
	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/service/internal"
)

// AUTH
type userRepo interface {
	SaveUser(context.Context, user.User) (user.User, error)
	UserByLogin(context.Context, user.Login) (user.User, error)
}

type hasher interface {
	Hash(user.Password) (string, error)
	Equals(string, user.Password) bool
}

type authService struct {
	userRepo userRepo
	hasher   hasher
}

func NewAuthService(userRepo userRepo, hasher hasher) authService {
	return authService{userRepo: userRepo, hasher: hasher}
}

func (srv authService) Register(ctx context.Context, authUser user.AuthUser) (user.User, error) {
	hashPass, err := srv.hasher.Hash(authUser.Pass())
	if err != nil {
		return user.User{}, err
	}

	userToSave := user.New(authUser.Login(), hashPass)
	if err := userToSave.Valid(); err != nil {
		return user.User{}, fmt.Errorf("authService Register: %w", err)
	}

	usr, err := srv.userRepo.SaveUser(ctx, userToSave)
	if err != nil {
		return user.User{}, err
	}

	return usr, nil
}

func (srv authService) Login(ctx context.Context, authUser user.AuthUser) (user.User, error) {
	usr, err := srv.userRepo.UserByLogin(ctx, authUser.Login())
	if err != nil {
		return user.User{}, fmt.Errorf("authService Login: %w", err)
	}

	if err := usr.Valid(); err != nil {
		return user.User{}, internal.ErrFieldConflict{
			Field:  usr.Login(),
			ErrStr: err.Error(),
		}
	}

	if !srv.hasher.Equals(usr.Hash(), authUser.Pass()) {
		return user.User{}, internal.ErrFieldConflict{
			Field:  authUser.Pass(),
			ErrStr: "неверный пароль",
		}
	}

	return usr, nil
}

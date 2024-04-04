package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/user"
)

// USER
type authService interface {
	Register(context.Context, user.AuthUser) (user.User, error)
	Login(context.Context, user.AuthUser) (user.User, error)
}

type UserHandler struct {
	authService authService
	fnSet       func(http.ResponseWriter, string) // header or cockie
	log         *slog.Logger
}

func NewUserHandler(authService authService, fnSet func(http.ResponseWriter, string), log *slog.Logger) UserHandler {
	return UserHandler{authService: authService, fnSet: fnSet, log: log}
}

type userReq struct {
	Login string `json:"login"`
	Pass  string `json:"password"`
}

func (ur userReq) Valid() bool { return ur.Login != "" || ur.Pass != "" }

func (ur userReq) toAuthUser() user.AuthUser {
	return user.NewAuthUser(user.Login(ur.Login), user.Password(ur.Pass))
}

// Регистрация пользователя
// POST /api/user/register
func (uh UserHandler) Register() http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		var userReq userReq

		if err := parseBody(req.Body, &userReq); err != nil {
			uh.log.Error("parse Body", "err", err)
			http.Error(rw, err.Error(), statusBadReq)
			return
		}

		if !userReq.Valid() {
			uh.log.Error("userReq not valid", "req", userReq)
			http.Error(rw, "userReq not valid", statusBadReq)
			return
		}

		userFromDB, err := uh.authService.Register(req.Context(), userReq.toAuthUser())
		if err != nil {
			var errFieldConf errFieldConflict
			if errors.As(err, &errFieldConf) {
				uh.log.Error(errFieldConf.Error())
				http.Error(rw, errFieldConf.Error(), statusConflict)
				return
			}

			uh.log.Error("err service", "err", err)
			http.Error(rw, err.Error(), statusInternal)
			return
		}

		uh.fnSet(rw, userFromDB.ID().String())
	}
}

// Аутентификация пользователя
// POST /api/user/login
func (uh UserHandler) Login() http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		var userReq userReq
		if err := parseBody(req.Body, &userReq); err != nil {
			uh.log.Error("parse Body", "err", err)
			http.Error(rw, err.Error(), statusInternal)
			return
		}

		if !userReq.Valid() {
			uh.log.Error("userReq not valid", "req", userReq)
			http.Error(rw, "userReq not valid", statusBadReq)
			return
		}

		userFromDB, err := uh.authService.Login(req.Context(), userReq.toAuthUser())
		if err != nil {
			uh.log.Error("loginSrv", "err", err)
			http.Error(rw, err.Error(), statusUnauth)
			return
		}

		uh.fnSet(rw, userFromDB.ID().String())
	}
}

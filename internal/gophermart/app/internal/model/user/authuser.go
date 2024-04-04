package user

// User для аутентификации
type AuthUser struct {
	login    Login
	password Password
}

func NewAuthUser(login Login, pass Password) AuthUser {
	return AuthUser{
		login:    login,
		password: pass,
	}
}

func (au AuthUser) Login() Login   { return au.login }
func (au AuthUser) Pass() Password { return au.password }

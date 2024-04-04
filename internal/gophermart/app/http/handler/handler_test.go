package handler

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/order"
	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/user"
)

type fakeAuther struct {
	dataSet string
}

func (fauth *fakeAuther) Set(rw http.ResponseWriter, data string) {
	fauth.dataSet = data
}

func (fauth fakeAuther) Check(req *http.Request) (string, error) {
	return "", nil
}

type fakeAuthSrv struct {
	usr user.User
	err error
}

func (fsrv fakeAuthSrv) Register(_ context.Context, authUser user.AuthUser) (user.User, error) {
	if fsrv.err != nil {
		return user.User{}, fsrv.err
	}
	return fsrv.usr, nil
}

func (fsrv fakeAuthSrv) Login(context.Context, user.AuthUser) (user.User, error) {
	if fsrv.err != nil {
		return user.User{}, fsrv.err
	}
	return fsrv.usr, nil
}

type errField struct {
	data any
	error
}

func (err errField) Data() any { return err.data }

func TestRegister(t *testing.T) {
	type testCase struct {
		name    string
		usr     user.User
		body    string
		method  string
		authSrv fakeAuthSrv
		status  int
	}

	tc := []testCase{
		{
			name:    "#1 not error",
			usr:     user.New(user.Login("LOGIN"), "hash"),
			authSrv: fakeAuthSrv{},
			body: fmt.Sprintf(
				`{"login":"%s","password":"%s"}`,
				"LOGIN", "PASS"),
			method: http.MethodPost,
			status: http.StatusOK, // 200
		},

		{
			name:    "#2 невалидный боди",
			usr:     user.New(user.Login("LOGIN"), "hash"),
			body:    "{{",
			authSrv: fakeAuthSrv{},
			method:  http.MethodPost,
			status:  http.StatusBadRequest, //400
		},

		{
			name: "#3 пустой user в запросе",
			usr:  user.User{},
			body: fmt.Sprintf(
				`{"login":"%s","password":"%s"}`,
				"", ""),
			authSrv: fakeAuthSrv{},
			method:  http.MethodPost,
			status:  http.StatusBadRequest, //400
		},

		{
			name: "#4 ошибка err-field-conflict (логин занят)",
			usr:  user.New(user.Login("LOGIN"), "hash"),
			body: fmt.Sprintf(
				`{"login":"%s","password":"%s"}`,
				"LOGIN", "PASS"),
			authSrv: fakeAuthSrv{
				err: errField{ // имитация ошибки, когда login занят
					error: errors.New("логин занят"),
					data:  user.User{},
				},
			},
			method: http.MethodPost,
			status: http.StatusConflict, // 409
		},

		{
			name: "#5 ошибка-2 err-field-conflict",
			usr:  user.New(user.Login("LOGIN"), "hash"),
			body: fmt.Sprintf(
				`{"login":"%s","password":"%s"}`,
				"LOGIN", "PASS"),
			authSrv: fakeAuthSrv{ // любая другая ошибка от service
				err: errors.New("другая ошибка"),
			},
			method: http.MethodPost,
			status: http.StatusInternalServerError, // 500
		},
	}

	l := slog.New(slog.NewTextHandler(os.Stdout, nil))
	for _, test := range tc {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(
				test.method, "/api/user/register",
				strings.NewReader(test.body))

			fakeAuther := fakeAuther{}
			userID := test.usr.ID()

			test.authSrv.usr = test.usr
			uh := NewUserHandler(test.authSrv, fakeAuther.Set, l)
			rw := httptest.NewRecorder()

			uh.Register().ServeHTTP(rw, req)
			res := rw.Result()
			defer res.Body.Close()

			if res.StatusCode != test.status {
				t.Errorf("код ответа неверный [%d]!=[%d]", res.StatusCode, test.status)
				return
			}

			_, err := io.ReadAll(res.Body)
			if err != nil {
				t.Error(err)
			}

			// проверяем, что сработала функция установки заголовка
			if userID.String() != test.authSrv.usr.ID().String() {
				t.Errorf("данные на аутентификацию не отправлены [%s]!=[%s]", userID.String(), test.authSrv.usr.ID().String())
				return
			}
		})
	}
}

func TestLogin(t *testing.T) {
	type testCase struct {
		name    string
		body    string
		authSrv fakeAuthSrv
		status  int
	}

	method := http.MethodPost
	tc := []testCase{
		{
			name: "#1 not err",
			body: fmt.Sprintf(
				`{"login":"%s","password":"%s"}`,
				"LOGIN", "PASS"),
			authSrv: fakeAuthSrv{
				usr: user.New(user.Login("LOGIN"), "hash"),
			},
			status: http.StatusOK,
		},

		{
			name: "#2 невалидный body",
			body: "{",
			authSrv: fakeAuthSrv{
				usr: user.New(user.Login("LOGIN"), "hash"),
			},
			status: http.StatusInternalServerError,
		},

		{
			name: "#3 невалидный запрос",
			body: fmt.Sprintf(
				`{"login":"%s","password":"%s"}`,
				"", ""),
			authSrv: fakeAuthSrv{
				usr: user.New(user.Login("LOGIN"), "hash"),
			},
			status: http.StatusBadRequest,
		},

		{
			name: "#4 ",
			body: fmt.Sprintf(
				`{"login":"%s","password":"%s"}`,
				"LOGIN", "PASS"),
			authSrv: fakeAuthSrv{
				usr: user.New(user.Login("LOGIN"), "hash"),
				err: errField{
					data:  user.Password(""),
					error: errors.New("неверный логин/пароль"),
				},
			},
			status: http.StatusUnauthorized,
		},
	}

	fakeAuther := fakeAuther{}
	l := slog.New(slog.NewTextHandler(os.Stdout, nil))
	for _, test := range tc {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(
				method, "/api/user/login",
				strings.NewReader(test.body),
			)

			userID := test.authSrv.usr.ID()

			rw := httptest.NewRecorder()

			uh := NewUserHandler(test.authSrv, fakeAuther.Set, l)
			uh.Login().ServeHTTP(rw, req)

			res := rw.Result()
			defer res.Body.Close()

			// проверяем код ответа
			if res.StatusCode != test.status {
				t.Errorf("код ответа неверный acctual [%d] expec [%d]",
					res.StatusCode, test.status)
				return
			}

			if res.StatusCode != http.StatusOK {
				return
			}

			// проверяем, что сработала функция установки заголовка
			if userID.String() != fakeAuther.dataSet {
				t.Errorf("данные на аутентификацию не отправлены [%s]!=[%s]", userID.String(), fakeAuther.dataSet)
				return
			}
		})
	}
}

type fakeRead struct {
	error
}

func (r fakeRead) Read(p []byte) (n int, err error) {
	return 1, r.error
}

type fakeSaveOrderSrv struct {
	err error
}

func (fsrv fakeSaveOrderSrv) LoadOrder(context.Context, user.ID, order.Number) error {
	return fsrv.err
}

func (fsrv fakeSaveOrderSrv) Withdraw(context.Context, order.Order) error {
	return fsrv.err
}

func TestLoadHandle(t *testing.T) {
	method := http.MethodPost

	type testCase struct {
		name    string
		read    io.Reader
		userID  any
		status  int
		saveSrv saveOrderService
	}

	tc := []testCase{
		{
			name:    "#1 no errors",
			userID:  user.NewID(), // valid userID
			read:    strings.NewReader("123"),
			saveSrv: fakeSaveOrderSrv{},
			status:  http.StatusAccepted,
		},

		{
			name:    "#2 not valid userID",
			userID:  "0000", // not valid userID
			read:    strings.NewReader("123"),
			saveSrv: fakeSaveOrderSrv{},
			status:  http.StatusInternalServerError,
		},

		{
			name:    "#3 err read body",
			userID:  user.NewID(),
			read:    fakeRead{error: errors.New("implement err reader")},
			saveSrv: fakeSaveOrderSrv{},
			status:  http.StatusInternalServerError,
		},

		{
			name:    "#4 err parse number",
			userID:  user.NewID(),
			read:    strings.NewReader("a23"),
			saveSrv: fakeSaveOrderSrv{},
			status:  http.StatusUnprocessableEntity,
		},

		{
			name:   "#5 number not valid",
			userID: user.NewID(),
			read:   strings.NewReader("123"),
			saveSrv: fakeSaveOrderSrv{
				err: errField{
					data:  order.Number(1),
					error: errors.New("номер не прошел валидацию"),
				},
			},
			status: http.StatusUnprocessableEntity,
		},

		{
			name:   "#6 number upload another user",
			userID: user.NewID(),
			read:   strings.NewReader("123"),
			saveSrv: fakeSaveOrderSrv{
				err: errField{
					data:  user.NewID(),
					error: errors.New("заказ уже загружен другим пользователем"),
				},
			},
			status: http.StatusConflict,
		},

		{
			name:   "#6 number exist",
			read:   strings.NewReader("123"),
			userID: user.NewID(),
			saveSrv: fakeSaveOrderSrv{
				err: errField{
					data:  order.NewID(),
					error: errors.New("заказ уже загружен жтим пользователем"),
				},
			},
			status: http.StatusOK,
		},

		{
			name:   "#7 -",
			read:   strings.NewReader("123"),
			userID: user.NewID(),
			saveSrv: fakeSaveOrderSrv{
				err: errors.New("ошибка сервиса"),
			},
			status: http.StatusInternalServerError,
		},
	}

	l := slog.New(slog.NewTextHandler(os.Stdout, nil))
	for _, test := range tc {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(
				method, "/api/user/orders",
				test.read,
			)

			req = req.WithContext(
				context.WithValue(
					context.Background(),
					GetContextKey(), test.userID,
				),
			)

			oh := NewOrderHandler(test.saveSrv, nil, l)
			rw := httptest.NewRecorder()
			oh.LoadHandle().ServeHTTP(rw, req)

			res := rw.Result()
			defer res.Body.Close()

			// проверяем код ответа
			if res.StatusCode != test.status {
				t.Errorf("код ответа неверный acctual [%d] expec [%d]",
					res.StatusCode, test.status)
				return
			}
		})
	}
}

type errWithd struct {
	error
}

func (e errWithd) IsSuccessful() bool { return true }

func TestWithdraw(t *testing.T) {
	method := http.MethodPost

	type testCase struct {
		name    string
		body    io.Reader
		userID  any
		status  int
		saveSrv saveOrderService
	}

	tc := []testCase{
		{
			name:   "#1 no errors",
			userID: user.NewID(), // valid userID
			body: strings.NewReader(
				`{"order":"123","sum":321}`,
			),
			saveSrv: fakeSaveOrderSrv{},
			status:  http.StatusOK,
		},

		{
			name:   "#2 not valid userID",
			userID: "0000", // not valid userID
			body: strings.NewReader(
				`{"order":"123","sum":321}`,
			),
			saveSrv: fakeSaveOrderSrv{},
			status:  http.StatusInternalServerError,
		},

		{
			name:    "#3 err read body",
			userID:  user.NewID(),
			body:    fakeRead{error: errors.New("implement err reader")},
			saveSrv: fakeSaveOrderSrv{},
			status:  http.StatusInternalServerError,
		},

		{
			name:   "#4 err parse number",
			userID: user.NewID(),
			body: strings.NewReader(
				`{"order":"a123","sum":321}`,
			),
			saveSrv: fakeSaveOrderSrv{},
			status:  http.StatusUnprocessableEntity,
		},

		{
			name:   "#5 not succesful",
			userID: user.NewID(),
			body: strings.NewReader(
				`{"order":"123","sum":321}`,
			),
			saveSrv: fakeSaveOrderSrv{
				err: errWithd{error: fmt.Errorf("баланс меньше")},
			},
			status: http.StatusPaymentRequired,
		},

		{
			name:   "#6 -",
			userID: user.NewID(),
			body: strings.NewReader(
				`{"order":"123","sum":321}`,
			),
			saveSrv: fakeSaveOrderSrv{
				err: errors.New("ошибка сервиса"),
			},
			status: http.StatusInternalServerError,
		},
	}

	l := slog.New(slog.NewTextHandler(os.Stdout, nil))
	for _, test := range tc {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(
				method, "/api/user/balance/withdraw",
				test.body,
			).WithContext(
				context.WithValue(
					context.Background(),
					GetContextKey(), test.userID,
				),
			)

			oh := NewOrderHandler(test.saveSrv, nil, l)
			rw := httptest.NewRecorder()
			oh.Withdraw().ServeHTTP(rw, req)

			res := rw.Result()
			defer res.Body.Close()

			// проверяем код ответа
			if res.StatusCode != test.status {
				t.Errorf("код ответа неверный acctual [%d] expec [%d]",
					res.StatusCode, test.status)
				return
			}
		})
	}
}

type fakeGetOrderSrv struct {
	infos []order.Info
	cur   *order.Accrual
	with  *order.Accrual
	error
}

func (fsrv fakeGetOrderSrv) GetOrders(context.Context, user.ID) ([]order.Order, error) {
	if fsrv.error != nil {
		return nil, fsrv.error
	}

	var ordrs []order.Order
	for _, oi := range fsrv.infos {
		ordrs = append(ordrs, order.New(order.NewID(), user.NewID(), oi))
	}

	return ordrs, nil
}

func (fsrv fakeGetOrderSrv) Withdrawals(context.Context, user.ID) ([]order.Order, error) {
	if fsrv.error != nil {
		return nil, fsrv.error
	}

	var ordrs []order.Order
	for _, oi := range fsrv.infos {
		ordrs = append(ordrs, order.New(order.NewID(), user.NewID(), oi))
	}

	return ordrs, nil
}

func (fsrv fakeGetOrderSrv) Balance(_ context.Context, userID user.ID) (order.Balance, error) {
	if fsrv.error != nil {
		return order.Balance{}, fsrv.error
	}

	bal := order.NewBalance(userID, *fsrv.cur, *fsrv.with)
	return bal, nil
}

func TestGetOrders(t *testing.T) {
	type orderData struct {
		Numer  string   `json:"number"`
		Status string   `json:"status"`
		Acc    *float64 `json:"accrual,omitempty"`
		Date   string   `json:"uploaded_at"`
	}

	type testCase struct {
		name        string
		userID      any
		status      int
		ordrSrv     fakeGetOrderSrv
		contentType string
	}

	tc := []testCase{
		{
			name:   "#1 not errors",
			userID: user.NewID(),
			ordrSrv: fakeGetOrderSrv{
				infos: []order.Info{
					order.NewInfo(
						order.Number(1),
						order.StatusNew,
						nil,
					),
					order.NewInfo(
						order.Number(2),
						order.StatusWithdraw,
						order.NewAccrual(20),
					),
				},
			},
			status:      http.StatusOK,
			contentType: "application/json",
		},

		{
			name:    "#2 not valid userID",
			userID:  "000",
			ordrSrv: fakeGetOrderSrv{},
			status:  http.StatusInternalServerError,
		},

		{
			name:    "#3 service by error",
			userID:  user.NewID(),
			ordrSrv: fakeGetOrderSrv{error: errors.New("err get servise")},
			status:  http.StatusInternalServerError,
		},

		{
			name:    "#4 empty orders",
			userID:  user.NewID(),
			ordrSrv: fakeGetOrderSrv{infos: []order.Info{}},
			status:  http.StatusNoContent,
		},
	}

	l := slog.New(slog.NewTextHandler(os.Stdout, nil))
	for _, test := range tc {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet, "/api/user/orders",
				http.NoBody,
			).WithContext(
				context.WithValue(
					context.Background(),
					GetContextKey(), test.userID,
				),
			)

			oh := NewOrderHandler(nil, test.ordrSrv, l)
			rw := httptest.NewRecorder()
			oh.GetOrders().ServeHTTP(rw, req)

			res := rw.Result()
			defer res.Body.Close()

			// проверяем код ответа
			if res.StatusCode != test.status {
				t.Errorf("код ответа неверный acctual [%d] expec [%d]",
					res.StatusCode, test.status)
				return
			}

			if test.status != http.StatusOK {
				return
			}

			var data []orderData
			if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
				t.Errorf("decode body [%s]\n", err)
				return
			}

			if cmpRes := slices.CompareFunc(
				test.ordrSrv.infos, data,
				func(oi order.Info, oData orderData) int {
					var cmpAcc int
					if oi.Accrual() != nil {
						cmpAcc = cmp.Compare(oi.Accrual().ToFloat(), *oData.Acc)
					}
					cmpDate := cmp.Compare(oi.Date().String(), oData.Date)
					cmpStatus := cmp.Compare(oi.Status().String(), oData.Status)
					cmpNumber := cmp.Compare(oi.Number().String(), oData.Numer)
					if cmpAcc != 0 && cmpStatus != 0 && cmpNumber != 0 && cmpDate != 0 {
						return 1
					}
					return 0
				}); cmpRes != 0 {
				t.Errorf("body no compare [%v]", data)
			}

			// проверяем content-type
			if res.Header.Get("Content-Type") != test.contentType {
				t.Errorf("код ответа неверный acctual [%d] expec [%d]",
					res.StatusCode, test.status)
				return
			}
		})
	}
}

func TestWithdrawals(t *testing.T) {
	type orderData struct {
		Order string  `json:"order"`
		Sum   float64 `json:"sum"`
		Date  string  `json:"processed_at"`
	}

	type testCase struct {
		name        string
		userID      any
		status      int
		ordrSrv     fakeGetOrderSrv
		contentType string
	}

	tc := []testCase{
		{
			name:   "#1 not errors",
			userID: user.NewID(),
			ordrSrv: fakeGetOrderSrv{
				infos: []order.Info{
					order.NewInfo(
						order.Number(1),
						order.StatusNew,
						order.NewAccrual(10),
					),
					order.NewInfo(
						order.Number(2),
						order.StatusWithdraw,
						order.NewAccrual(20),
					),
				},
			},
			status:      http.StatusOK,
			contentType: "application/json",
		},

		{
			name:    "#2 not valid userID",
			userID:  "000",
			ordrSrv: fakeGetOrderSrv{},
			status:  http.StatusInternalServerError,
		},

		{
			name:    "#3 service by error",
			userID:  user.NewID(),
			ordrSrv: fakeGetOrderSrv{error: errors.New("err get servise")},
			status:  http.StatusInternalServerError,
		},

		{
			name:    "#4 empty orders",
			userID:  user.NewID(),
			ordrSrv: fakeGetOrderSrv{infos: []order.Info{}},
			status:  http.StatusNoContent,
		},
	}

	l := slog.New(slog.NewTextHandler(os.Stdout, nil))
	for _, test := range tc {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet, "/api/user/withdrawals",
				http.NoBody,
			).WithContext(
				context.WithValue(
					context.Background(),
					GetContextKey(), test.userID,
				),
			)

			oh := NewOrderHandler(nil, test.ordrSrv, l)
			rw := httptest.NewRecorder()
			oh.Withdrawals().ServeHTTP(rw, req)

			res := rw.Result()
			defer res.Body.Close()

			// проверяем код ответа
			if res.StatusCode != test.status {
				t.Errorf("код ответа неверный acctual [%d] expec [%d]",
					res.StatusCode, test.status)
				return
			}

			if res.StatusCode != http.StatusOK {
				return
			}

			var data []orderData
			if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
				t.Errorf("decode body [%s]\n", err)
				return
			}

			if cmpRes := slices.CompareFunc(
				test.ordrSrv.infos, data,
				func(oi order.Info, oData orderData) int {
					var cmpAcc int
					if oi.Accrual() != nil {
						cmpAcc = cmp.Compare(oi.Accrual().ToFloat(), oData.Sum)
					}
					cmpDate := cmp.Compare(oi.Date().String(), oData.Date)
					cmpNumber := cmp.Compare(oi.Number().String(), oData.Order)
					if cmpAcc != 0 && cmpNumber != 0 && cmpDate != 0 {
						return 1
					}
					return 0
				}); cmpRes != 0 {
				t.Errorf("body no compare [%v]", data)
				return
			}

			// проверяем content-type
			if res.Header.Get("Content-Type") != test.contentType {
				t.Errorf("код ответа неверный acctual [%d] expec [%d]",
					res.StatusCode, test.status)
				return
			}
		})
	}
}

func TestBalance(t *testing.T) {
	type balanceResp struct {
		Current   float64 `json:"current"`
		Withdrawn float64 `json:"withdrawn"`
	}

	type testCase struct {
		name        string
		userID      any
		status      int
		ordrSrv     fakeGetOrderSrv
		contentType string
	}

	tc := []testCase{
		{
			name:   "#1 not errors",
			userID: user.NewID(),
			ordrSrv: fakeGetOrderSrv{
				cur:  order.NewAccrual(10),
				with: order.NewAccrual(100),
			},
			status:      http.StatusOK,
			contentType: "application/json",
		},

		{
			name:    "#2 not valid userID",
			userID:  "000",
			ordrSrv: fakeGetOrderSrv{},
			status:  http.StatusInternalServerError,
		},

		{
			name:    "#3 service by error",
			userID:  user.NewID(),
			ordrSrv: fakeGetOrderSrv{error: errors.New("err get servise")},
			status:  http.StatusInternalServerError,
		},
	}

	l := slog.New(slog.NewTextHandler(os.Stdout, nil))
	for _, test := range tc {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(
				http.MethodGet, "/api/user/balance",
				http.NoBody,
			).WithContext(
				context.WithValue(
					context.Background(),
					GetContextKey(), test.userID,
				),
			)

			oh := NewOrderHandler(nil, test.ordrSrv, l)
			rw := httptest.NewRecorder()
			oh.Balance().ServeHTTP(rw, req)

			res := rw.Result()
			defer res.Body.Close()

			// проверяем код ответа
			if res.StatusCode != test.status {
				t.Errorf("код ответа неверный acctual [%d] expec [%d]",
					res.StatusCode, test.status)
				return
			}

			if res.StatusCode != http.StatusOK {
				return
			}

			var blnResp balanceResp
			if err := json.NewDecoder(res.Body).Decode(&blnResp); err != nil {
				t.Errorf("decode body [%s]\n", err)
				return
			}

			cmpCur := cmp.Compare(blnResp.Current, test.ordrSrv.cur.ToFloat())
			cmpWith := cmp.Compare(blnResp.Withdrawn, test.ordrSrv.with.ToFloat())
			if cmpCur != 0 || cmpWith != 0 {
				t.Errorf("body no compare [%v]", blnResp)
				return
			}

			// проверяем content-type
			if res.Header.Get("Content-Type") != test.contentType {
				t.Errorf("код ответа неверный acctual [%d] expec [%d]",
					res.StatusCode, test.status)
				return
			}
		})
	}
}

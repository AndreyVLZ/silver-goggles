package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

var (
	statusOK         = http.StatusOK                  // 200
	statusAccepted   = http.StatusAccepted            // 202
	statusNoContent  = http.StatusNoContent           // 204
	statusBadReq     = http.StatusBadRequest          // 400
	statusUnauth     = http.StatusUnauthorized        // 401
	statusPaymentReq = http.StatusPaymentRequired     // 402
	statusConflict   = http.StatusConflict            // 409
	statusUnprocess  = http.StatusUnprocessableEntity // 422
	statusInternal   = http.StatusInternalServerError // 500
)

type errFieldConflict interface {
	Data() any
	error
}

type AuthByCookie struct {
	cookieKey string
}

func NewAuthBycookie(cookieKey string) AuthByCookie {
	return AuthByCookie{cookieKey: cookieKey}
}

func (auth AuthByCookie) Check(req *http.Request) (string, error) {
	reqCookie, err := req.Cookie(auth.cookieKey)
	if err != nil {
		return "", err
	}
	slog.Debug("authByCockie", "value", reqCookie.Value, "key", auth.cookieKey)

	return reqCookie.Value, nil
}

func (auth AuthByCookie) Set(rw http.ResponseWriter, data string) {
	slog.Debug("authByCockie", "value", data, "key", auth.cookieKey)
	http.SetCookie(rw, &http.Cookie{
		Name:  auth.cookieKey,
		Value: data,
		Path:  "/",
	})
}

type AuthByHeader struct {
	headerKey string
}

func NewAuthByHeader(headerKey string) AuthByHeader {
	return AuthByHeader{headerKey: headerKey}
}

func (auth AuthByHeader) Check(req *http.Request) (string, error) {
	if idVal := req.Header.Get(auth.headerKey); idVal != "" {
		return idVal, nil
	}
	return "", fmt.Errorf("err: parseHeader [%s]", auth.headerKey)
}

func (auth AuthByHeader) Set(rw http.ResponseWriter, data string) {
	rw.Header().Set(auth.headerKey, data)
}

func parseBody(r io.ReadCloser, data any) error {
	defer r.Close()
	return json.NewDecoder(r).Decode(data)
}

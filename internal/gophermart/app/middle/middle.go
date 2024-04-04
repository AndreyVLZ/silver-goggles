package middle

import (
	"context"
	"net/http"

	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/user"
)

const appJSON string = "application/json"
const textPlain string = "text/plain"

type middle func(http.Handler) http.Handler

type Select struct {
	Get  http.Handler
	Post http.Handler
}

func (s Select) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	switch {
	case req.Method == http.MethodGet && s.Get != nil:
		s.Get.ServeHTTP(rw, req)
	case req.Method == http.MethodPost && s.Post != nil:
		s.Post.ServeHTTP(rw, req)
	default:
		rw.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func Use(use http.HandlerFunc, arrMiddle ...middle) http.Handler {
	for i := range arrMiddle {
		use = arrMiddle[len(arrMiddle)-1-i](use).ServeHTTP
	}
	return use
}

func Get() middle       { return method(http.MethodGet) }
func Post() middle      { return method(http.MethodPost) }
func AppJSON() middle   { return contentType(appJSON) }
func TextPlain() middle { return contentType(textPlain) }

func method(method string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Method != method {
				rw.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			next.ServeHTTP(rw, req)
		})
	}
}

func contentType(contentType string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.Header.Get("Content-Type") != contentType {
				rw.WriteHeader(http.StatusUnsupportedMediaType)
				return
			}
			next.ServeHTTP(rw, req)
		})
	}
}

func AuthByFunc(ctxKey any, fnCheck func(*http.Request) (string, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(rw http.ResponseWriter, req *http.Request) {
				userIDVal, err := fnCheck(req)
				if err != nil {
					http.Error(rw, "нет userIDval", http.StatusUnauthorized)
					return
				}

				userID, err := user.ParseID(userIDVal)
				if err != nil {
					http.Error(rw, "parse userIDval", http.StatusUnauthorized)
					return
				}

				ctx := context.WithValue(req.Context(), ctxKey, userID)

				next.ServeHTTP(rw, req.WithContext(ctx))
			},
		)
	}
}

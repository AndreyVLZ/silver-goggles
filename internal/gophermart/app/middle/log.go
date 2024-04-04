package middle

import (
	"log/slog"
	"net/http"
)

type responseData struct {
	Status int `json:"httpStatus"`
	size   int
}

func newResponceData() *responseData {
	return &responseData{
		Status: http.StatusOK,
	}
}

type requestData struct {
	URI         string `json:"uri"`
	Method      string `json:"method"`
	ContentType string `json:"content-type"`
}

type loggingResponseWriter struct {
	http.ResponseWriter
	responseData *responseData
}

func newLoggingResponseWriter(rw http.ResponseWriter, resData *responseData) *loggingResponseWriter {
	return &loggingResponseWriter{
		ResponseWriter: rw,
		responseData:   resData,
	}
}

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size

	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.responseData.Status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func Logger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			reqData := &requestData{
				URI:         req.URL.String(),
				Method:      req.Method,
				ContentType: req.Header.Get("Content-Type"),
			}

			log.With("request", reqData)

			resData := newResponceData()
			defer func() {
				log.Info("-resp-", "respData", resData, "reqData", reqData)
			}()

			lmw := newLoggingResponseWriter(rw, resData)

			next.ServeHTTP(lmw, req)
		})
	}
}

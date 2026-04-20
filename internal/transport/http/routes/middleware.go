package routes

import (
	"log"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func loggingMiddleware(logger *log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		startedAt := time.Now()
		next.ServeHTTP(recorder, r)
		logger.Printf(
			"http request: method=%s path=%s status=%d duration=%s remote=%s query=%s",
			r.Method,
			r.URL.Path,
			recorder.statusCode,
			time.Since(startedAt).Round(time.Millisecond),
			r.RemoteAddr,
			r.URL.Query(),
		)
	})
}

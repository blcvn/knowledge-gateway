package write

import (
	"log"
	"net/http"
	"time"

	"kg-service/internal/access"
)

type Middleware struct{}

func NewMiddleware() Middleware {
	return Middleware{}
}

func (m Middleware) Trace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(lrw, r)

		identity, _ := access.IdentityFromContext(r.Context())
		log.Printf(
			"write request method=%s path=%s tenant=%s app=%s status=%d duration=%s",
			r.Method,
			r.URL.RequestURI(),
			identity.TenantID,
			identity.AppID,
			lrw.statusCode,
			time.Since(start).Truncate(time.Millisecond),
		)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *loggingResponseWriter) Write(p []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	return w.ResponseWriter.Write(p)
}

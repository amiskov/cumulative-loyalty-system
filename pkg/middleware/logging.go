package middleware

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/amiskov/cumulative-loyalty-system/pkg/logger"
)

type loggingMiddleware struct {
	Logger *logger.Logger
}

const requestIDKey logger.CtxLoggerKey = "requestID"

func NewLoggingMiddleware(l *logger.Logger) *loggingMiddleware {
	return &loggingMiddleware{
		Logger: l,
	}
}

func (l *loggingMiddleware) SetupTracing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = randBytesHex(16)
			r.Header.Set("X-Request-ID", requestID)
			r.Header.Set("trace-id", requestID)
			w.Header().Set("trace-id", requestID)
			w.Header().Set("X-Request-ID", requestID)
		}
		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

func (l *loggingMiddleware) SetupLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxlogger := l.Logger.With(
			zap.String("trace-id", requestIDFromContext(r.Context())),
		).WithOptions(
			zap.AddCaller(),
			zap.AddCallerSkip(1),
			zap.AddStacktrace(zap.ErrorLevel),
		).Sugar()

		ctx := context.WithValue(r.Context(), logger.LoggerKey, ctxlogger)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func (l *loggingMiddleware) AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)

		logger.Log(r.Context()).Infow(r.URL.Path,
			"method", r.Method,
			"remote_addr", r.RemoteAddr,
			"url", r.URL.Path,
			"work_time", time.Since(start),
		)
	})
}

func randBytesHex(n int) string {
	randBytes := make([]byte, n)
	rand.Read(randBytes)
	return fmt.Sprintf("%x", randBytes)
}

func requestIDFromContext(ctx context.Context) string {
	requestID, ok := ctx.Value(requestIDKey).(string)
	if !ok {
		return "-"
	}
	return requestID
}

package logger

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// RequestID echoのリクエストIDをセット
func RequestID(h echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		requestID := c.Request().Header.Get(echo.HeaderXRequestID)
		if requestID == "" {
			requestID = uuid.New().String()
			c.Request().Header.Set(echo.HeaderXRequestID, requestID)
		}
		c.Set("RequestID", requestID)

		err := h(c)

		c.Response().Header().Set(echo.HeaderXRequestID, requestID)

		return err
	}
}

// RequestLogger ログの設定
func RequestLogger(logger *zap.Logger, prefix URLPrefix) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {

			req := c.Request()
			res := c.Response()
			start := time.Now()

			err := next(c)

			var status int
			var msg string
			if err != nil {
				httpError, ok := err.(*echo.HTTPError)
				if ok {
					status = httpError.Code
					msg = httpError.Message.(string)
				}
			} else {
				status = res.Status
			}

			// latency
			latency := time.Since(start)
			// request byte size
			bytesIn := req.Header.Get(echo.HeaderContentLength)
			if bytesIn == "" {
				bytesIn = "0"
			}
			// path
			path := req.URL.Path
			if path == "" {
				path = "/"
			}
			// request_id
			requestID := req.Header.Get(echo.HeaderXRequestID)

			fields := []zap.Field{
				zap.String("request_id", requestID),
				zap.String("host", req.Host),
				zap.String("uri", req.RequestURI),
				zap.String("method", req.Method),
				zap.String("path", path),
				zap.String("remote_id", req.RemoteAddr),
				zap.String("reflect", req.Referer()),
				zap.String("ua", req.UserAgent()),
				zap.String("bytes_in", bytesIn),
				zap.Int("status", status),
				zap.Duration("latency", latency),
				zap.String("bytes_out", strconv.FormatInt(res.Size, 10)),
			}

			var f func(msg string, fields ...zap.Field)

			logLevel := zapcore.DebugLevel
			switch {
			case status >= http.StatusInternalServerError:
				if msg == "" {
					msg = "ServerError"
				}
				logLevel = zapcore.ErrorLevel
			case status >= http.StatusBadRequest:
				if msg == "" {
					msg = "ClientError"
				}
				logLevel = zapcore.WarnLevel
			case status >= http.StatusMultipleChoices:
				msg = "Redirection"
				f = logger.Info
			default:
				msg = fmt.Sprintf("Success %s", path)
				logLevel = zapcore.InfoLevel
			}

			if l := prefix.UrlLogLevel(c); l != nil {
				logLevel = *l
			}

			switch logLevel {
			case zapcore.ErrorLevel:
				f = logger.Error
			case zapcore.WarnLevel:
				f = logger.Warn
			case zapcore.InfoLevel:
				f = logger.Info
			case zapcore.DebugLevel:
				f = logger.Debug
			}

			f(msg, fields...)

			return err
		}
	}
}

// LatencyForPrometheus Prometheusの設定
func LatencyForPrometheus(summary prometheus.Summary) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			end := time.Now()
			summary.Observe(end.Sub(start).Seconds())
			return err
		}
	}
}

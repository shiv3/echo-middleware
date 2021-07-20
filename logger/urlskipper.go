package logger

import (
	"github.com/labstack/echo/v4"
	"go.uber.org/zap/zapcore"
)

type URLPrefix interface {
	UrlSkipper(c echo.Context) bool
	UrlLogLevel(c echo.Context) *zapcore.Level
}

type URLPrefixImpl struct {
	SkipPathPrefix map[string]bool
	PathLogLevel   map[string]zapcore.Level
}

func NewURLPrefixImpl(skipPathPrefix map[string]bool, pathLogLevel map[string]zapcore.Level) URLPrefixImpl {
	return URLPrefixImpl{SkipPathPrefix: skipPathPrefix, PathLogLevel: pathLogLevel}
}

func (s URLPrefixImpl) UrlLogLevel(c echo.Context) *zapcore.Level {
	if l, ok := s.PathLogLevel[c.Path()]; ok {
		return &l
	}
	return nil
}

// urlSkipper ignores metrics route on some middleware
func (s URLPrefixImpl) UrlSkipper(c echo.Context) bool {
	return s.SkipPathPrefix[c.Path()]
}

package logger

import (
	"fmt"
	"sync"
	"time"

	"github.com/nezhahq/service"
)

var (
	defaultLogger = &ServiceLogger{enabled: true, logger: service.ConsoleLogger}

	loggerOnce sync.Once
)

type ServiceLogger struct {
	enabled bool
	logger  service.Logger
}

func InitDefaultLogger(enabled bool, logger service.Logger) {
	loggerOnce.Do(func() {
		defaultLogger.enabled = enabled
		defaultLogger.logger = logger
	})
}

func SetEnable(enable bool) {
	defaultLogger.SetEnable(enable)
}

func Println(v ...interface{}) {
	defaultLogger.Println(v...)
}

func Printf(format string, v ...interface{}) {
	defaultLogger.Printf(format, v...)
}

func Error(v ...interface{}) error {
	return defaultLogger.Error(v...)
}

func Errorf(format string, v ...interface{}) error {
	return defaultLogger.Errorf(format, v...)
}

func NewServiceLogger(enable bool, logger service.Logger) *ServiceLogger {
	return &ServiceLogger{
		enabled: enable,
		logger:  logger,
	}
}

func (s *ServiceLogger) SetEnable(enable bool) {
	s.enabled = enable
}

func (s *ServiceLogger) Println(v ...interface{}) {
	if s.enabled {
		s.logger.Infof("NEZHA@%s>> %v", time.Now().Format(time.DateTime), fmt.Sprint(v...))
	}
}

func (s *ServiceLogger) Printf(format string, v ...interface{}) {
	if s.enabled {
		s.logger.Infof("NEZHA@%s>> "+format, append([]interface{}{time.Now().Format(time.DateTime)}, v...)...)
	}
}

func (s *ServiceLogger) Error(v ...interface{}) error {
	if s.enabled {
		return s.logger.Errorf("NEZHA@%s>> %v", time.Now().Format(time.DateTime), fmt.Sprint(v...))
	}
	return nil
}

func (s *ServiceLogger) Errorf(format string, v ...interface{}) error {
	if s.enabled {
		return s.logger.Errorf("NEZHA@%s>> "+format, append([]interface{}{time.Now().Format(time.DateTime)}, v...)...)
	}
	return nil
}

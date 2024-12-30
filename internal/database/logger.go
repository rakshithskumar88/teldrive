package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

const msgPrefix = "[DB] "

type Logger struct {
	cfg glogger.Config
	lg  *zap.SugaredLogger
}

func NewLogger(lg *zap.SugaredLogger, slowThreshold time.Duration, ignoreRecordNotFoundError bool, level zapcore.Level) *Logger {
	cfg := glogger.Config{
		SlowThreshold:             slowThreshold,
		Colorful:                  false,
		IgnoreRecordNotFoundError: ignoreRecordNotFoundError,
	}
	switch level {
	case zapcore.DebugLevel, zapcore.InfoLevel:
		cfg.LogLevel = glogger.Info
	case zapcore.WarnLevel:
		cfg.LogLevel = glogger.Warn
	case zapcore.ErrorLevel:
		cfg.LogLevel = glogger.Error
	default:
		cfg.LogLevel = glogger.Silent
	}
	return &Logger{cfg: cfg, lg: lg.WithOptions(zap.AddCallerSkip(3))}
}

func (l *Logger) LogMode(level glogger.LogLevel) glogger.Interface {
	newlogger := *l
	newlogger.cfg.LogLevel = level
	return &newlogger
}

func (l *Logger) Info(ctx context.Context, s string, i ...interface{}) {
	if l.cfg.LogLevel >= glogger.Info {
		l.lg.Infof(msgPrefix+s, i)
	}
}

func (l *Logger) Warn(ctx context.Context, s string, i ...interface{}) {
	if l.cfg.LogLevel >= glogger.Warn {
		l.lg.Warnf(msgPrefix+s, i)
	}
}

func (l *Logger) Error(ctx context.Context, s string, i ...interface{}) {
	if l.cfg.LogLevel >= glogger.Error {
		l.lg.Errorf(msgPrefix+s, i)
	}
}

func (l *Logger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.cfg.LogLevel <= glogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	logger := l.lg

	const (
		traceStr     = msgPrefix + "%s\n[%.3fms] [rows:%v] %s"
		traceWarnStr = msgPrefix + "%s %s\n[%.3fms] [rows:%v] %s"
		traceErrStr  = msgPrefix + "%s %s\n[%.3fms] [rows:%v] %s"
	)

	switch {
	case err != nil && l.cfg.LogLevel >= glogger.Error && (!errors.Is(err, gorm.ErrRecordNotFound) || !l.cfg.IgnoreRecordNotFoundError):
		sql, rows := fc()
		if rows == -1 {
			logger.Errorf(traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			logger.Errorf(traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case elapsed > l.cfg.SlowThreshold && l.cfg.SlowThreshold != 0 && l.cfg.LogLevel >= glogger.Warn:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.cfg.SlowThreshold)
		if rows == -1 {
			logger.Warnf(traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			logger.Warnf(traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case l.cfg.LogLevel == glogger.Info:
		sql, rows := fc()
		if rows == -1 {
			logger.Infof(traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			logger.Infof(traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
}

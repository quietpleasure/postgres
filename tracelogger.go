package postgres

import (
	logrus_adapter "github.com/jackc/pgx-logrus"
	zap_adapter "github.com/jackc/pgx-zap"
	zero_adapter "github.com/jackc/pgx-zerolog"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

func WithZapLogger(log *zap.Logger) Option {
	return func(options *options) error {
		if log != nil {
			lvl, err := tracelog.LogLevelFromString(log.Level().String())
			if err != nil {
				lvl = tracelog.LogLevelTrace
			}
			options.tracelogger = &tracelog.TraceLog{
				Logger:   zap_adapter.NewLogger(log),
				LogLevel: lvl,
			}
		}
		return nil
	}
}

func WithZeroLogger(log *zerolog.Logger) Option {
	return func(options *options) error {
		if log != nil {
			lvl, err := tracelog.LogLevelFromString(log.GetLevel().String())
			if err != nil {
				lvl = tracelog.LogLevelTrace
			}
			options.tracelogger = &tracelog.TraceLog{
				Logger:   zero_adapter.NewLogger(*log),
				LogLevel: lvl,
			}
		}
		return nil
	}
}

func WithLogrusLogger(log logrus.FieldLogger, level string) Option {
	return func(options *options) error {
		if log != nil {
			lvl, err := tracelog.LogLevelFromString(level)
			if err != nil {
				lvl = tracelog.LogLevelTrace
			}
			options.tracelogger = &tracelog.TraceLog{
				Logger:   logrus_adapter.NewLogger(log),
				LogLevel: lvl,
			}
		}
		return nil
	}
}

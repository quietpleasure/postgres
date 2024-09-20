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

// TODO: with hook level
func WithZapLogger(log *zap.Logger, level string) Option {
	return func(options *options) error {
		if log != nil {
			lvl, err := tracelog.LogLevelFromString(level)
			if err != nil {
				return err
			}
			options.tracelogger = &tracelog.TraceLog{
				Logger:   zap_adapter.NewLogger(log),
				LogLevel: lvl,
			}
		}
		return nil
	}
}

// TODO: with hook level
func WithZeroLogger(log *zerolog.Logger, level string) Option {
	return func(options *options) error {
		if log != nil {
			lvl, err := tracelog.LogLevelFromString(level)
			if err != nil {
				return err
			}
			options.tracelogger = &tracelog.TraceLog{
				Logger:   zero_adapter.NewLogger(*log),
				LogLevel: lvl,
			}
		}
		return nil
	}
}

// TODO: with hook level
func WithLogrusLogger(log logrus.FieldLogger, level string) Option {
	return func(options *options) error {
		if log != nil {
			lvl, err := tracelog.LogLevelFromString(level)
			if err != nil {
				return err
			}
			options.tracelogger = &tracelog.TraceLog{
				Logger:   logrus_adapter.NewLogger(log),
				LogLevel: lvl,
			}
		}
		return nil
	}
}

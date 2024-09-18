package postgres

import (
	logrus_adapter "github.com/jackc/pgx-logrus"
	zap_adapter "github.com/jackc/pgx-zap"
	zero_adapter "github.com/jackc/pgx-zerolog"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

func WithZapLogger(log *zap.Logger) Option {
	return func(options *options) error {
		if log != nil {
			options.logger = zap_adapter.NewLogger(log)
		}
		return nil
	}
}

func WithZeroLogger(log *zerolog.Logger) Option {
	return func(options *options) error {
		if log != nil {
			options.logger = zero_adapter.NewLogger(*log)
		}
		return nil
	}
}

func WithLogrusLogger(log logrus.FieldLogger) Option {
	return func(options *options) error {
		if log != nil {
			options.logger = logrus_adapter.NewLogger(log)
		}
		return nil
	}
}

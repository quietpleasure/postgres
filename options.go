package postgres

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/tracelog"
)

type Config struct {
	Database  string `yaml:"database"`
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	User      string `yaml:"user"`
	Pass      string `yaml:"pass"`
	Schema    string `yaml:"schema"`
	Withlog   bool   `yaml:"withlog"`
	AddParams struct {
		SSLMode               string        `yaml:"ssl_mode"`
		MaxConns              int           `yaml:"max_conns"`
		MaxConnLifeTime       time.Duration `yaml:"max_conn_life_time"`
		MaxConnIdleTime       time.Duration `yaml:"max_conn_idle_time"`
		MaxConnLifeTimeJitter time.Duration `yaml:"max_conn_life_time_jitter"`
		HealthCheckPeriod     time.Duration `yaml:"health_check_period"`
	} `yaml:"add_params"`
}

func (c Config) Options() []Option {
	options := make([]Option, 0, 2)
	return append(
		options,
		WithHost(c.Host),
		WithPort(c.Port),
		WithDatabase(c.Database),
		WithUser(c.User),
		WithPass(c.Pass),
		WithSchema(c.Schema),
		WithSSLMode(c.AddParams.SSLMode),
		WithMaxConns(c.AddParams.MaxConns),
		WithMaxConnLifeTime(c.AddParams.MaxConnLifeTime),
		WithMaxConnLifeTimeJitter(c.AddParams.MaxConnLifeTimeJitter),
		WithMaxConnIdleTime(c.AddParams.MaxConnIdleTime),
		WithHealthCheckPeriod(c.AddParams.HealthCheckPeriod),
	)
}

type Option func(option *options) error

type options struct {
	host                  *net.IP
	port                  *int
	database              *string
	schema                *string
	user                  *string
	pass                  *string
	sslmode               *string
	maxconns              *int
	minconns              *int
	maxconnlifetime       *time.Duration
	maxconnidletime       *time.Duration
	healthcheckperiod     *time.Duration
	maxconnlifetimejitter *time.Duration
	tracelogger           *tracelog.TraceLog
	hook                  *hook
	closectx              context.Context
}

// WithCloseOnDone closes the pool automatically when ctx is done (e.g. the application's root context).
func WithCloseOnDone(ctx context.Context) Option {
	return func(options *options) error {
		if ctx == nil {
			return fmt.Errorf("close context cannot be nil")
		}
		options.closectx = ctx
		return nil
	}
}

// default host=127.0.0.1
func WithHost(host string) Option {
	return func(options *options) error {
		if host == "" || host == "localhost" {
			host = default_host
		}
		ip := new(net.IP)
		if err := ip.UnmarshalText([]byte(host)); err != nil {
			return err
		}
		options.host = ip
		return nil
	}
}

// default port=5432
func WithPort(port int) Option {
	return func(options *options) error {
		switch {
		case port == 0:
			port = default_port
		case port < 0:
			return fmt.Errorf("port cannot be less than zero")
		}
		options.port = &port
		return nil
	}
}

// default database=postgres
func WithDatabase(database string) Option {
	return func(options *options) error {
		if database == "" {
			database = default_database
		}
		options.database = &database
		return nil
	}
}

// default schema=public
func WithSchema(schema string) Option {
	return func(options *options) error {
		if schema == "" {
			schema = default_schema
		}
		options.schema = &schema
		return nil
	}
}

// default user=postgres
func WithUser(user string) Option {
	return func(options *options) error {
		if user == "" {
			user = default_user
		}
		options.user = &user
		return nil
	}
}

func WithPass(pass string) Option {
	return func(options *options) error {
		options.pass = &pass
		return nil
	}
}

// default ssl_mode=disable
func WithSSLMode(mode string) Option {
	return func(options *options) error {
		if mode == "" {
			mode = disable_ssl_mode
		}
		options.sslmode = &mode
		return nil
	}
}

// MaxConns is the maximum size of the pool. The default is the greater of 4 or runtime.NumCPU().
func WithMaxConns(conns int) Option {
	return func(options *options) error {
		if conns < 0 {
			return fmt.Errorf("max connections cannot be less than zero")
		}
		if conns == 0 {
			options.maxconns = nil
		} else {
			options.maxconns = &conns
		}
		return nil
	}
}

// MinConns is the minimum size of the pool. After connection closes, the pool might dip below MinConns. A low number of MinConns might mean the pool is empty after MaxConnLifetime until the health check has a chance to create new connections.
func WithMinConns(conns int) Option {
	return func(options *options) error {
		if conns < 0 {
			return fmt.Errorf("min connections cannot be less than zero")
		}
		if conns == 0 {
			options.minconns = nil
		} else {
			options.minconns = &conns
		}
		return nil
	}
}

// MaxConnLifetime is the duration since creation after which a connection will be automatically closed.
func WithMaxConnLifeTime(lifetime time.Duration) Option {
	return func(options *options) error {
		if lifetime < 0 {
			return fmt.Errorf("max connection life time cannot be less than zero")
		}
		if lifetime == 0 {
			options.maxconnlifetime = nil
		} else {
			options.maxconnlifetime = &lifetime
		}
		return nil
	}
}

// MaxConnIdleTime is the duration after which an idle connection will be automatically closed by the health check.
func WithMaxConnIdleTime(idletime time.Duration) Option {
	return func(options *options) error {
		if idletime < 0 {
			return fmt.Errorf("max connection idle time cannot be less than zero")
		}
		if idletime == 0 {
			options.maxconnidletime = nil
		} else {
			options.maxconnidletime = &idletime
		}
		return nil
	}
}

// HealthCheckPeriod is the duration between checks of the health of idle connections.
func WithHealthCheckPeriod(period time.Duration) Option {
	return func(options *options) error {
		if period < 0 {
			return fmt.Errorf("health check period cannot be less than zero")
		}
		if period == 0 {
			options.healthcheckperiod = nil
		} else {
			options.healthcheckperiod = &period
		}
		return nil
	}
}

// MaxConnLifetimeJitter is the duration after MaxConnLifetime to randomly decide to close a connection. This helps prevent all connections from being closed at the exact same time, starving the pool.
func WithMaxConnLifeTimeJitter(jitter time.Duration) Option {
	return func(options *options) error {
		if jitter < 0 {
			return fmt.Errorf("max connection life time jitter cannot be less than zero")
		}
		if jitter == 0 {
			options.maxconnlifetimejitter = nil
		} else {
			options.maxconnlifetimejitter = &jitter
		}
		return nil
	}
}

// WithHook sets the schema and table for the hook, default schema=public, table={application name}_logs
// or logs if there is an error when specifying the name of the executable file
func WithHook(schema, table string) Option {
	return func(options *options) error {
		rep := strings.NewReplacer("-", "_")
		if schema == "" {
			schema = default_schema
		} else {
			schema = rep.Replace(schema)
		}
		if table == "" {
			binpath, err := os.Executable()
			if err != nil {
				table = "logs"
			} else {
				table = rep.Replace(strings.Split(filepath.Base(binpath), ".")[0]) + "_logs"
			}
		}
		// names are quoted in SQL, so fold to lower case as postgres does for unquoted identifiers
		options.hook = &hook{
			schema: strings.ToLower(schema),
			table:  strings.ToLower(table),
		}
		return nil
	}
}

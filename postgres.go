package postgres

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	postgres         = "postgres"
	default_port     = 5432
	default_host     = "127.0.0.1"
	default_user     = "postgres"
	default_database = "postgres"
	disable_ssl_mode = "disable"
)

// Function for passing connection parameters
type Option func(option *options) error

type options struct {
	host                  *net.IP
	port                  *int
	database              *string
	user                  *string
	pass                  *string
	sslmode               *string
	maxconns              *int
	minconns              *int
	maxconnlifetime       *time.Duration
	maxconnidletime       *time.Duration
	healthcheckperiod     *time.Duration
	maxconnlifetimejitter *time.Duration
}

var ErrNoRows error = pgx.ErrNoRows

// Структура со встроенным пулом соединений
type Pool struct {
	*pgxpool.Pool
}

// Creates a new connection pool with parameters. If no parameters are passed, the default settings will be applied. Immediately after connection, a ping is carried out for verification.
func New(ctx context.Context, opts ...Option) (*Pool, error) {
	var opt options
	for _, option := range opts {
		if err := option(&opt); err != nil {
			return nil, err
		}
	}

	if opt.host == nil {
		ip := new(net.IP)
		if err := ip.UnmarshalText([]byte(default_host)); err != nil {
			return nil, err
		}
		opt.host = ip
	}

	var port int
	if opt.port == nil {
		port = default_port
	} else {
		port = *opt.port
	}
	var database string
	if opt.database == nil {
		database = default_database
	} else {
		database = *opt.database
	}
	var user string
	if opt.user == nil {
		user = default_user
	} else {
		user = *opt.user
	}
	var pass string
	if opt.pass == nil {
		pass = ""
	} else {
		pass = *opt.pass
	}

	val := url.Values{}
	if opt.sslmode != nil {
		val.Set("sslmode", *opt.sslmode)
	}

	url := &url.URL{
		Scheme:   postgres,
		Host:     fmt.Sprintf("%s:%d", *opt.host, port),
		Path:     database,
		User:     url.UserPassword(user, pass),
		RawQuery: val.Encode(),
	}

	conCfg, err := pgxpool.ParseConfig(url.String())
	if err != nil {
		return nil, err
	}
	if opt.maxconns != nil && *opt.maxconns != 0 {
		conCfg.MaxConns = int32(*opt.maxconns)
	}
	if opt.minconns != nil && *opt.minconns != 0 {
		conCfg.MinConns = int32(*opt.minconns)
	}
	if opt.maxconnlifetime != nil && *opt.maxconnlifetime != 0 {
		conCfg.MaxConnLifetime = *opt.maxconnlifetime
	}
	if opt.maxconnidletime != nil {
		conCfg.MaxConnIdleTime = *opt.maxconnidletime
	}
	if opt.healthcheckperiod != nil {
		conCfg.HealthCheckPeriod = *opt.healthcheckperiod
	}
	if opt.maxconnlifetimejitter != nil {
		conCfg.MaxConnLifetimeJitter = *opt.maxconnlifetimejitter
	}
	conCfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		if err := conn.Ping(ctx); err != nil {
			return fmt.Errorf("ping after connect: %s", err)
		}
		return nil
	}
	pool, err := pgxpool.NewWithConfig(ctx, conCfg)
	if err != nil {
		return nil, err
	}
	return &Pool{pool}, nil
}

// default host=127.0.0.1
func WithHost(host string) Option {
	return func(options *options) error {
		if host == "" {
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
		options.maxconns = &conns
		return nil
	}
}

// MinConns is the minimum size of the pool. After connection closes, the pool might dip below MinConns. A low number of MinConns might mean the pool is empty after MaxConnLifetime until the health check has a chance to create new connections.
func WithMinConns(conns int) Option {
	return func(options *options) error {
		if conns < 0 {
			return fmt.Errorf("min connections cannot be less than zero")
		}
		options.minconns = &conns
		return nil
	}
}

// MaxConnLifetime is the duration since creation after which a connection will be automatically closed.
func WithMaxConnLifeTime(lifetime time.Duration) Option {
	return func(options *options) error {
		if lifetime < 0 {
			return fmt.Errorf("max connection life time cannot be less than zero")
		}
		options.maxconnlifetime = &lifetime
		return nil
	}
}

// MaxConnIdleTime is the duration after which an idle connection will be automatically closed by the health check.
func WithMaxConnIdleTime(idletime time.Duration) Option {
	return func(options *options) error {
		if idletime < 0 {
			return fmt.Errorf("max connection idle time cannot be less than zero")
		}
		options.maxconnidletime = &idletime
		return nil
	}
}

// HealthCheckPeriod is the duration between checks of the health of idle connections.
func WithHealthCheckPeriod(period time.Duration) Option {
	return func(options *options) error {
		if period <= 0 {
			return fmt.Errorf("health check period cannot be less than or equal to zero")
		}
		options.healthcheckperiod = &period
		return nil
	}
}

// MaxConnLifetimeJitter is the duration after MaxConnLifetime to randomly decide to close a connection. This helps prevent all connections from being closed at the exact same time, starving the pool.
func WithMaxConnLifeTimeJitter(jitter time.Duration) Option {
	return func(options *options) error {
		if jitter < 0 {
			return fmt.Errorf("max connection life time jitter cannot be less than zero")
		}
		options.maxconnlifetimejitter = &jitter
		return nil
	}
}

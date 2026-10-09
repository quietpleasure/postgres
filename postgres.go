package postgres

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	self_name        = "postgres"
	default_port     = 5432
	default_host     = "127.0.0.1"
	default_user     = "postgres"
	default_pass     = "postgres"
	default_database = "postgres"
	disable_ssl_mode = "disable"
	default_schema   = "public"
)

// var ErrNoRows error = pgx.ErrNoRows

type Postgres struct {
	*pgxpool.Pool

	hook       *hook
	MainSchema string
}

// Creates a new connection pool with parameters. If no parameters are passed, the default settings will be applied. Immediately after connection, a ping is carried out for verification.
// ctx limits only the initialization (connect, ping, hook setup); cancelling it later does not close the pool.
// Close the pool with Close/CloseConnect, or bind it to a context with WithCloseOnDone.
func New(ctx context.Context, opts ...Option) (*Postgres, error) {
	var opt options
	for _, option := range opts {
		if err := option(&opt); err != nil {
			return nil, err
		}
	}

	ip := new(net.IP)
	if opt.host == nil {
		if err := ip.UnmarshalText([]byte(default_host)); err != nil {
			return nil, err
		}
	} else {
		ip = opt.host
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
		pass = default_pass
	} else {
		pass = *opt.pass
	}

	val := url.Values{}
	if opt.sslmode != nil {
		val.Set("sslmode", *opt.sslmode)
	}

	url := &url.URL{
		Scheme:   self_name,
		Host:     fmt.Sprintf("%s:%d", *ip, port),
		Path:     database,
		User:     url.UserPassword(user, pass),
		RawQuery: val.Encode(),
	}

	conCfg, err := pgxpool.ParseConfig(url.String())
	if err != nil {
		return nil, err
	}
	if opt.tracelogger != nil {
		conCfg.ConnConfig.Tracer = opt.tracelogger
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

	pool, err := pgxpool.NewWithConfig(ctx, conCfg)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	schema := default_schema
	if opt.schema != nil {
		schema = *opt.schema
	}
	pg := &Postgres{Pool: pool, hook: opt.hook, MainSchema: schema}

	if pg.hook != nil {
		if err := pg.createHookDestination(ctx); err != nil {
			pool.Close()
			return nil, err
		}
	}

	if opt.closectx != nil {
		context.AfterFunc(opt.closectx, pool.Close)
	}
	return pg, nil
}

func (p *Postgres) CloseConnect() {
	p.Close()
}

func (p *Postgres) GetMainSchema() string {
	return p.MainSchema
}

func (p *Postgres) GetHookSchema() string {
	if p.hook != nil {
		return p.hook.schema
	}
	return ""
}

func (p *Postgres) GetHookTable() string {
	if p.hook != nil {
		return p.hook.table
	}
	return ""
}

func IntervalToDuration(interval pgtype.Interval) time.Duration {
	return time.Duration(interval.Microseconds * int64(time.Microsecond))
}

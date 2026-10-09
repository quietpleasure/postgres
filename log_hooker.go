package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type hook struct {
	schema string
	table  string
}

// ident returns the quoted "schema"."table" name, safe to embed in SQL.
func (h *hook) ident() string {
	return pgx.Identifier{h.schema, h.table}.Sanitize()
}

// createHookDestination creates the log schema/table if they do not exist and
// checks the structure of an existing table. All statements run in one
// transaction, serialized by an advisory lock, so several processes starting
// at the same time do not race on CREATE SCHEMA / CREATE TABLE.
func (p *Postgres) createHookDestination(ctx context.Context) error {
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, p.hook.ident()); err != nil {
		return fmt.Errorf("lock hook destination: %w", err)
	}

	// check schema exists (pg_namespace, unlike information_schema.schemata, lists schemas regardless of privileges)
	var exists bool
	err = tx.QueryRow(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = $1)`, p.hook.schema,
	).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		// создаю схему
		query := "CREATE SCHEMA " + pgx.Identifier{p.hook.schema}.Sanitize()
		if _, err := tx.Exec(ctx, query); err != nil {
			return err
		}
	}

	// check table exists
	rows, err := tx.Query(ctx,
		`SELECT column_name, data_type FROM information_schema.columns WHERE table_schema = $1 and table_name = $2`,
		p.hook.schema,
		p.hook.table,
	)
	if err != nil {
		return err
	}
	cmap := make(map[string]string, 4)
	// CollectRows closes rows before the next statement on the same connection.
	_, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (struct{}, error) {
		var cname, dtype string
		if err := row.Scan(&cname, &dtype); err != nil {
			return struct{}{}, err
		}
		cmap[cname] = dtype
		return struct{}{}, nil
	})
	if err != nil {
		return err
	}

	switch len(cmap) {
	case 0:
		// создаю таблицу
		if err := p.createHookTable(ctx, tx); err != nil {
			return err
		}
	case 4:
		// проверяю  имена и типы колонок
		type1, ok1 := cmap["id"]
		type2, ok2 := cmap["ts"]
		type3, ok3 := cmap["lvl"]
		type4, ok4 := cmap["raw_msg"]
		if !ok1 || !ok2 || !ok3 || !ok4 {
			return fmt.Errorf("unexpected columns: %v", cmap)
		}
		if type1 != "uuid" || type2 != "timestamp with time zone" || type3 != "character varying" || type4 != "jsonb" {
			return fmt.Errorf("unexpected columns types: %v", cmap)
		}
	default:
		return fmt.Errorf("unexpected columns count: %d", len(cmap))
	}
	return tx.Commit(ctx)
}

func (p *Postgres) createHookTable(ctx context.Context, tx pgx.Tx) error {
	query := fmt.Sprintf(`CREATE TABLE %s (
		id uuid NOT NULL DEFAULT gen_random_uuid(),
		ts timestamptz NOT NULL DEFAULT now(),
		lvl varchar NOT NULL,
		raw_msg jsonb NOT NULL,
		CONSTRAINT %s PRIMARY KEY (id)
	);`, p.hook.ident(), pgx.Identifier{"pk_" + p.hook.table}.Sanitize())
	if _, err := tx.Exec(ctx, query); err != nil {
		return err
	}
	query = fmt.Sprintf("CREATE INDEX %s ON %s (lvl)", pgx.Identifier{p.hook.table + "_lvl"}.Sanitize(), p.hook.ident())
	if _, err := tx.Exec(ctx, query); err != nil {
		return err
	}
	return nil
}

// implements Hooker interface for logger
func (p *Postgres) SendLog(ctx context.Context, ts time.Time, lvl string, msg []byte) error {
	if p.hook == nil {
		return errors.New("log hook is not configured, use WithHook")
	}
	query := fmt.Sprintf(`INSERT INTO %s (ts,lvl,raw_msg) VALUES ($1,upper($2),$3)`, p.hook.ident())
	_, err := p.Exec(
		ctx,
		query,
		ts,
		lvl,
		msg,
	)
	return err
}

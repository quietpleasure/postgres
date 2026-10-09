package postgres

import (
	"context"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
)

// testOptions builds connection options from PG_TEST_* env variables.
// Tests that need a real database are skipped when PG_TEST_DATABASE is not set.
func testOptions(t *testing.T) []Option {
	t.Helper()
	database := os.Getenv("PG_TEST_DATABASE")
	if database == "" {
		t.Skip("PG_TEST_DATABASE is not set, skipping test with real postgres")
	}
	port, _ := strconv.Atoi(os.Getenv("PG_TEST_PORT"))
	return []Option{
		WithHost(os.Getenv("PG_TEST_HOST")),
		WithPort(port),
		WithDatabase(database),
		WithUser(os.Getenv("PG_TEST_USER")),
		WithPass(os.Getenv("PG_TEST_PASS")),
	}
}

func TestCreateHookDestinationConcurrent(t *testing.T) {
	opts := testOptions(t)
	ctx := context.Background()
	const (
		schema = "hook_test"
		table  = "t_logs"
	)

	admin, err := New(ctx, opts...)
	if err != nil {
		t.Fatal(err)
	}
	// Cleanup выполняется в обратном порядке: сначала drop, потом закрытие пула
	t.Cleanup(admin.Close)
	drop := func() {
		if _, err := admin.Exec(ctx, "DROP SCHEMA IF EXISTS "+pgx.Identifier{schema}.Sanitize()+" CASCADE"); err != nil {
			t.Fatal(err)
		}
	}
	drop()
	t.Cleanup(drop)

	initAll := func(n int) {
		var wg sync.WaitGroup
		errs := make(chan error, n)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				hookOpts := append(append([]Option{}, opts...), WithHook(schema, table))
				pg, err := New(ctx, hookOpts...)
				if err != nil {
					errs <- err
					return
				}
				pg.Close()
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			t.Error(err)
		}
	}

	// одновременное создание схемы и таблицы несколькими процессами
	initAll(8)
	// повторная инициализация по уже существующей таблице
	initAll(3)

	var cols int
	err = admin.QueryRow(ctx,
		`SELECT count(*) FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2`,
		schema, table,
	).Scan(&cols)
	if err != nil {
		t.Fatal(err)
	}
	if cols != 4 {
		t.Fatalf("expected 4 columns, got %d", cols)
	}
}

package db

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func setupTxTestDB(t *testing.T) *DBManager {
	t.Helper()

	manager := NewDBManager()
	if err := manager.Open("app", "sqlite", filepath.Join(t.TempDir(), "app.db")); err != nil {
		t.Fatalf("open app db: %v", err)
	}
	t.Cleanup(func() {
		_ = manager.Close()
	})

	d, err := manager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}
	if _, err := d.Exec(`CREATE TABLE records (id TEXT PRIMARY KEY, name TEXT NOT NULL)`); err != nil {
		t.Fatalf("create records table: %v", err)
	}

	return manager
}

func TestWithTxCommits(t *testing.T) {
	manager := setupTxTestDB(t)

	err := manager.WithTx(context.Background(), "app", func(ctx context.Context) error {
		exec, err := manager.Executor(ctx, "app")
		if err != nil {
			return err
		}
		_, err = exec.Exec(exec.Rebind(`INSERT INTO records (id, name) VALUES (?, ?)`), "1", "committed")
		return err
	})
	if err != nil {
		t.Fatalf("with tx: %v", err)
	}

	exec, err := manager.Executor(context.Background(), "app")
	if err != nil {
		t.Fatalf("executor: %v", err)
	}
	var count int
	if err := exec.Get(&count, `SELECT COUNT(*) FROM records`); err != nil {
		t.Fatalf("count records: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 committed record, got %d", count)
	}
}

func TestWithTxRollsBackOnError(t *testing.T) {
	manager := setupTxTestDB(t)
	expectedErr := errors.New("stop")

	err := manager.WithTx(context.Background(), "app", func(ctx context.Context) error {
		exec, err := manager.Executor(ctx, "app")
		if err != nil {
			return err
		}
		if _, err := exec.Exec(exec.Rebind(`INSERT INTO records (id, name) VALUES (?, ?)`), "1", "rolled back"); err != nil {
			return err
		}
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected rollback cause %v, got %v", expectedErr, err)
	}

	exec, err := manager.Executor(context.Background(), "app")
	if err != nil {
		t.Fatalf("executor: %v", err)
	}
	var count int
	if err := exec.Get(&count, `SELECT COUNT(*) FROM records`); err != nil {
		t.Fatalf("count records: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected rollback to remove records, got %d", count)
	}
}

func TestWithTxReusesNestedTransaction(t *testing.T) {
	manager := setupTxTestDB(t)

	err := manager.WithTx(context.Background(), "app", func(ctx context.Context) error {
		outer, err := manager.Executor(ctx, "app")
		if err != nil {
			return err
		}

		if err := manager.WithTx(ctx, "app", func(nestedCtx context.Context) error {
			nested, err := manager.Executor(nestedCtx, "app")
			if err != nil {
				return err
			}
			if outer != nested {
				t.Fatalf("expected nested transaction executor to be reused")
			}
			_, err = nested.Exec(nested.Rebind(`INSERT INTO records (id, name) VALUES (?, ?)`), "1", "nested")
			return err
		}); err != nil {
			return err
		}

		var count int
		if err := outer.Get(&count, `SELECT COUNT(*) FROM records`); err != nil {
			return err
		}
		if count != 1 {
			t.Fatalf("expected outer transaction to see nested write, got %d", count)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("with nested tx: %v", err)
	}
}

func TestDynamicExecutorUsesActiveTransaction(t *testing.T) {
	manager := setupTxTestDB(t)

	err := manager.WithTx(context.Background(), "app", func(ctx context.Context) error {
		exec, err := manager.DynamicExecutor(ctx, "app")
		if err != nil {
			return err
		}
		_, err = exec.Exec(
			`INSERT INTO records (id, name) VALUES (:id, :name)`,
			map[string]interface{}{"id": "1", "name": "dynamic"},
		)
		return err
	})
	if err != nil {
		t.Fatalf("with tx: %v", err)
	}

	exec, err := manager.DynamicExecutor(context.Background(), "app")
	if err != nil {
		t.Fatalf("dynamic executor: %v", err)
	}
	var name string
	if err := exec.Get(&name, `SELECT name FROM records WHERE id = :id`, map[string]interface{}{"id": "1"}); err != nil {
		t.Fatalf("get record: %v", err)
	}
	if name != "dynamic" {
		t.Fatalf("expected dynamic record, got %q", name)
	}
}

func TestWithTxRejectsSharedDatabase(t *testing.T) {
	manager := setupTxTestDB(t)
	called := false

	err := manager.WithTx(context.Background(), "shared", func(ctx context.Context) error {
		called = true
		return nil
	})
	if !errors.Is(err, ErrTransactionsOnlySupportApp) {
		t.Fatalf("expected app-only transaction error, got %v", err)
	}
	if called {
		t.Fatalf("expected shared transaction callback not to run")
	}
}

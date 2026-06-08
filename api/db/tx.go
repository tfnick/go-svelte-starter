package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/tfnick/sqlx"
)

var ErrTransactionsOnlySupportApp = errors.New("transactions only support app database")

type txContextKey struct{}

type txEntry struct {
	dbName string
	tx     *sqlx.Tx
}

type txContextValue struct {
	entries []txEntry
}

type Executor interface {
	Rebind(query string) string
	Exec(query string, args ...interface{}) (sql.Result, error)
	NamedExec(query string, arg interface{}) (sql.Result, error)
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
}

type DynamicExecutor struct {
	exec Executor
}

func (e DynamicExecutor) Exec(query string, arg ...interface{}) (sql.Result, error) {
	q, args, err := dynamicQuery(e.exec, query, arg...)
	if err != nil {
		return nil, err
	}
	return e.exec.Exec(q, args...)
}

func (e DynamicExecutor) Get(dest interface{}, query string, arg ...interface{}) error {
	q, args, err := dynamicQuery(e.exec, query, arg...)
	if err != nil {
		return err
	}
	return e.exec.Get(dest, q, args...)
}

func (e DynamicExecutor) Select(dest interface{}, query string, arg ...interface{}) error {
	q, args, err := dynamicQuery(e.exec, query, arg...)
	if err != nil {
		return err
	}
	return e.exec.Select(dest, q, args...)
}

func dynamicQuery(exec Executor, query string, arg ...interface{}) (string, []interface{}, error) {
	var a interface{}
	if len(arg) > 0 {
		a = arg[0]
	}

	query = sqlx.Preprocess(query, a)
	if a == nil {
		return query, nil, nil
	}

	q, args, err := sqlx.Named(query, a)
	if err != nil {
		return "", nil, err
	}
	q, args, err = sqlx.In(q, args...)
	if err != nil {
		return "", nil, err
	}
	return exec.Rebind(q), args, nil
}

func (m *DBManager) WithTx(ctx context.Context, name string, fn func(context.Context) error) error {
	if name != "app" {
		return fmt.Errorf("%w: %s", ErrTransactionsOnlySupportApp, name)
	}
	if tx := txFromContext(ctx, name); tx != nil {
		return fn(ctx)
	}

	d, err := m.GetDB(name)
	if err != nil {
		return err
	}

	tx, err := d.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction %s failed: %w", name, err)
	}

	txCtx := contextWithTx(ctx, name, tx)
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(txCtx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("rollback transaction %s failed after %v: %w", name, err, rollbackErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction %s failed: %w", name, err)
	}
	return nil
}

func (m *DBManager) Executor(ctx context.Context, name string) (Executor, error) {
	if tx := txFromContext(ctx, name); tx != nil {
		return tx, nil
	}
	return m.GetDB(name)
}

func (m *DBManager) SQLDB(name string) (*sql.DB, error) {
	d, err := m.GetDB(name)
	if err != nil {
		return nil, err
	}
	return d.DB, nil
}

func (m *DBManager) SQLTx(ctx context.Context, name string) (*sql.Tx, bool) {
	tx := txFromContext(ctx, name)
	if tx == nil {
		return nil, false
	}
	return tx.Tx, true
}

func (m *DBManager) DynamicExecutor(ctx context.Context, name string) (DynamicExecutor, error) {
	exec, err := m.Executor(ctx, name)
	if err != nil {
		return DynamicExecutor{}, err
	}
	return DynamicExecutor{exec: exec}, nil
}

func contextWithTx(ctx context.Context, name string, tx *sqlx.Tx) context.Context {
	value, _ := ctx.Value(txContextKey{}).(txContextValue)
	next := txContextValue{
		entries: make([]txEntry, 0, len(value.entries)+1),
	}
	next.entries = append(next.entries, value.entries...)
	next.entries = append(next.entries, txEntry{dbName: name, tx: tx})
	return context.WithValue(ctx, txContextKey{}, next)
}

func txFromContext(ctx context.Context, name string) *sqlx.Tx {
	value, ok := ctx.Value(txContextKey{}).(txContextValue)
	if !ok {
		return nil
	}

	for i := len(value.entries) - 1; i >= 0; i-- {
		entry := value.entries[i]
		if entry.dbName == name {
			return entry.tx
		}
	}
	return nil
}

func WithTx(ctx context.Context, name string, fn func(context.Context) error) error {
	return DefaultManager.WithTx(ctx, name, fn)
}

func ExecutorFor(ctx context.Context, name string) (Executor, error) {
	return DefaultManager.Executor(ctx, name)
}

func DynamicExecutorFor(ctx context.Context, name string) (DynamicExecutor, error) {
	return DefaultManager.DynamicExecutor(ctx, name)
}

func SQLDBFor(name string) (*sql.DB, error) {
	return DefaultManager.SQLDB(name)
}

func SQLTxFor(ctx context.Context, name string) (*sql.Tx, bool) {
	return DefaultManager.SQLTx(ctx, name)
}

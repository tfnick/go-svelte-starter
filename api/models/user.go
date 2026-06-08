package models

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/tfnick/go-svelte-starter/api/db"
	"github.com/tfnick/go-svelte-starter/api/framework/data/modelerror"
	"github.com/tfnick/go-svelte-starter/api/framework/data/namelookup"
	"github.com/tfnick/go-svelte-starter/api/framework/timefmt"
	"github.com/tfnick/sqlx"
)

type User struct {
	ID                  string `json:"id" db:"id"`
	Name                string `json:"name" db:"name"`
	Email               string `json:"email" db:"email"`
	PasswordHash        string `json:"-" db:"password_hash"`
	EmailVerified       int    `json:"email_verified,omitempty" db:"email_verified"`
	IsActive            int    `json:"is_active,omitempty" db:"is_active"`
	IsAdmin             int    `json:"is_admin,omitempty" db:"is_admin"`
	MembershipLevel     string `json:"membership_level" db:"membership_level"`
	MembershipExpiresAt string `json:"membership_expires_at" db:"membership_expires_at"`
	CreatedAt           string `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt           string `json:"updated_at,omitempty" db:"updated_at"`
}

type UserQuery struct {
	ID    string `db:"id"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

func CreateUser(ctx context.Context, user *User) error {
	if user.ID == "" {
		user.ID = uuid.Must(uuid.NewV7()).String()
	}
	user.CreatedAt = timefmt.NowSQLiteDateTime()
	user.UpdatedAt = user.CreatedAt

	eng, err := db.DynamicExecutorFor(ctx, "app")
	if err != nil {
		return fmt.Errorf("database unavailable: %w", err)
	}

	sql := `INSERT INTO users (id, name, email, password_hash, created_at, updated_at) VALUES (:id, :name, :email, :password_hash, :created_at, :updated_at)`
	if _, err := eng.Exec(sql, user); err != nil {
		return fmt.Errorf("create user failed: %w", err)
	}
	return nil
}

func CreateOAuthUser(ctx context.Context, user *User) error {
	if user.ID == "" {
		user.ID = uuid.Must(uuid.NewV7()).String()
	}
	if user.IsActive == 0 {
		user.IsActive = 1
	}
	user.EmailVerified = 1
	user.CreatedAt = timefmt.NowSQLiteDateTime()
	user.UpdatedAt = user.CreatedAt

	eng, err := db.DynamicExecutorFor(ctx, "app")
	if err != nil {
		return fmt.Errorf("database unavailable: %w", err)
	}

	sql := `
		INSERT INTO users (
			id, name, email, password_hash, email_verified, is_active, created_at, updated_at
		) VALUES (
			:id, :name, :email, :password_hash, :email_verified, :is_active, :created_at, :updated_at
		)
	`
	if _, err := eng.Exec(sql, user); err != nil {
		return fmt.Errorf("create oauth user failed: %w", err)
	}
	return nil
}

func GetUserByID(ctx context.Context, id string) (*User, error) {
	eng, err := db.DynamicExecutorFor(ctx, "app")
	if err != nil {
		return nil, fmt.Errorf("database unavailable: %w", err)
	}

	sql := `SELECT * FROM users WHERE id = :id`
	var user User
	err = eng.Get(&user, sql, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return nil, fmt.Errorf("get user failed: %w", err)
	}
	return &user, nil
}

func GetUserByEmail(ctx context.Context, email string) (*User, error) {
	eng, err := db.DynamicExecutorFor(ctx, "app")
	if err != nil {
		return nil, fmt.Errorf("database unavailable: %w", err)
	}

	sql := `SELECT * FROM users WHERE email = :email`
	var user User
	err = eng.Get(&user, sql, map[string]interface{}{
		"email": email,
	})
	if err != nil {
		return nil, fmt.Errorf("get user failed: %w", err)
	}
	return &user, nil
}

func GetUserByEmailOptional(ctx context.Context, email string) (*User, error) {
	eng, err := db.DynamicExecutorFor(ctx, "app")
	if err != nil {
		return nil, fmt.Errorf("database unavailable: %w", err)
	}

	sqlStmt := `SELECT * FROM users WHERE LOWER(email) = LOWER(:email) LIMIT 1`
	var user User
	err = eng.Get(&user, sqlStmt, map[string]interface{}{
		"email": email,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by email failed: %w", err)
	}
	return &user, nil
}

func GetAllUsers(ctx context.Context) ([]User, error) {
	eng, err := db.DynamicExecutorFor(ctx, "app")
	if err != nil {
		return nil, fmt.Errorf("database unavailable: %w", err)
	}

	sql := `
		SELECT id, name, email, email_verified, is_active, is_admin, membership_level, membership_expires_at, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`
	var users []User
	if err := eng.Select(&users, sql, UserQuery{}); err != nil {
		return nil, fmt.Errorf("get users failed: %w", err)
	}
	return users, nil
}

func CountUsers(ctx context.Context) (int, error) {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return 0, fmt.Errorf("database unavailable: %w", err)
	}

	var count int
	if err := d.Get(&count, `SELECT COUNT(*) FROM users`); err != nil {
		return 0, fmt.Errorf("count users failed: %w", err)
	}
	return count, nil
}

func ListUsers(ctx context.Context, limit int, offset int) ([]User, error) {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return nil, fmt.Errorf("database unavailable: %w", err)
	}

	query := d.Rebind(`
		SELECT id, name, email, email_verified, is_active, is_admin, membership_level, membership_expires_at, created_at, updated_at
		FROM users
		ORDER BY created_at DESC, id DESC
		LIMIT ? OFFSET ?
	`)
	var users []User
	if err := d.Select(&users, query, limit, offset); err != nil {
		return nil, fmt.Errorf("list users failed: %w", err)
	}
	return users, nil
}

func GetUserDisplayNamesByIDs(ctx context.Context, ids []string) (map[string]string, error) {
	uniqueIDs := namelookup.UniqueNonEmpty(ids)
	if len(uniqueIDs) == 0 {
		return map[string]string{}, nil
	}

	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return nil, fmt.Errorf("database unavailable: %w", err)
	}

	query, args, err := sqlx.In(`SELECT id, name FROM users WHERE id IN (?)`, uniqueIDs)
	if err != nil {
		return nil, fmt.Errorf("build user name query failed: %w", err)
	}
	query = d.Rebind(query)

	var rows []namelookup.Row
	if err := d.Select(&rows, query, args...); err != nil {
		return nil, fmt.Errorf("query user names failed: %w", err)
	}
	return namelookup.RowsToMap(rows), nil
}

func UpdateUser(ctx context.Context, user *User) error {
	user.UpdatedAt = timefmt.NowSQLiteDateTime()

	eng, err := db.DynamicExecutorFor(ctx, "app")
	if err != nil {
		return fmt.Errorf("database unavailable: %w", err)
	}

	sql := `
		UPDATE users SET
			updated_at = :updated_at
			#[ , name = :name ]
			#[ , email = :email ]
		WHERE id = :id
		`
	result, err := eng.Exec(sql, user)
	if err != nil {
		return fmt.Errorf("update user failed: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found: %w", modelerror.ErrNotFound)
	}
	return nil
}

func SetUserActive(ctx context.Context, id string, active bool) error {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return fmt.Errorf("database unavailable: %w", err)
	}

	value := 0
	if active {
		value = 1
	}

	query := d.Rebind(`UPDATE users SET is_active = ?, updated_at = ? WHERE id = ?`)
	result, err := d.Exec(query, value, timefmt.NowSQLiteDateTime(), id)
	if err != nil {
		return fmt.Errorf("set user active failed: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows failed: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found: %w", modelerror.ErrNotFound)
	}
	return nil
}

func UpdateUserMembership(ctx context.Context, userID string, level string, expiresAt string) error {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return fmt.Errorf("database unavailable: %w", err)
	}

	query := d.Rebind(`
		UPDATE users
		SET membership_level = ?, membership_expires_at = ?, updated_at = ?
		WHERE id = ?
	`)
	result, err := d.Exec(query, level, expiresAt, timefmt.NowSQLiteDateTime(), userID)
	if err != nil {
		return fmt.Errorf("update user membership failed: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows failed: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found: %w", modelerror.ErrNotFound)
	}
	return nil
}

func DeleteUser(ctx context.Context, id string) error {
	eng, err := db.DynamicExecutorFor(ctx, "app")
	if err != nil {
		return fmt.Errorf("database unavailable: %w", err)
	}

	sql := `DELETE FROM users WHERE id = :id`
	result, err := eng.Exec(sql, map[string]interface{}{
		"id": id,
	})
	if err != nil {
		return fmt.Errorf("delete user failed: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found: %w", modelerror.ErrNotFound)
	}
	return nil
}

func FindUsers(ctx context.Context, query UserQuery) ([]User, error) {
	eng, err := db.DynamicExecutorFor(ctx, "app")
	if err != nil {
		return nil, fmt.Errorf("database unavailable: %w", err)
	}

	sql := `
		SELECT * FROM users
		WHERE 1=1
			#[ AND id = :id ]
			#[ AND name LIKE :name ]
			#[ AND email LIKE :email ]
		ORDER BY created_at DESC
		`
	var users []User
	if err := eng.Select(&users, sql, query); err != nil {
		return nil, fmt.Errorf("find users failed: %w", err)
	}
	return users, nil
}

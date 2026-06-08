package models

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/tfnick/go-svelte-starter/api/db"
)

type OpenAPIKey struct {
	ID          string `db:"id"`
	PartnerID   string `db:"partner_id"`
	AccountID   string `db:"account_id"`
	TokenHash   string `db:"token_hash"`
	Environment string `db:"environment"`
	Scopes      string `db:"scopes"`
	Status      string `db:"status"`
}

type OpenAPIConsumer struct {
	KeyID       string
	PartnerID   string
	AccountID   string
	Scopes      []string
	Environment string
}

func ResolveOpenAPIConsumer(ctx context.Context, rawKey string) (*OpenAPIConsumer, error) {
	tokenHash := sha256.Sum256([]byte(rawKey))
	key, err := GetOpenAPIKeyByTokenHash(ctx, hex.EncodeToString(tokenHash[:]))
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, fmt.Errorf("open api key not found")
	}
	if err := ValidateOpenAPIKey(key); err != nil {
		return nil, err
	}

	return &OpenAPIConsumer{
		KeyID:       key.ID,
		PartnerID:   key.PartnerID,
		AccountID:   key.AccountID,
		Scopes:      splitScopes(key.Scopes),
		Environment: key.Environment,
	}, nil
}

func GetOpenAPIKeyByTokenHash(ctx context.Context, tokenHash string) (*OpenAPIKey, error) {
	d, err := db.ExecutorFor(ctx, "app")
	if err != nil {
		return nil, fmt.Errorf("database unavailable: %w", err)
	}

	query := d.Rebind(`
		SELECT
			k.id,
			k.partner_id,
			p.account_id,
			k.token_hash,
			k.environment,
			k.scopes,
			k.status
		FROM open_api_keys k
		JOIN open_api_partners p ON p.id = k.partner_id
		WHERE k.token_hash = ?
		  AND k.revoked_at IS NULL
		  AND (k.expires_at IS NULL OR k.expires_at > CURRENT_TIMESTAMP)
	`)

	var key OpenAPIKey
	if err := d.Get(&key, query, tokenHash); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query open api key failed: %w", err)
	}
	return &key, nil
}

func ValidateOpenAPIKey(key *OpenAPIKey) error {
	if key.Status != "active" {
		return fmt.Errorf("open api key is not active")
	}
	if key.AccountID == "" {
		return fmt.Errorf("open api key has no account binding")
	}
	if !hasScope(splitScopes(key.Scopes), "account:read") {
		return fmt.Errorf("open api key missing required scope")
	}
	return nil
}

func splitScopes(scopes string) []string {
	if scopes == "" {
		return nil
	}

	parts := strings.Split(scopes, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		scope := strings.TrimSpace(part)
		if scope != "" {
			result = append(result, scope)
		}
	}
	return result
}

func hasScope(scopes []string, required string) bool {
	for _, scope := range scopes {
		if scope == required {
			return true
		}
	}
	return false
}

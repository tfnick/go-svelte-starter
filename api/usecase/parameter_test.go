package usecase_test

import (
	"testing"

	"github.com/tfnick/sqlx"
	fwusecase "github.com/tfnick/go-svelte-starter/api/framework/usecase"
	"github.com/tfnick/go-svelte-starter/api/models"
	"github.com/tfnick/go-svelte-starter/api/usecase"
)

func TestCreateParameterIntegrationChannelStoresCredentialValueAndReturnsAdminCo(t *testing.T) {
	manager := setupUsecaseOrderTxDB(t)
	appDB, err := manager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}

	credentialValue := `{"api_key":"sk_param","webhook_secret":"whsec_param"}`
	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	channel, err := usecase.CreateParameterIntegrationChannel(ctx, usecase.SaveParameterIntegrationChannelCmd{
		Scenario:        models.IntegrationScenarioPayment,
		ChannelCode:     "param-creem",
		ProviderCode:    "creem",
		AdapterKey:      "payment.creem.hosted_checkout",
		Environment:     "test",
		Enabled:         true,
		Priority:        10,
		WebhookEnabled:  true,
		ConfigJSON:      `{"base_url":"https://api.creem.io","product_id":"prod_1"}`,
		MetadataJSON:    `{"owner":"finance"}`,
		CredentialType:  "payment_bundle",
		CredentialValue: credentialValue,
	})
	if err != nil {
		t.Fatalf("create parameter channel: %v", err)
	}
	if channel.ID == "" || channel.CredentialValue != credentialValue || channel.ConfigJSON == "" {
		t.Fatalf("unexpected channel co: %#v", channel)
	}

	var row struct {
		ValueText   string `db:"value_text"`
		Ciphertext  string `db:"ciphertext"`
		MaskedValue string `db:"masked_value"`
	}
	if err := appDB.Get(&row, `
		SELECT cred.value_text, cred.ciphertext, cred.masked_value
		FROM integration_credentials cred
		INNER JOIN integration_channels channel ON channel.credential_id = cred.id
		WHERE channel.id = ?
	`, channel.ID); err != nil {
		t.Fatalf("load stored credential: %v", err)
	}
	if row.ValueText != credentialValue {
		t.Fatalf("expected credential value to be stored plainly, got %#v", row)
	}
	if row.Ciphertext != credentialValue || row.MaskedValue != "" {
		t.Fatalf("expected legacy credential columns to be compatibility-only, got %#v", row)
	}
}

func TestUpdateParameterIntegrationChannelPreservesOrUpdatesCredentialValue(t *testing.T) {
	manager := setupUsecaseOrderTxDB(t)
	appDB, err := manager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}
	seedParameterChannel(t, appDB)

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	before := loadParameterCredentialValue(t, appDB, "param-sms-channel")
	updated, err := usecase.UpdateParameterIntegrationChannel(ctx, usecase.SaveParameterIntegrationChannelCmd{
		ID:             "param-sms-channel",
		Scenario:       models.IntegrationScenarioSMS,
		ChannelCode:    "param-sms",
		ProviderCode:   "aliyun",
		AdapterKey:     "sms.aliyun.adapter",
		Environment:    "test",
		Enabled:        true,
		Priority:       40,
		WebhookEnabled: false,
		ConfigJSON:     `{"base_url":"https://sms.example.com"}`,
		MetadataJSON:   `{"owner":"ops"}`,
		CredentialType: "api_key",
	})
	if err != nil {
		t.Fatalf("update without credential value: %v", err)
	}
	if updated.CredentialValue != "sms-secret" {
		t.Fatalf("expected existing credential value, got %#v", updated)
	}
	if after := loadParameterCredentialValue(t, appDB, "param-sms-channel"); after != before {
		t.Fatal("expected credential value to be preserved when request value is empty")
	}

	changed, err := usecase.UpdateParameterIntegrationChannel(ctx, usecase.SaveParameterIntegrationChannelCmd{
		ID:              "param-sms-channel",
		Scenario:        models.IntegrationScenarioSMS,
		ChannelCode:     "param-sms",
		ProviderCode:    "aliyun",
		AdapterKey:      "sms.aliyun.adapter",
		Environment:     "test",
		Enabled:         true,
		Priority:        50,
		WebhookEnabled:  true,
		ConfigJSON:      `{"base_url":"https://sms-rotated.example.com"}`,
		MetadataJSON:    `{"owner":"platform"}`,
		CredentialType:  "api_key",
		CredentialValue: "new-sms-secret",
	})
	if err != nil {
		t.Fatalf("update with credential value: %v", err)
	}
	if changed.CredentialValue != "new-sms-secret" {
		t.Fatalf("expected updated credential value, got %#v", changed)
	}
	if afterChange := loadParameterCredentialValue(t, appDB, "param-sms-channel"); afterChange != "new-sms-secret" {
		t.Fatalf("unexpected stored credential value: %q", afterChange)
	}
}

func TestParameterIntegrationChannelRejectsSensitiveConfigKeys(t *testing.T) {
	setupUsecaseOrderTxDB(t)

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	_, err := usecase.CreateParameterIntegrationChannel(ctx, usecase.SaveParameterIntegrationChannelCmd{
		Scenario:        models.IntegrationScenarioLLM,
		ChannelCode:     "bad-deepseek",
		ProviderCode:    "deepseek",
		AdapterKey:      "llm.deepseek.openai_compatible",
		Enabled:         true,
		ConfigJSON:      `{"api_key":"should-not-be-here"}`,
		MetadataJSON:    "{}",
		CredentialType:  "api_key",
		CredentialValue: "secret",
	})
	if fwusecase.CodeOf(err) != fwusecase.CodeValidation {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestParameterIntegrationChannelRejectsSensitiveConfigKeysInsideArrays(t *testing.T) {
	setupUsecaseOrderTxDB(t)

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	_, err := usecase.CreateParameterIntegrationChannel(ctx, usecase.SaveParameterIntegrationChannelCmd{
		Scenario:        models.IntegrationScenarioSMS,
		ChannelCode:     "bad-sms-array",
		ProviderCode:    "aliyun",
		AdapterKey:      "sms.aliyun.adapter",
		Enabled:         true,
		ConfigJSON:      `{"items":[{"api_key":"should-not-be-here"}]}`,
		MetadataJSON:    "{}",
		CredentialType:  "api_key",
		CredentialValue: "secret",
	})
	if fwusecase.CodeOf(err) != fwusecase.CodeValidation {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestParameterIntegrationChannelRejectsCredentialTypeOutsideDictionary(t *testing.T) {
	setupUsecaseOrderTxDB(t)

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	_, err := usecase.CreateParameterIntegrationChannel(ctx, usecase.SaveParameterIntegrationChannelCmd{
		Scenario:        models.IntegrationScenarioLLM,
		ChannelCode:     "bad-credential-type",
		ProviderCode:    "custom",
		AdapterKey:      "custom.llm.adapter",
		Enabled:         true,
		ConfigJSON:      "{}",
		MetadataJSON:    "{}",
		CredentialType:  "raw_password",
		CredentialValue: "secret",
	})
	if fwusecase.CodeOf(err) != fwusecase.CodeValidation {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestListParameterIntegrationSchemasFiltersByScenario(t *testing.T) {
	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)

	schemas, err := usecase.ListParameterIntegrationSchemas(ctx, usecase.ListParameterIntegrationSchemasQry{
		Scenario: models.IntegrationScenarioPayment,
	})
	if err != nil {
		t.Fatalf("list schemas: %v", err)
	}
	if len(schemas) != 1 {
		t.Fatalf("expected one payment schema, got %#v", schemas)
	}
	if schemas[0].AdapterKey != "payment.creem.hosted_checkout" {
		t.Fatalf("unexpected schema: %#v", schemas[0])
	}
	if schemas[0].CredentialFormat != usecase.ParameterIntegrationCredentialFormatJSONObject {
		t.Fatalf("expected json object credential schema, got %#v", schemas[0])
	}
	if len(schemas[0].ConfigFields) == 0 || len(schemas[0].CredentialFields) == 0 {
		t.Fatalf("expected config and credential fields, got %#v", schemas[0])
	}
	baseURLField := schemas[0].ConfigFields[0]
	if baseURLField.Key != "base_url" || baseURLField.Kind != usecase.ParameterIntegrationSchemaFieldURL {
		t.Fatalf("expected payment API URL field, got %#v", baseURLField)
	}
	if baseURLField.DictionaryType != "" || len(baseURLField.Options) != 0 {
		t.Fatalf("payment API URL must be a free URL input, got %#v", baseURLField)
	}

	smsSchemas, err := usecase.ListParameterIntegrationSchemas(ctx, usecase.ListParameterIntegrationSchemasQry{
		Scenario: models.IntegrationScenarioSMS,
	})
	if err != nil {
		t.Fatalf("list SMS schemas: %v", err)
	}
	if len(smsSchemas) != 1 {
		t.Fatalf("expected one SMS schema, got %#v", smsSchemas)
	}
	if smsSchemas[0].AdapterKey != "sms.aliyun.adapter" {
		t.Fatalf("unexpected SMS schema: %#v", smsSchemas[0])
	}
	if smsSchemas[0].ProviderCode != "aliyun" || smsSchemas[0].CredentialFormat != usecase.ParameterIntegrationCredentialFormatPlain {
		t.Fatalf("unexpected SMS schema contract: %#v", smsSchemas[0])
	}

	emailSchemas, err := usecase.ListParameterIntegrationSchemas(ctx, usecase.ListParameterIntegrationSchemasQry{
		Scenario: models.IntegrationScenarioEmail,
	})
	if err != nil {
		t.Fatalf("list Email schemas: %v", err)
	}
	if len(emailSchemas) != 2 {
		t.Fatalf("expected two Email schemas, got %#v", emailSchemas)
	}
	if emailSchemas[0].AdapterKey != "email.aliyun.smtp" || emailSchemas[1].AdapterKey != "email.resend.api" {
		t.Fatalf("unexpected Email schemas: %#v", emailSchemas)
	}
	if emailSchemas[0].CredentialType != "smtp_password" || emailSchemas[0].CredentialFormat != usecase.ParameterIntegrationCredentialFormatJSONObject {
		t.Fatalf("unexpected Aliyun Email schema: %#v", emailSchemas[0])
	}
	if emailSchemas[1].CredentialType != "api_key" || emailSchemas[1].CredentialFormat != usecase.ParameterIntegrationCredentialFormatPlain {
		t.Fatalf("unexpected Resend Email schema: %#v", emailSchemas[1])
	}
}

func TestParameterIntegrationChannelValidatesAdapterSchema(t *testing.T) {
	setupUsecaseOrderTxDB(t)
	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)

	base := usecase.SaveParameterIntegrationChannelCmd{
		Scenario:        models.IntegrationScenarioPayment,
		ChannelCode:     "schema-creem",
		ProviderCode:    "creem",
		AdapterKey:      "payment.creem.hosted_checkout",
		Environment:     "test",
		Enabled:         true,
		ConfigJSON:      `{"base_url":"https://api.creem.io","product_id":"prod_schema"}`,
		MetadataJSON:    "{}",
		CredentialType:  "payment_bundle",
		CredentialValue: `{"api_key":"sk_schema","webhook_secret":"whsec_schema"}`,
	}

	missingProduct := base
	missingProduct.ChannelCode = "schema-missing-product"
	missingProduct.ConfigJSON = `{"base_url":"https://api.creem.io"}`
	if _, err := usecase.CreateParameterIntegrationChannel(ctx, missingProduct); fwusecase.CodeOf(err) != fwusecase.CodeValidation {
		t.Fatalf("expected missing product_id validation, got %v", err)
	}

	invalidURL := base
	invalidURL.ChannelCode = "schema-invalid-url"
	invalidURL.ConfigJSON = `{"base_url":"creem","product_id":"prod_schema"}`
	if _, err := usecase.CreateParameterIntegrationChannel(ctx, invalidURL); fwusecase.CodeOf(err) != fwusecase.CodeValidation {
		t.Fatalf("expected invalid URL validation, got %v", err)
	}

	missingSecret := base
	missingSecret.ChannelCode = "schema-missing-secret"
	missingSecret.CredentialValue = `{"api_key":"sk_schema"}`
	if _, err := usecase.CreateParameterIntegrationChannel(ctx, missingSecret); fwusecase.CodeOf(err) != fwusecase.CodeValidation {
		t.Fatalf("expected missing webhook secret validation, got %v", err)
	}

	if _, err := usecase.CreateParameterIntegrationChannel(ctx, base); err != nil {
		t.Fatalf("expected schema-valid channel to be created: %v", err)
	}
}

func TestParameterIntegrationChannelAcceptsPlainCredentialSchema(t *testing.T) {
	setupUsecaseOrderTxDB(t)
	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)

	channel, err := usecase.CreateParameterIntegrationChannel(ctx, usecase.SaveParameterIntegrationChannelCmd{
		Scenario:        models.IntegrationScenarioLLM,
		ChannelCode:     "schema-deepseek",
		ProviderCode:    "deepseek",
		AdapterKey:      "llm.deepseek.openai_compatible",
		Environment:     "test",
		Enabled:         true,
		ConfigJSON:      `{"base_url":"https://api.deepseek.com"}`,
		MetadataJSON:    "{}",
		CredentialType:  "api_key",
		CredentialValue: "sk_deepseek",
	})
	if err != nil {
		t.Fatalf("create LLM channel: %v", err)
	}
	if channel.CredentialType != "api_key" || channel.CredentialValue != "sk_deepseek" {
		t.Fatalf("unexpected channel: %#v", channel)
	}
}

func TestParameterIntegrationChannelAcceptsEmailSMTPJSONCredentialSchema(t *testing.T) {
	setupUsecaseOrderTxDB(t)
	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)

	channel, err := usecase.CreateParameterIntegrationChannel(ctx, usecase.SaveParameterIntegrationChannelCmd{
		Scenario:       models.IntegrationScenarioEmail,
		ChannelCode:    "aliyun-email",
		ProviderCode:   "aliyun",
		AdapterKey:     "email.aliyun.smtp",
		Environment:    "production",
		Enabled:        true,
		ConfigJSON:     `{"smtp_host":"smtp.qiye.aliyun.com","smtp_port":465,"security":"ssl","from_email":"noreply@example.com"}`,
		MetadataJSON:   "{}",
		CredentialType: "smtp_password",
		CredentialValue: `{
			"username":"noreply@example.com",
			"password":"mailbox-secret"
		}`,
	})
	if err != nil {
		t.Fatalf("create Aliyun Email channel: %v", err)
	}
	if channel.Scenario != models.IntegrationScenarioEmail || channel.CredentialType != "smtp_password" {
		t.Fatalf("unexpected Email channel: %#v", channel)
	}

	badSecurity := usecase.SaveParameterIntegrationChannelCmd{
		Scenario:        models.IntegrationScenarioEmail,
		ChannelCode:     "bad-email-security",
		ProviderCode:    "aliyun",
		AdapterKey:      "email.aliyun.smtp",
		Environment:     "production",
		Enabled:         true,
		ConfigJSON:      `{"smtp_host":"smtp.qiye.aliyun.com","smtp_port":465,"security":"invalid","from_email":"noreply@example.com"}`,
		MetadataJSON:    "{}",
		CredentialType:  "smtp_password",
		CredentialValue: `{"username":"noreply@example.com","password":"mailbox-secret"}`,
	}
	if _, err := usecase.CreateParameterIntegrationChannel(ctx, badSecurity); fwusecase.CodeOf(err) != fwusecase.CodeValidation {
		t.Fatalf("expected invalid SMTP security validation, got %v", err)
	}
}

func TestParameterIntegrationChannelAcceptsEmailResendPlainCredentialSchema(t *testing.T) {
	setupUsecaseOrderTxDB(t)
	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)

	channel, err := usecase.CreateParameterIntegrationChannel(ctx, usecase.SaveParameterIntegrationChannelCmd{
		Scenario:        models.IntegrationScenarioEmail,
		ChannelCode:     "resend-email",
		ProviderCode:    "resend",
		AdapterKey:      "email.resend.api",
		Environment:     "production",
		Enabled:         true,
		ConfigJSON:      `{"base_url":"https://api.resend.com","from_email":"noreply@example.com"}`,
		MetadataJSON:    "{}",
		CredentialType:  "api_key",
		CredentialValue: "re_secret",
	})
	if err != nil {
		t.Fatalf("create Resend Email channel: %v", err)
	}
	if channel.CredentialType != "api_key" || channel.CredentialValue != "re_secret" {
		t.Fatalf("unexpected Resend Email channel: %#v", channel)
	}
}

func TestSetParameterIntegrationChannelEnabled(t *testing.T) {
	manager := setupUsecaseOrderTxDB(t)
	appDB, err := manager.GetDB("app")
	if err != nil {
		t.Fatalf("get app db: %v", err)
	}
	seedParameterChannel(t, appDB)

	ctx := fwusecase.NewContext(t.Context(), fwusecase.SurfaceInternalAPI)
	channel, err := usecase.SetParameterIntegrationChannelEnabled(ctx, usecase.SetParameterIntegrationChannelEnabledCmd{
		ID:      "param-sms-channel",
		Enabled: false,
	})
	if err != nil {
		t.Fatalf("disable channel: %v", err)
	}
	if channel.Enabled {
		t.Fatalf("expected disabled channel, got %#v", channel)
	}
}

func seedParameterChannel(t *testing.T, appDB *sqlx.DB) {
	t.Helper()

	if _, err := appDB.Exec(appDB.Rebind(`
		INSERT INTO integration_credentials (id, credential_type, ciphertext, key_version, masked_value, value_text, enabled)
		VALUES (?, 'api_key', 'sms-secret', '', '', 'sms-secret', 1)
	`), "param-sms-credential"); err != nil {
		t.Fatalf("insert credential: %v", err)
	}
	if _, err := appDB.Exec(appDB.Rebind(`
		INSERT INTO integration_channels (
			id, scenario, channel_code, provider_code, adapter_key, environment, enabled,
			priority, credential_id, webhook_enabled, config_json, metadata_json
		) VALUES (?, 'sms', 'param-sms', 'aliyun', 'sms.aliyun.adapter', 'test', 1,
			30, ?, 0, '{"base_url":"https://sms.example.com"}', '{"owner":"ops"}')
	`), "param-sms-channel", "param-sms-credential"); err != nil {
		t.Fatalf("insert channel: %v", err)
	}
}

func loadParameterCredentialValue(t *testing.T, appDB *sqlx.DB, channelID string) string {
	t.Helper()

	var value string
	if err := appDB.Get(&value, `
		SELECT cred.value_text
		FROM integration_credentials cred
		INNER JOIN integration_channels channel ON channel.credential_id = cred.id
		WHERE channel.id = ?
	`, channelID); err != nil {
		t.Fatalf("load credential value: %v", err)
	}
	return value
}

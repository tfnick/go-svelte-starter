<script>
  import { onMount } from 'svelte';

  import {
    createParameterIntegrationChannel,
    getDictionaries,
    listParameterIntegrationChannels,
    listParameterIntegrationSchemas,
    setParameterIntegrationChannelEnabled,
    updateParameterIntegrationChannel
  } from '../api.js';
  import Notice from '../components/Notice.svelte';
  import { formatLocalDateTime } from '../helpers/dateTime.js';

  const scenarios = [
    { key: 'payment', label: 'Payment', defaultProvider: 'creem', defaultAdapter: 'payment.creem.hosted_checkout', credentialType: 'payment_bundle' },
    { key: 'llm', label: 'LLM', defaultProvider: 'deepseek', defaultAdapter: 'llm.deepseek.openai_compatible', credentialType: 'api_key' },
    { key: 'sms', label: 'SMS', defaultProvider: 'aliyun', defaultAdapter: 'sms.aliyun.adapter', credentialType: 'api_key' },
    { key: 'email', label: 'Email', defaultProvider: 'aliyun', defaultAdapter: 'email.aliyun.smtp', credentialType: 'smtp_password' },
    { key: 'oss', label: 'OSS', defaultProvider: 'cloudflare_r2', defaultAdapter: 'oss.cloudflare_r2.s3_compatible', credentialType: 's3_access_key' }
  ];

  const defaultJSON = '{}';
  const environmentDictionaryType = 'integration_environment';
  const credentialTypeDictionaryType = 'integration_credential_type';
  const fallbackEnvironmentOptions = [
    { value: 'test', label: 'Test' },
    { value: 'production', label: 'Production' }
  ];
  const fallbackCredentialTypeOptions = [
    { value: 'payment_bundle', label: 'Payment Bundle' },
    { value: 'api_key', label: 'API Key' },
    { value: 'smtp_password', label: 'SMTP Password' },
    { value: 's3_access_key', label: 'S3 Access Key' }
  ];

  let activeScenario = $state('payment');
  let channelsByScenario = $state({
    payment: [],
    llm: [],
    sms: [],
    email: [],
    oss: []
  });
  let schemasByScenario = $state({
    payment: [],
    llm: [],
    sms: [],
    email: [],
    oss: []
  });
  let loadingByScenario = $state({
    payment: false,
    llm: false,
    sms: false,
    email: false,
    oss: false
  });
  let loadingSchemasByScenario = $state({
    payment: false,
    llm: false,
    sms: false,
    email: false,
    oss: false
  });
  let dictionariesByType = $state({});
  let structuredConfig = $state({});
  let structuredCredential = $state({});
  let credentialVisibility = $state({});
  let customCredentialVisible = $state(false);
  let form = $state(emptyForm('payment'));
  let saving = $state(false);
  let error = $state('');
  let message = $state('');

  onMount(() => {
    initializePage();
  });

  async function initializePage() {
    await Promise.all([loadSchemas(activeScenario), loadScenario(activeScenario)]);
    resetForm(activeScenario);
  }

  function scenarioMeta(key = activeScenario) {
    return scenarios.find((scenario) => scenario.key === key) || scenarios[0];
  }

  function emptyForm(scenario) {
    const meta = scenarioMeta(scenario);
    const schema = defaultSchemaForScenario(meta.key);
    return {
      id: '',
      scenario: meta.key,
      channel_code: '',
      provider_code: schema?.provider_code || meta.defaultProvider,
      adapter_key: schema?.adapter_key || meta.defaultAdapter,
      environment: 'production',
      enabled: true,
      priority: 100,
      webhook_enabled: false,
      config_json: configJSONFromSchema(schema),
      metadata_json: defaultJSON,
      credential_type: schema?.credential_type || meta.credentialType,
      credential_value: ''
    };
  }

  function scenarioChannels(key) {
    return channelsByScenario[key] || [];
  }

  function scenarioSchemas(key = activeScenario) {
    return schemasByScenario[key] || [];
  }

  function defaultSchemaForScenario(key) {
    return scenarioSchemas(key)[0] || null;
  }

  function currentSchema() {
    return schemaForAdapter(form.scenario, form.adapter_key);
  }

  function schemaForAdapter(scenario, adapterKey) {
    return scenarioSchemas(scenario).find((schema) => schema.adapter_key === adapterKey) || null;
  }

  function currentConfigFields() {
    return currentSchema()?.config_fields || [];
  }

  function currentCredentialFields() {
    return currentSchema()?.credential_fields || [];
  }

  function environmentOptions() {
    const options = dictionariesByType[environmentDictionaryType] || [];
    return options.length > 0 ? options : fallbackEnvironmentOptions;
  }

  function credentialTypeOptions() {
    const options = dictionariesByType[credentialTypeDictionaryType] || [];
    return options.length > 0 ? options : fallbackCredentialTypeOptions;
  }

  async function selectScenario(key) {
    if (activeScenario === key) return;
    activeScenario = key;
    await loadSchemas(key);
    resetForm(key);
    await loadScenario(key);
  }

  async function loadScenario(key) {
    loadingByScenario = { ...loadingByScenario, [key]: true };
    error = '';
    try {
      const channels = await listParameterIntegrationChannels(key);
      channelsByScenario = { ...channelsByScenario, [key]: channels };
    } catch (err) {
      error = err.message || 'Failed to load integration channels';
    } finally {
      loadingByScenario = { ...loadingByScenario, [key]: false };
    }
  }

  async function loadSchemas(key) {
    if (scenarioSchemas(key).length > 0) {
      await loadSchemaDictionaries(scenarioSchemas(key));
      return;
    }

    loadingSchemasByScenario = { ...loadingSchemasByScenario, [key]: true };
    error = '';
    try {
      const schemas = await listParameterIntegrationSchemas(key);
      schemasByScenario = { ...schemasByScenario, [key]: schemas || [] };
      await loadSchemaDictionaries(schemas || []);
    } catch (err) {
      error = err.message || 'Failed to load integration schemas';
    } finally {
      loadingSchemasByScenario = { ...loadingSchemasByScenario, [key]: false };
    }
  }

  async function loadSchemaDictionaries(schemas) {
    const types = new Set([environmentDictionaryType, credentialTypeDictionaryType]);
    for (const schema of schemas || []) {
      for (const field of [...(schema.config_fields || []), ...(schema.credential_fields || [])]) {
        if (field.dictionary_type) {
          types.add(field.dictionary_type);
        }
      }
    }

    const missing = [...types].filter((type) => !(type in dictionariesByType));
    if (missing.length === 0) return;

    const response = await getDictionaries(missing);
    dictionariesByType = {
      ...dictionariesByType,
      ...(response?.dictionaries || {})
    };
  }

  async function refreshCurrent() {
    await Promise.all([loadSchemas(activeScenario), loadScenario(activeScenario)]);
  }

  function resetForm(scenario = activeScenario) {
    form = emptyForm(scenario);
    syncStructuredStateFromForm();
  }

  function editChannel(channel) {
    activeScenario = channel.scenario;
    form = {
      id: channel.id,
      scenario: channel.scenario,
      channel_code: channel.channel_code,
      provider_code: channel.provider_code,
      adapter_key: channel.adapter_key,
      environment: channel.environment,
      enabled: Boolean(channel.enabled),
      priority: channel.priority || 100,
      webhook_enabled: Boolean(channel.webhook_enabled),
      config_json: formatJSON(channel.config_json),
      metadata_json: formatJSON(channel.metadata_json),
      credential_type: channel.credential_type,
      credential_value: channel.credential_value || ''
    };
    syncStructuredStateFromForm();
  }

  function applyAdapterKey(adapterKey) {
    form.adapter_key = adapterKey;
    const schema = currentSchema();
    if (!schema) {
      return;
    }

    form.provider_code = schema.provider_code || form.provider_code;
    form.credential_type = schema.credential_type || form.credential_type;
    form.config_json = configJSONFromSchema(schema);
    form.credential_value = '';
    syncStructuredStateFromForm();
  }

  function applyAdapterSelection(value) {
    if (value === '__custom__') {
      form.adapter_key = '';
      form.credential_value = '';
      structuredConfig = parseJSONObject(form.config_json);
      structuredCredential = {};
      credentialVisibility = {};
      return;
    }
    applyAdapterKey(value);
  }

  function adapterSelectValue() {
    return currentSchema() ? form.adapter_key : '__custom__';
  }

  async function saveChannel() {
    saving = true;
    error = '';
    message = '';

    const payload = {
      scenario: form.scenario,
      channel_code: form.channel_code,
      provider_code: form.provider_code,
      adapter_key: form.adapter_key,
      environment: form.environment,
      enabled: form.enabled,
      priority: Number(form.priority) || 100,
      webhook_enabled: form.webhook_enabled,
      config_json: compactJSON(form.config_json),
      metadata_json: compactJSON(form.metadata_json),
      credential_type: form.credential_type,
      credential_value: credentialValueForSave()
    };

    try {
      const saved = form.id
        ? await updateParameterIntegrationChannel(form.id, payload)
        : await createParameterIntegrationChannel(payload);
      message = form.id ? 'Integration channel updated' : 'Integration channel created';
      editChannel(saved);
      await loadScenario(saved.scenario);
    } catch (err) {
      error = err.message || 'Failed to save integration channel';
    } finally {
      saving = false;
    }
  }

  async function toggleChannel(channel) {
    error = '';
    message = '';
    try {
      const updated = await setParameterIntegrationChannelEnabled(channel.id, !channel.enabled);
      message = updated.enabled ? 'Integration channel enabled' : 'Integration channel disabled';
      await loadScenario(channel.scenario);
      if (form.id === channel.id) {
        editChannel(updated);
      }
    } catch (err) {
      error = err.message || 'Failed to update integration channel';
    }
  }

  function syncStructuredStateFromForm() {
    structuredConfig = parseJSONObject(form.config_json);
    structuredCredential = parseCredentialValue(form.credential_value, currentSchema());
    credentialVisibility = {};
    customCredentialVisible = false;
  }

  function updateConfigField(field, value) {
    const base = parseJSONObject(form.config_json);
    const nextStructured = { ...structuredConfig };
    const normalized = normalizeFieldInput(field, value);

    if (isEmptyFieldInput(field, value)) {
      delete base[field.key];
      delete nextStructured[field.key];
    } else {
      base[field.key] = normalized;
      nextStructured[field.key] = normalized;
    }

    structuredConfig = nextStructured;
    form.config_json = formatJSON(JSON.stringify(base));
  }

  function updateCredentialField(field, value) {
    const nextStructured = { ...structuredCredential };
    if (String(value || '').trim() === '') {
      delete nextStructured[field.key];
    } else {
      nextStructured[field.key] = String(value);
    }

    structuredCredential = nextStructured;
    form.credential_value = credentialValueForSave();
  }

  function updatePlainCredential(value) {
    form.credential_value = String(value || '');
  }

  function credentialValueForSave() {
    const schema = currentSchema();
    if (!schema) {
      return form.credential_value;
    }

    if (schema.credential_format === 'plain') {
      const field = currentCredentialFields()[0];
      return field ? String(structuredCredential[field.key] || '').trim() : form.credential_value;
    }

    const payload = {};
    for (const [key, value] of Object.entries(structuredCredential)) {
      if (String(value || '').trim() !== '') {
        payload[key] = String(value);
      }
    }
    return Object.keys(payload).length > 0 ? JSON.stringify(payload) : '';
  }

  function parseCredentialValue(value, schema) {
    const raw = String(value || '').trim();
    if (!schema || raw === '') return {};

    if (schema.credential_format === 'plain') {
      const field = schema.credential_fields?.[0];
      return field ? { [field.key]: raw } : {};
    }

    return parseJSONObject(raw);
  }

  function credentialInputType(field) {
    if (field.kind !== 'secret') return inputTypeForField(field);
    return credentialVisibility[field.key] ? 'text' : 'password';
  }

  function toggleCredentialVisibility(field) {
    credentialVisibility = {
      ...credentialVisibility,
      [field.key]: !credentialVisibility[field.key]
    };
  }

  function configJSONFromSchema(schema) {
    const defaults = {};
    for (const field of schema?.config_fields || []) {
      if (field.default_value !== '') {
        defaults[field.key] = normalizeFieldInput(field, field.default_value);
      }
    }
    return formatJSON(JSON.stringify(defaults));
  }

  function fieldOptions(field) {
    const dictionaryOptions = field.dictionary_type ? dictionariesByType[field.dictionary_type] || [] : [];
    return dictionaryOptions.length > 0 ? dictionaryOptions : field.options || [];
  }

  function inputTypeForField(field) {
    if (field.kind === 'number') return 'number';
    if (field.kind === 'url') return 'url';
    if (field.kind === 'secret') return 'password';
    return 'text';
  }

  function normalizeFieldInput(field, value) {
    if (field.kind === 'boolean') {
      return Boolean(value);
    }
    if (field.kind === 'number') {
      const numberValue = Number(value);
      return Number.isFinite(numberValue) ? numberValue : value;
    }
    return String(value || '').trim();
  }

  function isEmptyFieldInput(field, value) {
    if (field.kind === 'boolean') {
      return false;
    }
    return String(value || '').trim() === '';
  }

  function parseJSONObject(value) {
    const trimmed = String(value || '').trim();
    if (!trimmed) return {};
    try {
      const parsed = JSON.parse(trimmed);
      return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed : {};
    } catch {
      return {};
    }
  }

  function compactJSON(value) {
    const trimmed = String(value || '').trim();
    if (!trimmed) return defaultJSON;
    try {
      return JSON.stringify(JSON.parse(trimmed));
    } catch {
      return trimmed;
    }
  }

  function formatJSON(value) {
    const trimmed = String(value || '').trim();
    if (!trimmed) return defaultJSON;
    try {
      return JSON.stringify(JSON.parse(trimmed), null, 2);
    } catch {
      return trimmed;
    }
  }

  function formatDate(value) {
    return formatLocalDateTime(value);
  }

  function webhookHelpText() {
    const scenario = String(form.scenario || '<scenario>').trim() || '<scenario>';
    const channelCode = String(form.channel_code || '<channel_code>').trim() || '<channel_code>';
    const providerCode = String(form.provider_code || '<provider_code>').trim() || '<provider_code>';
    return `Configure the provider callback URL as https://<public-domain>/api/integrations/${scenario}/${channelCode}/webhooks/${providerCode}`;
  }
</script>

<section class="space-y-6">
  <div class="flex flex-wrap items-start justify-between gap-4">
    <div>
      <h1 class="text-2xl font-bold leading-tight">Parameter</h1>
      <p class="mt-1 text-sm text-base-content/60">Channel integration settings for Payment, LLM, SMS, Email, and OSS.</p>
    </div>
    <div class="flex gap-2">
      <button class="btn btn-outline btn-sm" type="button" onclick={refreshCurrent} disabled={loadingByScenario[activeScenario] || loadingSchemasByScenario[activeScenario]}>
        {#if loadingByScenario[activeScenario] || loadingSchemasByScenario[activeScenario]}
          <span class="loading loading-spinner loading-xs"></span>
        {/if}
        Refresh
      </button>
      <button class="btn btn-primary btn-sm" type="button" onclick={() => resetForm()}>New channel</button>
    </div>
  </div>

  <Notice type="success" message={message} />
  <Notice type="error" message={error} />

  <div class="grid gap-6 xl:grid-cols-[0.52fr_1.08fr]">
    <div class="card border border-base-300 bg-base-100 shadow-sm">
      <div class="card-body gap-4">
        <div class="flex items-center justify-between gap-3">
          <h2 class="card-title text-lg">{form.id ? 'Edit channel' : 'Create channel'}</h2>
          <span class="badge badge-outline">{scenarioMeta(form.scenario).label}</span>
        </div>

        {#if form.id}
          <div class="max-w-full truncate rounded border border-base-300 px-3 py-2 font-mono text-xs text-base-content/60">
            {form.id}
          </div>
        {/if}

        <div class="grid gap-3 sm:grid-cols-2">
          <label class="form-control">
            <span class="label"><span class="label-text">Channel code</span></span>
            <input class="input input-bordered font-mono text-sm" bind:value={form.channel_code} placeholder="creem-prod" />
          </label>

          <label class="form-control">
            <span class="label"><span class="label-text">Provider code</span></span>
            <input class="input input-bordered font-mono text-sm" bind:value={form.provider_code} placeholder="creem" />
          </label>
        </div>

        <label class="form-control">
          <span class="label"><span class="label-text">Adapter key</span></span>
          {#if scenarioSchemas(form.scenario).length > 0}
            <select class="select select-bordered font-mono text-sm" value={adapterSelectValue()} onchange={(event) => applyAdapterSelection(event.currentTarget.value)}>
              {#each scenarioSchemas(form.scenario) as schema}
                <option value={schema.adapter_key}>{schema.label}</option>
              {/each}
              <option value="__custom__">Custom adapter</option>
            </select>
            {#if !currentSchema()}
              <input class="input input-bordered font-mono text-sm" bind:value={form.adapter_key} placeholder="custom.adapter.key" />
            {/if}
          {:else}
            <input class="input input-bordered font-mono text-sm" bind:value={form.adapter_key} />
          {/if}
        </label>

        <div class="grid gap-3 sm:grid-cols-[0.64fr_0.36fr]">
          <label class="form-control">
            <span class="label"><span class="label-text">Environment</span></span>
            <select class="select select-bordered font-mono text-sm" bind:value={form.environment}>
              {#each environmentOptions() as option}
                <option value={option.value}>{option.label}</option>
              {/each}
            </select>
          </label>

          <label class="form-control">
            <span class="label"><span class="label-text">Priority</span></span>
            <input class="input input-bordered" type="number" min="1" bind:value={form.priority} />
          </label>
        </div>

        <div class="grid gap-3 sm:grid-cols-2">
          <label class="label cursor-pointer justify-start gap-3 rounded border border-base-300 px-3">
            <input class="toggle toggle-primary" type="checkbox" bind:checked={form.enabled} />
            <span class="label-text">Enabled</span>
          </label>

          <label class="label cursor-pointer justify-start gap-3 rounded border border-base-300 px-3">
            <input class="toggle toggle-primary" type="checkbox" bind:checked={form.webhook_enabled} />
            <span class="label-text inline-flex items-center gap-1.5">
              <span>Webhook</span>
              <span class="tooltip tooltip-right" data-tip={webhookHelpText()}>
                <button
                  class="inline-flex h-4 w-4 items-center justify-center rounded-full border border-base-content/30 bg-transparent text-[10px] font-semibold leading-none text-base-content/60"
                  aria-label={webhookHelpText()}
                  type="button"
                >
                  ?
                </button>
              </span>
            </span>
          </label>
        </div>

        {#if currentSchema()}
          <div class="rounded border border-base-300 p-3">
            <div class="mb-2 flex items-center justify-between gap-3">
              <h3 class="text-sm font-semibold">Config</h3>
              <span class="badge badge-ghost max-w-52 truncate">{currentSchema().label}</span>
            </div>
            <div class="grid gap-3">
              {#each currentConfigFields() as field}
                <label class="form-control">
                  <span class="label">
                    <span class="label-text inline-flex items-center gap-1.5">
                      <span>{field.label}{field.required ? ' *' : ''}</span>
                      {#if field.help_text}
                        <span class="tooltip tooltip-right" data-tip={field.help_text}>
                          <button
                            class="inline-flex h-4 w-4 items-center justify-center rounded-full border border-base-content/30 bg-transparent text-[10px] font-semibold leading-none text-base-content/60"
                            aria-label={field.help_text}
                            type="button"
                          >
                            ?
                          </button>
                        </span>
                      {/if}
                    </span>
                  </span>
                  {#if fieldOptions(field).length > 0}
                    <select
                      class="select select-bordered font-mono text-sm"
                      value={structuredConfig[field.key] ?? ''}
                      onchange={(event) => updateConfigField(field, event.currentTarget.value)}
                    >
                      <option value=""></option>
                      {#each fieldOptions(field) as option}
                        <option value={option.value}>{option.label}</option>
                      {/each}
                    </select>
                  {:else if field.kind === 'boolean'}
                    <input
                      class="toggle toggle-primary"
                      type="checkbox"
                      checked={Boolean(structuredConfig[field.key])}
                      onchange={(event) => updateConfigField(field, event.currentTarget.checked)}
                    />
                  {:else}
                    <input
                      class="input input-bordered font-mono text-sm"
                      type={inputTypeForField(field)}
                      value={structuredConfig[field.key] ?? ''}
                      placeholder={field.placeholder}
                      oninput={(event) => updateConfigField(field, event.currentTarget.value)}
                    />
                  {/if}
                </label>
              {/each}
            </div>
          </div>
        {/if}

        <div class="grid gap-3 sm:grid-cols-2">
          <label class="form-control">
            <span class="label"><span class="label-text">Credential type</span></span>
            <select class="select select-bordered font-mono text-sm" bind:value={form.credential_type} disabled={Boolean(currentSchema())}>
              {#each credentialTypeOptions() as option}
                <option value={option.value}>{option.label}</option>
              {/each}
            </select>
          </label>
        </div>

        {#if currentSchema()}
          <div class="rounded border border-base-300 p-3">
            <h3 class="mb-2 text-sm font-semibold">Credential</h3>
            <div class="grid gap-3">
              {#each currentCredentialFields() as field}
                <label class="form-control">
                  <span class="label">
                    <span class="label-text inline-flex items-center gap-1.5">
                      <span>{field.label}{field.required ? ' *' : ''}</span>
                      {#if field.help_text}
                        <span class="tooltip tooltip-right" data-tip={field.help_text}>
                          <button
                            class="inline-flex h-4 w-4 items-center justify-center rounded-full border border-base-content/30 bg-transparent text-[10px] font-semibold leading-none text-base-content/60"
                            aria-label={field.help_text}
                            type="button"
                          >
                            ?
                          </button>
                        </span>
                      {/if}
                    </span>
                  </span>
                  <div class="join w-full">
                    <input
                      class="input input-bordered join-item min-w-0 flex-1 font-mono text-sm"
                      type={credentialInputType(field)}
                      value={structuredCredential[field.key] ?? ''}
                      placeholder={field.placeholder}
                      oninput={(event) => updateCredentialField(field, event.currentTarget.value)}
                    />
                    {#if field.kind === 'secret'}
                      <button
                        class="btn join-item btn-outline w-12 px-0"
                        type="button"
                        title={credentialVisibility[field.key] ? 'Hide value' : 'Show value'}
                        aria-label={credentialVisibility[field.key] ? 'Hide value' : 'Show value'}
                        onclick={() => toggleCredentialVisibility(field)}
                      >
                        {#if credentialVisibility[field.key]}
                          <svg class="h-4 w-4" aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M3 3l18 18" />
                            <path d="M10.73 5.08A10.43 10.43 0 0 1 12 5c7 0 10 7 10 7a13.16 13.16 0 0 1-1.67 2.68" />
                            <path d="M6.61 6.61A13.52 13.52 0 0 0 2 12s3 7 10 7a9.74 9.74 0 0 0 5.39-1.61" />
                            <path d="M9.88 9.88a3 3 0 0 0 4.24 4.24" />
                          </svg>
                        {:else}
                          <svg class="h-4 w-4" aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M2.06 12.35a1 1 0 0 1 0-.7C3.42 7.5 7.36 5 12 5s8.58 2.5 9.94 6.65a1 1 0 0 1 0 .7C20.58 16.5 16.64 19 12 19s-8.58-2.5-9.94-6.65Z" />
                            <circle cx="12" cy="12" r="3" />
                          </svg>
                        {/if}
                      </button>
                    {/if}
                  </div>
                </label>
              {/each}
            </div>
          </div>
        {:else}
          <label class="form-control">
            <span class="label"><span class="label-text">Credential value</span></span>
            <div class="join w-full">
              <input
                class="input input-bordered join-item min-w-0 flex-1 font-mono text-sm"
                type={customCredentialVisible ? 'text' : 'password'}
                value={form.credential_value}
                oninput={(event) => updatePlainCredential(event.currentTarget.value)}
                placeholder="Credential value"
              />
              <button
                class="btn join-item btn-outline w-12 px-0"
                type="button"
                title={customCredentialVisible ? 'Hide value' : 'Show value'}
                aria-label={customCredentialVisible ? 'Hide value' : 'Show value'}
                onclick={() => (customCredentialVisible = !customCredentialVisible)}
              >
                {#if customCredentialVisible}
                  <svg class="h-4 w-4" aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M3 3l18 18" />
                    <path d="M10.73 5.08A10.43 10.43 0 0 1 12 5c7 0 10 7 10 7a13.16 13.16 0 0 1-1.67 2.68" />
                    <path d="M6.61 6.61A13.52 13.52 0 0 0 2 12s3 7 10 7a9.74 9.74 0 0 0 5.39-1.61" />
                    <path d="M9.88 9.88a3 3 0 0 0 4.24 4.24" />
                  </svg>
                {:else}
                  <svg class="h-4 w-4" aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M2.06 12.35a1 1 0 0 1 0-.7C3.42 7.5 7.36 5 12 5s8.58 2.5 9.94 6.65a1 1 0 0 1 0 .7C20.58 16.5 16.64 19 12 19s-8.58-2.5-9.94-6.65Z" />
                    <circle cx="12" cy="12" r="3" />
                  </svg>
                {/if}
              </button>
            </div>
          </label>
        {/if}

        <details class="collapse collapse-arrow rounded border border-base-300">
          <summary class="collapse-title text-sm font-semibold">Advanced JSON</summary>
          <div class="collapse-content grid gap-3">
            <label class="form-control">
              <span class="label"><span class="label-text">Config JSON</span></span>
              <textarea
                class="textarea textarea-bordered min-h-32 font-mono text-sm"
                bind:value={form.config_json}
                onblur={syncStructuredStateFromForm}
              ></textarea>
            </label>

            <label class="form-control">
              <span class="label"><span class="label-text">Metadata JSON</span></span>
              <textarea class="textarea textarea-bordered min-h-24 font-mono text-sm" bind:value={form.metadata_json}></textarea>
            </label>
          </div>
        </details>

        <button class="btn btn-primary" type="button" onclick={saveChannel} disabled={saving}>
          {#if saving}
            <span class="loading loading-spinner loading-sm"></span>
          {/if}
          Save channel
        </button>
      </div>
    </div>

    <div class="min-w-0">
      <div class="tabs tabs-lift">
        {#each scenarios as scenario}
          <input
            type="radio"
            name="parameter_scenario_tabs"
            class="tab"
            aria-label={scenario.label}
            checked={activeScenario === scenario.key}
            onchange={() => selectScenario(scenario.key)}
          />
          <div class="tab-content border-base-300 bg-base-100 p-4">
            <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
              <div>
                <h2 class="text-lg font-semibold">{scenario.label}</h2>
                <p class="text-xs text-base-content/60">{scenarioChannels(scenario.key).length} channels</p>
              </div>
              {#if loadingByScenario[scenario.key] || loadingSchemasByScenario[scenario.key]}
                <span class="loading loading-spinner loading-sm"></span>
              {/if}
            </div>

            {#if scenarioChannels(scenario.key).length === 0}
              <div class="rounded border border-dashed border-base-300 p-6 text-center text-sm text-base-content/60">
                {loadingByScenario[scenario.key] ? 'Loading integration channels...' : 'No integration channels'}
              </div>
            {:else}
              <div class="overflow-x-auto">
                <table class="table table-sm">
                  <thead>
                    <tr>
                      <th>Channel</th>
                      <th>Provider</th>
                      <th>Environment</th>
                      <th>Credential</th>
                      <th>Status</th>
                      <th>Updated</th>
                      <th class="text-right">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#each scenarioChannels(scenario.key) as channel}
                      <tr class:selected={form.id === channel.id}>
                        <td>
                          <div class="font-medium">{channel.channel_code}</div>
                          <div class="max-w-72 truncate font-mono text-xs text-base-content/50">{channel.adapter_key}</div>
                        </td>
                        <td>
                          <div class="font-mono text-xs">{channel.provider_code}</div>
                          <div class="text-xs text-base-content/50">priority {channel.priority}</div>
                        </td>
                        <td class="font-mono text-xs">{channel.environment}</td>
                        <td>
                          <div class="font-mono text-xs">{channel.credential_type}</div>
                          <div class="text-xs text-base-content/60">{channel.credential_value ? 'configured' : '--'}</div>
                        </td>
                        <td>
                          <div class="flex flex-col items-start gap-1">
                            <span class="badge {channel.enabled ? 'badge-success' : 'badge-outline'}">
                              {channel.enabled ? 'enabled' : 'disabled'}
                            </span>
                            <span class="badge {channel.webhook_enabled ? 'badge-info' : 'badge-ghost'}">
                              {channel.webhook_enabled ? 'webhook' : 'no webhook'}
                            </span>
                          </div>
                        </td>
                        <td class="text-xs">{formatDate(channel.updated_at)}</td>
                        <td class="text-right">
                          <div class="join">
                            <button class="btn join-item btn-xs" type="button" onclick={() => editChannel(channel)}>Edit</button>
                            <button class="btn join-item btn-xs" type="button" onclick={() => toggleChannel(channel)}>
                              {channel.enabled ? 'Disable' : 'Enable'}
                            </button>
                          </div>
                        </td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            {/if}
          </div>
        {/each}
      </div>
    </div>
  </div>
</section>

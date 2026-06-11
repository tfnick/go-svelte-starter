<script>
  import { onMount } from 'svelte';

  import {
    createVariable,
    listVariables,
    setVariableEnabled,
    updateVariable
  } from '../api.js';
  import Notice from '../components/Notice.svelte';
  import { formatLocalDateTime } from '../helpers/dateTime.js';

  const valueTypes = [
    { value: 'string', label: 'String' },
    { value: 'number', label: 'Number' },
    { value: 'boolean', label: 'Boolean' },
    { value: 'json', label: 'JSON' }
  ];

  let variables = $state([]);
  let form = $state(emptyForm());
  let loading = $state(false);
  let saving = $state(false);
  let error = $state('');
  let message = $state('');

  onMount(() => {
    loadVariables();
  });

  function emptyForm() {
    return {
      id: '',
      key: '',
      name: '',
      value_type: 'string',
      value_json: '',
      enabled: true,
      description: ''
    };
  }

  async function loadVariables() {
    loading = true;
    error = '';
    try {
      variables = await listVariables();
    } catch (err) {
      error = err.message || 'Failed to load variables';
    } finally {
      loading = false;
    }
  }

  function resetForm() {
    form = emptyForm();
  }

  function editVariable(variable) {
    form = {
      id: variable.id,
      key: variable.key,
      name: variable.name,
      value_type: variable.value_type,
      value_json: valueForEditor(variable.value_type, variable.value_json),
      enabled: Boolean(variable.enabled),
      description: variable.description || ''
    };
  }

  async function saveVariable() {
    saving = true;
    error = '';
    message = '';

    const payload = {
      key: form.key,
      name: form.name,
      value_type: form.value_type,
      value_json: valueForPayload(form.value_type, form.value_json),
      enabled: form.enabled,
      description: form.description
    };

    try {
      const saved = form.id
        ? await updateVariable(form.id, payload)
        : await createVariable(payload);
      message = form.id ? 'Variable updated' : 'Variable created';
      editVariable(saved);
      await loadVariables();
    } catch (err) {
      error = err.message || 'Failed to save variable';
    } finally {
      saving = false;
    }
  }

  async function toggleVariable(variable) {
    error = '';
    message = '';
    try {
      const updated = await setVariableEnabled(variable.id, !variable.enabled);
      message = updated.enabled ? 'Variable enabled' : 'Variable disabled';
      await loadVariables();
      if (form.id === variable.id) {
        editVariable(updated);
      }
    } catch (err) {
      error = err.message || 'Failed to update variable';
    }
  }

  function valueForPayload(valueType, value) {
    const trimmed = String(value || '').trim();
    if (valueType === 'string') return trimmed;
    if (!trimmed) {
      if (valueType === 'boolean') return 'false';
      if (valueType === 'json') return '{}';
    }
    try {
      return JSON.stringify(JSON.parse(trimmed));
    } catch {
      return trimmed;
    }
  }

  function valueForEditor(valueType, value) {
    const trimmed = String(value || '').trim();
    if (!trimmed) return '';
    try {
      const decoded = JSON.parse(trimmed);
      if (valueType === 'string') return String(decoded ?? '');
      return JSON.stringify(decoded, null, 2);
    } catch {
      return trimmed;
    }
  }

  function typeLabel(value) {
    return valueTypes.find((type) => type.value === value)?.label || value;
  }

  function formatValue(variable) {
    if (variable.value_type === 'string') {
      try {
        return JSON.parse(variable.value_json);
      } catch {
        return variable.value_json;
      }
    }
    return variable.value_json;
  }

  function formatDate(value) {
    return formatLocalDateTime(value);
  }
</script>

<section class="space-y-6">
  <div class="flex flex-wrap items-start justify-between gap-4">
    <div>
      <h1 class="text-2xl font-bold leading-tight">Variable</h1>
      <p class="mt-1 text-sm text-base-content/60">Typed global parameters and logic-control values.</p>
    </div>
    <div class="flex gap-2">
      <button class="btn btn-outline btn-sm" type="button" onclick={loadVariables} disabled={loading}>
        {#if loading}
          <span class="loading loading-spinner loading-xs"></span>
        {/if}
        Refresh
      </button>
      <button class="btn btn-primary btn-sm" type="button" onclick={resetForm}>New variable</button>
    </div>
  </div>

  <Notice type="success" message={message} />
  <Notice type="error" message={error} />

  <div class="grid gap-6 xl:grid-cols-[0.58fr_1.02fr]">
    <div class="card min-w-0 border border-base-200 bg-base-100 shadow-sm">
      <div class="card-body gap-4 p-5">
        <div class="flex items-center justify-between gap-3">
          <h2 class="card-title text-lg">{form.id ? 'Edit variable' : 'Create variable'}</h2>
          {#if form.id}
            <span class="badge badge-outline max-w-48 truncate font-mono text-xs">{form.id}</span>
          {/if}
        </div>

        <fieldset class="fieldset">
          <legend class="fieldset-legend">Key</legend>
          <input class="input font-mono text-sm w-full" bind:value={form.key} placeholder="checkout.max_retry" />
        </fieldset>

        <fieldset class="fieldset">
          <legend class="fieldset-legend">Name</legend>
          <input class="input w-full" bind:value={form.name} placeholder="Checkout max retry" />
        </fieldset>

        <fieldset class="fieldset">
          <legend class="fieldset-legend">Value type</legend>
          <select class="select w-full" bind:value={form.value_type}>
            {#each valueTypes as type}
              <option value={type.value}>{type.label}</option>
            {/each}
          </select>
        </fieldset>

        {#if form.value_type === 'boolean'}
          <fieldset class="fieldset">
          <legend class="fieldset-legend">Value</legend>
            <select class="select w-full" bind:value={form.value_json}>
              <option value="true">true</option>
              <option value="false">false</option>
            </select>
        </fieldset>
        {:else if form.value_type === 'json'}
          <fieldset class="fieldset">
          <legend class="fieldset-legend">Value JSON</legend>
            <textarea class="textarea min-h-36 font-mono text-sm w-full" bind:value={form.value_json} placeholder={'{}'}></textarea>
        </fieldset>
        {:else}
          <fieldset class="fieldset">
          <legend class="fieldset-legend">Value</legend>
            <input class="input font-mono text-sm w-full" bind:value={form.value_json} placeholder={form.value_type === 'number' ? '100' : 'active'} />
        </fieldset>
        {/if}

        <fieldset class="fieldset">
          <legend class="fieldset-legend">Description</legend>
          <textarea class="textarea min-h-24 w-full" bind:value={form.description}></textarea>
        </fieldset>

        <label class="fieldset-label cursor-pointer justify-start gap-3 rounded-box border border-base-200 px-3 bg-base-200/40 py-3">
          <input class="toggle toggle-primary" type="checkbox" bind:checked={form.enabled} />
          <span>Enabled</span>
        </label>

        <button class="btn btn-primary" type="button" onclick={saveVariable} disabled={saving}>
          {#if saving}
            <span class="loading loading-spinner loading-sm"></span>
          {/if}
          Save variable
        </button>
      </div>
    </div>

    <div class="card min-w-0 border border-base-200 bg-base-100 shadow-sm">
      <div class="card-body gap-4 p-5">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <h2 class="card-title text-lg">Variables</h2>
          <span class="badge badge-outline">{variables.length}</span>
        </div>

        {#if variables.length === 0}
          <div class="rounded-box border border-dashed border-base-200 p-6 text-center text-sm text-base-content/60">
            {loading ? 'Loading variables...' : 'No variables'}
          </div>
        {:else}
          <div class="max-w-full overflow-x-auto rounded-box border border-base-200">
            <table class="table table-zebra table-sm min-w-[44rem]">
              <thead>
                <tr>
                  <th>Variable</th>
                  <th>Value</th>
                  <th>Status</th>
                  <th>Updated</th>
                  <th class="text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                {#each variables as variable}
                  <tr class={form.id === variable.id ? 'bg-primary/5' : ''}>
                    <td>
                      <div class="font-medium">{variable.name}</div>
                      <div class="max-w-64 truncate font-mono text-xs text-base-content/60">{variable.key}</div>
                    </td>
                    <td>
                      <div class="font-mono text-xs text-base-content/50">{typeLabel(variable.value_type)}</div>
                      <div class="max-w-80 truncate font-mono text-xs">{formatValue(variable)}</div>
                      <div class="max-w-64 truncate text-xs text-base-content/50">{variable.description || '--'}</div>
                    </td>
                    <td>
                      <span class="badge {variable.enabled ? 'badge-success' : 'badge-outline'}">
                        {variable.enabled ? 'enabled' : 'disabled'}
                      </span>
                    </td>
                    <td class="text-xs">{formatDate(variable.updated_at)}</td>
                    <td class="text-right">
                      <div class="join">
                        <button class="btn join-item btn-xs" type="button" onclick={() => editVariable(variable)}>Edit</button>
                        <button class="btn join-item btn-xs" type="button" onclick={() => toggleVariable(variable)}>
                          {variable.enabled ? 'Disable' : 'Enable'}
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
    </div>
  </div>
</section>

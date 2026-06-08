<script>
  import { onMount } from 'svelte';

  import {
    createDictionaryType,
    createDictionaryValue,
    listDictionaryTypes,
    listDictionaryValues,
    setDictionaryTypeEnabled,
    setDictionaryValueEnabled,
    updateDictionaryType,
    updateDictionaryValue
  } from '../api.js';
  import Notice from '../components/Notice.svelte';
  import { formatLocalDateTime } from '../helpers/dateTime.js';

  let dictionaryTypes = $state([]);
  let dictionaryValues = $state([]);
  let selectedTypeId = $state('');
  let typeForm = $state(emptyTypeForm());
  let valueForm = $state(emptyValueForm());
  let loadingTypes = $state(false);
  let loadingValues = $state(false);
  let savingType = $state(false);
  let savingValue = $state(false);
  let error = $state('');
  let message = $state('');

  onMount(() => {
    loadTypes();
  });

  function emptyTypeForm() {
    return {
      id: '',
      type_key: '',
      name: '',
      enabled: true,
      description: ''
    };
  }

  function emptyValueForm(typeId = selectedTypeId) {
    return {
      id: '',
      dictionary_type_id: typeId || '',
      value_code: '',
      label: '',
      sort_order: 100,
      enabled: true,
      description: ''
    };
  }

  function selectedType() {
    return dictionaryTypes.find((type) => type.id === selectedTypeId);
  }

  async function loadTypes(preferredTypeId = selectedTypeId) {
    loadingTypes = true;
    error = '';
    try {
      dictionaryTypes = await listDictionaryTypes();
      const nextType = dictionaryTypes.find((type) => type.id === preferredTypeId) || dictionaryTypes[0];
      selectedTypeId = nextType?.id || '';
      if (nextType) {
        editType(nextType);
        await loadValues(nextType.id);
      } else {
        dictionaryValues = [];
        typeForm = emptyTypeForm();
        valueForm = emptyValueForm('');
      }
    } catch (err) {
      error = err.message || 'Failed to load dictionaries';
    } finally {
      loadingTypes = false;
    }
  }

  async function loadValues(typeId = selectedTypeId) {
    if (!typeId) {
      dictionaryValues = [];
      valueForm = emptyValueForm('');
      return;
    }

    loadingValues = true;
    error = '';
    try {
      dictionaryValues = await listDictionaryValues(typeId);
      if (valueForm.id && !dictionaryValues.some((value) => value.id === valueForm.id)) {
        valueForm = emptyValueForm(typeId);
      } else {
        valueForm = { ...valueForm, dictionary_type_id: typeId };
      }
    } catch (err) {
      error = err.message || 'Failed to load dictionary values';
    } finally {
      loadingValues = false;
    }
  }

  async function selectType(type) {
    if (!type?.id || type.id === selectedTypeId) return;
    selectedTypeId = type.id;
    editType(type);
    valueForm = emptyValueForm(type.id);
    await loadValues(type.id);
  }

  function editType(type) {
    typeForm = {
      id: type.id,
      type_key: type.type_key,
      name: type.name,
      enabled: Boolean(type.enabled),
      description: type.description || ''
    };
  }

  function resetTypeForm() {
    typeForm = emptyTypeForm();
  }

  async function saveType() {
    savingType = true;
    error = '';
    message = '';

    const payload = {
      type_key: typeForm.type_key,
      name: typeForm.name,
      enabled: typeForm.enabled,
      description: typeForm.description
    };

    try {
      const saved = typeForm.id
        ? await updateDictionaryType(typeForm.id, payload)
        : await createDictionaryType(payload);
      message = typeForm.id ? 'Dictionary updated' : 'Dictionary created';
      await loadTypes(saved.id);
    } catch (err) {
      error = err.message || 'Failed to save dictionary';
    } finally {
      savingType = false;
    }
  }

  async function toggleType(type) {
    if (!type?.id) return;
    error = '';
    message = '';
    try {
      const updated = await setDictionaryTypeEnabled(type.id, !type.enabled);
      message = updated.enabled ? 'Dictionary enabled' : 'Dictionary disabled';
      await loadTypes(updated.id);
    } catch (err) {
      error = err.message || 'Failed to update dictionary';
    }
  }

  function editValue(value) {
    valueForm = {
      id: value.id,
      dictionary_type_id: value.dictionary_type_id,
      value_code: value.value_code,
      label: value.label,
      sort_order: Number(value.sort_order || 0),
      enabled: Boolean(value.enabled),
      description: value.description || ''
    };
  }

  function resetValueForm() {
    valueForm = emptyValueForm(selectedTypeId);
  }

  async function saveValue() {
    if (!selectedTypeId) {
      error = 'Select a dictionary first';
      return;
    }

    savingValue = true;
    error = '';
    message = '';

    const payload = {
      dictionary_type_id: selectedTypeId,
      value_code: valueForm.value_code,
      label: valueForm.label,
      sort_order: Number(valueForm.sort_order || 0),
      enabled: valueForm.enabled,
      description: valueForm.description
    };

    try {
      const saved = valueForm.id
        ? await updateDictionaryValue(selectedTypeId, valueForm.id, payload)
        : await createDictionaryValue(selectedTypeId, payload);
      message = valueForm.id ? 'Dictionary value updated' : 'Dictionary value created';
      editValue(saved);
      await loadValues(selectedTypeId);
    } catch (err) {
      error = err.message || 'Failed to save dictionary value';
    } finally {
      savingValue = false;
    }
  }

  async function toggleValue(value) {
    error = '';
    message = '';
    try {
      const updated = await setDictionaryValueEnabled(value.id, !value.enabled);
      message = updated.enabled ? 'Dictionary value enabled' : 'Dictionary value disabled';
      await loadValues(selectedTypeId);
      if (valueForm.id === value.id) {
        editValue(updated);
      }
    } catch (err) {
      error = err.message || 'Failed to update dictionary value';
    }
  }

  function formatDate(value) {
    return formatLocalDateTime(value);
  }
</script>

<section class="space-y-6">
  <div class="flex flex-wrap items-start justify-between gap-4">
    <div>
      <h1 class="text-2xl font-bold leading-tight">Dictionary</h1>
      <p class="mt-1 text-sm text-base-content/60">Manage dictionary types and selectable values.</p>
    </div>
    <div class="flex gap-2">
      <button class="btn btn-outline btn-sm" type="button" onclick={() => loadTypes()} disabled={loadingTypes}>
        {#if loadingTypes}
          <span class="loading loading-spinner loading-xs"></span>
        {/if}
        Refresh
      </button>
      <button class="btn btn-primary btn-sm" type="button" onclick={resetTypeForm}>New dictionary</button>
    </div>
  </div>

  <Notice type="success" message={message} />
  <Notice type="error" message={error} />

  <div class="grid gap-6 xl:grid-cols-[0.52fr_1.08fr]">
    <div class="space-y-6">
      <div class="card border border-base-300 bg-base-100 shadow-sm">
        <div class="card-body gap-4">
          <div class="flex items-center justify-between gap-3">
            <h2 class="card-title text-lg">{typeForm.id ? 'Edit dictionary' : 'Create dictionary'}</h2>
            {#if typeForm.id}
              <span class="badge badge-outline max-w-44 truncate font-mono text-xs">{typeForm.id}</span>
            {/if}
          </div>

          <label class="form-control">
            <span class="label"><span class="label-text">Type key</span></span>
            <input class="input input-bordered font-mono text-sm" bind:value={typeForm.type_key} placeholder="order_status" />
          </label>

          <label class="form-control">
            <span class="label"><span class="label-text">Name</span></span>
            <input class="input input-bordered" bind:value={typeForm.name} placeholder="Order status" />
          </label>

          <label class="form-control">
            <span class="label"><span class="label-text">Description</span></span>
            <textarea class="textarea textarea-bordered min-h-20" bind:value={typeForm.description}></textarea>
          </label>

          <label class="label cursor-pointer justify-start gap-3 rounded border border-base-300 px-3">
            <input class="toggle toggle-primary" type="checkbox" bind:checked={typeForm.enabled} />
            <span class="label-text">Enabled</span>
          </label>

          <button class="btn btn-primary" type="button" onclick={saveType} disabled={savingType}>
            {#if savingType}
              <span class="loading loading-spinner loading-sm"></span>
            {/if}
            Save dictionary
          </button>
        </div>
      </div>

      <div class="card border border-base-300 bg-base-100 shadow-sm">
        <div class="card-body gap-4">
          <div class="flex items-center justify-between gap-3">
            <h2 class="card-title text-lg">Dictionaries</h2>
            <span class="badge badge-outline">{dictionaryTypes.length}</span>
          </div>

          {#if dictionaryTypes.length === 0}
            <div class="rounded border border-dashed border-base-300 p-6 text-center text-sm text-base-content/60">
              {loadingTypes ? 'Loading dictionaries...' : 'No dictionaries'}
            </div>
          {:else}
            <div class="space-y-2">
              {#each dictionaryTypes as type}
                <button
                  class="w-full rounded border p-3 text-left transition hover:border-primary {selectedTypeId === type.id ? 'border-primary bg-primary/5' : 'border-base-300'}"
                  type="button"
                  onclick={() => selectType(type)}
                >
                  <div class="flex items-center justify-between gap-3">
                    <div class="min-w-0">
                      <div class="truncate font-medium">{type.name}</div>
                      <div class="truncate font-mono text-xs text-base-content/60">{type.type_key}</div>
                    </div>
                    <span class="badge {type.enabled ? 'badge-success' : 'badge-outline'}">
                      {type.enabled ? 'enabled' : 'disabled'}
                    </span>
                  </div>
                  <div class="mt-1 truncate text-xs text-base-content/50">{type.description || '--'}</div>
                </button>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    </div>

    <div class="space-y-6">
      <div class="card border border-base-300 bg-base-100 shadow-sm">
        <div class="card-body gap-4">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 class="card-title text-lg">{valueForm.id ? 'Edit value' : 'Create value'}</h2>
              <p class="mt-1 text-sm text-base-content/60">
                {selectedType()?.name || 'Select a dictionary to manage values'}
              </p>
            </div>
            <button class="btn btn-outline btn-sm" type="button" onclick={resetValueForm} disabled={!selectedTypeId}>New value</button>
          </div>

          <div class="grid gap-3 sm:grid-cols-2">
            <label class="form-control">
              <span class="label"><span class="label-text">Value code</span></span>
              <input class="input input-bordered font-mono text-sm" bind:value={valueForm.value_code} placeholder="pending" disabled={!selectedTypeId} />
            </label>

            <label class="form-control">
              <span class="label"><span class="label-text">Label</span></span>
              <input class="input input-bordered" bind:value={valueForm.label} placeholder="Pending" disabled={!selectedTypeId} />
            </label>
          </div>

          <div class="grid gap-3 sm:grid-cols-2">
            <label class="form-control">
              <span class="label"><span class="label-text">Sort order</span></span>
              <input class="input input-bordered" type="number" bind:value={valueForm.sort_order} disabled={!selectedTypeId} />
            </label>

            <label class="label mt-9 cursor-pointer justify-start gap-3 rounded border border-base-300 px-3">
              <input class="toggle toggle-primary" type="checkbox" bind:checked={valueForm.enabled} disabled={!selectedTypeId} />
              <span class="label-text">Enabled</span>
            </label>
          </div>

          <label class="form-control">
            <span class="label"><span class="label-text">Description</span></span>
            <textarea class="textarea textarea-bordered min-h-20" bind:value={valueForm.description} disabled={!selectedTypeId}></textarea>
          </label>

          <button class="btn btn-primary" type="button" onclick={saveValue} disabled={savingValue || !selectedTypeId}>
            {#if savingValue}
              <span class="loading loading-spinner loading-sm"></span>
            {/if}
            Save value
          </button>
        </div>
      </div>

      <div class="card border border-base-300 bg-base-100 shadow-sm">
        <div class="card-body gap-4">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <h2 class="card-title text-lg">Values</h2>
            <div class="flex gap-2">
              {#if selectedTypeId}
                <button class="btn btn-outline btn-xs" type="button" onclick={() => toggleType(selectedType())}>
                  {selectedType()?.enabled ? 'Disable dictionary' : 'Enable dictionary'}
                </button>
              {/if}
              <span class="badge badge-outline">{dictionaryValues.length}</span>
            </div>
          </div>

          {#if !selectedTypeId}
            <div class="rounded border border-dashed border-base-300 p-6 text-center text-sm text-base-content/60">
              Select or create a dictionary
            </div>
          {:else if dictionaryValues.length === 0}
            <div class="rounded border border-dashed border-base-300 p-6 text-center text-sm text-base-content/60">
              {loadingValues ? 'Loading values...' : 'No values'}
            </div>
          {:else}
            <div class="overflow-x-auto">
              <table class="table table-sm">
                <thead>
                  <tr>
                    <th>Value</th>
                    <th>Order</th>
                    <th>Status</th>
                    <th>Updated</th>
                    <th class="text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {#each dictionaryValues as value}
                    <tr class:selected={valueForm.id === value.id}>
                      <td>
                        <div class="font-medium">{value.label}</div>
                        <div class="max-w-64 truncate font-mono text-xs text-base-content/60">{value.value_code}</div>
                        <div class="max-w-72 truncate text-xs text-base-content/50">{value.description || '--'}</div>
                      </td>
                      <td class="font-mono text-xs">{value.sort_order}</td>
                      <td>
                        <span class="badge {value.enabled ? 'badge-success' : 'badge-outline'}">
                          {value.enabled ? 'enabled' : 'disabled'}
                        </span>
                      </td>
                      <td class="text-xs">{formatDate(value.updated_at)}</td>
                      <td class="text-right">
                        <div class="join">
                          <button class="btn join-item btn-xs" type="button" onclick={() => editValue(value)}>Edit</button>
                          <button class="btn join-item btn-xs" type="button" onclick={() => toggleValue(value)}>
                            {value.enabled ? 'Disable' : 'Enable'}
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
  </div>
</section>

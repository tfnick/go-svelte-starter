<script>
  import { uploadSiteLogo } from '../api.js';
  import Notice from '../components/Notice.svelte';
  import { formatLocalDateTime } from '../helpers/dateTime.js';

  let { settings, onSettingsChanged } = $props();
  let activeTab = $state('general');
  let selectedLogo = $state(null);
  let saving = $state(false);
  let error = $state('');
  let message = $state('');
  let fileInput;

  function logoURL() {
    return settings?.logo_url || '/logo.png';
  }

  function logoStatus() {
    return settings?.logo_configured ? 'configured' : 'default';
  }

  function updatedAt() {
    return settings?.logo_updated_at ? formatLocalDateTime(settings.logo_updated_at) : '--';
  }

  function selectLogo(event) {
    selectedLogo = event.currentTarget.files?.[0] || null;
    error = '';
    message = '';
  }

  async function saveLogo() {
    if (!selectedLogo) {
      error = 'Logo file is required';
      return;
    }

    saving = true;
    error = '';
    message = '';
    try {
      await uploadSiteLogo(selectedLogo);
      selectedLogo = null;
      if (fileInput) {
        fileInput.value = '';
      }
      message = 'Logo updated';
      await onSettingsChanged?.();
    } catch (err) {
      error = err.message || 'Failed to update logo';
    } finally {
      saving = false;
    }
  }
</script>

<section class="space-y-6">
  <div class="flex flex-wrap items-start justify-between gap-4">
    <div>
      <h1 class="text-2xl font-bold leading-tight">Setting</h1>
      <p class="mt-1 text-sm text-base-content/60">Site preferences and retention controls.</p>
    </div>
    <button class="btn btn-outline btn-sm" type="button" onclick={() => onSettingsChanged?.()}>
      Refresh
    </button>
  </div>

  <Notice type="success" message={message} />
  <Notice type="error" message={error} />

  <div class="tabs tabs-lift">
    <input
      type="radio"
      name="setting_tabs"
      class="tab"
      aria-label="General"
      checked={activeTab === 'general'}
      onchange={() => (activeTab = 'general')}
    />
    <div class="tab-content border-base-300 bg-base-100 p-4">
      <div class="grid gap-6 xl:grid-cols-[0.48fr_1fr]">
        <div class="card border border-base-300 bg-base-100 shadow-sm">
          <div class="card-body gap-4">
            <div class="flex items-center justify-between gap-3">
              <h2 class="card-title text-lg">Logo</h2>
              <span class="badge badge-outline">{logoStatus()}</span>
            </div>

            <div class="rounded border border-base-300 bg-base-200/50 p-4">
              <img
                alt="Svelte Go Starter"
                class="h-[25px] w-[110px] object-contain"
                height="25"
                src={logoURL()}
                width="110"
              />
            </div>

            <div class="grid gap-1 text-sm">
              <div class="flex items-center justify-between gap-3">
                <span class="text-base-content/60">Size</span>
                <span class="font-mono">110 x 25</span>
              </div>
              <div class="flex items-center justify-between gap-3">
                <span class="text-base-content/60">Updated</span>
                <span class="text-right text-xs">{updatedAt()}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="card border border-base-300 bg-base-100 shadow-sm">
          <div class="card-body gap-4">
            <div class="flex items-center justify-between gap-3">
              <h2 class="card-title text-lg">General</h2>
              {#if selectedLogo}
                <span class="badge badge-ghost max-w-60 truncate">{selectedLogo.name}</span>
              {/if}
            </div>

            <label class="form-control">
              <span class="label"><span class="label-text">Logo image</span></span>
              <input
                accept="image/png,image/jpeg,image/webp"
                bind:this={fileInput}
                class="file-input file-input-bordered w-full"
                onchange={selectLogo}
                type="file"
              />
            </label>

            <div class="flex flex-wrap items-center justify-end gap-2">
              <button
                class="btn btn-primary"
                disabled={saving || !selectedLogo}
                onclick={saveLogo}
                type="button"
              >
                {#if saving}
                  <span class="loading loading-spinner loading-sm"></span>
                {/if}
                Upload logo
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <input
      type="radio"
      name="setting_tabs"
      class="tab"
      aria-label="Retain"
      checked={activeTab === 'retain'}
      onchange={() => (activeTab = 'retain')}
    />
    <div class="tab-content border-base-300 bg-base-100 p-4">
      <div class="rounded border border-dashed border-base-300 p-6 text-center text-sm text-base-content/60">
        No retain settings
      </div>
    </div>
  </div>
</section>

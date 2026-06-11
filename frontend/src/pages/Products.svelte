<script>
  import { onMount } from 'svelte';

  import { createProduct, getProducts, updateProduct } from '../api.js';
  import Notice from '../components/Notice.svelte';
  import { formatLocalDateTime } from '../helpers/dateTime.js';

  const membershipLevels = [
    { value: 'basic', label: 'Basic' },
    { value: 'premium', label: 'Premium' },
    { value: 'super', label: 'Super' }
  ];

  const billingTypes = [
    { value: 'subscription', label: 'Subscription' },
    { value: 'one_time', label: 'One-time service' }
  ];

  const subscriptionIntervals = [
    { value: 'month', label: 'Monthly' },
    { value: 'three_months', label: '3 months' },
    { value: 'six_months', label: '6 months' },
    { value: 'year', label: 'Yearly' }
  ];

  let products = $state([]);
  let form = $state(emptyForm());
  let loading = $state(false);
  let saving = $state(false);
  let error = $state('');
  let message = $state('');

  onMount(() => {
    loadProducts();
  });

  function emptyForm() {
    return {
      id: '',
      name: '',
      description: '',
      price: '',
      currency: 'USD',
      enabled: true,
      creem_product_id: '',
      billing_type: 'subscription',
      membership_level: 'premium',
      subscription_interval: 'month'
    };
  }

  async function loadProducts() {
    loading = true;
    error = '';
    try {
      products = await getProducts();
    } catch (err) {
      error = err.message || 'Failed to load products';
    } finally {
      loading = false;
    }
  }

  function resetForm() {
    form = emptyForm();
  }

  function editProduct(product) {
    form = {
      id: product.id,
      name: product.name || '',
      description: product.description || '',
      price: product.price ? String(product.price) : '',
      currency: product.currency || 'USD',
      enabled: Boolean(product.enabled),
      creem_product_id: product.creem_product_id || '',
      billing_type: product.billing_type || 'subscription',
      membership_level: product.membership_level || 'premium',
      subscription_interval: product.subscription_interval || 'month'
    };
  }

  async function saveProduct() {
    saving = true;
    error = '';
    message = '';

    const payload = {
      name: form.name,
      description: form.description,
      price: Math.max(0, Number.parseInt(form.price || '0', 10) || 0),
      currency: form.currency,
      enabled: form.enabled,
      creem_product_id: form.creem_product_id,
      billing_type: form.billing_type,
      membership_level: form.membership_level,
      subscription_interval: form.billing_type === 'subscription' ? form.subscription_interval : ''
    };

    try {
      const saved = form.id
        ? await updateProduct(form.id, payload)
        : await createProduct(payload);
      message = form.id ? 'Product updated' : 'Product created';
      editProduct(saved);
      await loadProducts();
    } catch (err) {
      error = err.message || 'Failed to save product';
    } finally {
      saving = false;
    }
  }

  function priceLabel(product) {
    const price = Number(product.price || 0);
    if (price <= 0) return 'Creem';
    const currency = product.currency || 'USD';
    return `${currency} ${(price / 100).toFixed(2)}`;
  }

  function billingLabel(product) {
    const billing = billingTypes.find((type) => type.value === product.billing_type)?.label || product.billing_type;
    if (product.billing_type !== 'subscription') return billing;
    const interval = subscriptionIntervals.find((item) => item.value === product.subscription_interval)?.label || product.subscription_interval;
    return `${billing} / ${interval}`;
  }

  function membershipLabel(value) {
    return membershipLevels.find((level) => level.value === value)?.label || value;
  }

  function statusClass(product) {
    return product.enabled ? 'badge-success' : 'badge-ghost';
  }

  function formatDate(value) {
    return formatLocalDateTime(value);
  }
</script>

<section class="space-y-6">
  <div class="flex flex-wrap items-start justify-between gap-4">
    <div>
      <h1 class="text-2xl font-bold leading-tight">Product</h1>
      <p class="mt-1 text-sm text-base-content/60">Local checkout catalog mapped to Creem products.</p>
    </div>
    <div class="flex gap-2">
      <button class="btn btn-outline btn-sm" type="button" onclick={loadProducts} disabled={loading}>
        {#if loading}
          <span class="loading loading-spinner loading-xs"></span>
        {/if}
        Refresh
      </button>
      <button class="btn btn-primary btn-sm" type="button" onclick={resetForm}>New product</button>
    </div>
  </div>

  <Notice type="success" message={message} />
  <Notice type="error" message={error} />

  <div class="grid gap-6 xl:grid-cols-[minmax(20rem,0.58fr)_minmax(0,1.08fr)]">
    <div class="card min-w-0 border border-base-200 bg-base-100 shadow-sm">
      <div class="card-body gap-5 p-5">
        <div class="flex items-center justify-between gap-3">
          <h2 class="card-title text-lg">{form.id ? 'Edit product' : 'Create product'}</h2>
          {#if form.id}
            <span class="badge badge-outline max-w-48 truncate font-mono text-xs">{form.id}</span>
          {/if}
        </div>

        <fieldset class="fieldset">
          <legend class="fieldset-legend">Name</legend>
          <input class="input w-full" bind:value={form.name} placeholder="Premium monthly" />
        </fieldset>

        <fieldset class="fieldset">
          <legend class="fieldset-legend">Creem product ID</legend>
          <input class="input w-full font-mono text-sm" bind:value={form.creem_product_id} placeholder="prod_..." />
        </fieldset>

        <div class="grid gap-3 sm:grid-cols-2">
          <fieldset class="fieldset">
            <legend class="fieldset-legend">Billing</legend>
            <select class="select w-full" bind:value={form.billing_type}>
              {#each billingTypes as type}
                <option value={type.value}>{type.label}</option>
              {/each}
            </select>
          </fieldset>

          <fieldset class="fieldset">
            <legend class="fieldset-legend">Membership</legend>
            <select class="select w-full" bind:value={form.membership_level}>
              {#each membershipLevels as level}
                <option value={level.value}>{level.label}</option>
              {/each}
            </select>
          </fieldset>
        </div>

        {#if form.billing_type === 'subscription'}
          <fieldset class="fieldset">
            <legend class="fieldset-legend">Interval</legend>
            <select class="select w-full" bind:value={form.subscription_interval}>
              {#each subscriptionIntervals as interval}
                <option value={interval.value}>{interval.label}</option>
              {/each}
            </select>
          </fieldset>
        {/if}

        <div class="grid gap-3 sm:grid-cols-[1fr_0.6fr]">
          <fieldset class="fieldset">
            <legend class="fieldset-legend">Display price cents</legend>
            <input class="input w-full" inputmode="numeric" bind:value={form.price} placeholder="9900" />
          </fieldset>

          <fieldset class="fieldset">
            <legend class="fieldset-legend">Currency</legend>
            <input class="input w-full uppercase" maxlength="3" bind:value={form.currency} placeholder="USD" />
          </fieldset>
        </div>

        <fieldset class="fieldset">
          <legend class="fieldset-legend">Description</legend>
          <textarea class="textarea min-h-24 w-full" bind:value={form.description}></textarea>
        </fieldset>

        <label class="fieldset-label cursor-pointer justify-start gap-3 rounded-box border border-base-200 bg-base-200/40 px-3 py-3">
          <input class="toggle toggle-primary" type="checkbox" bind:checked={form.enabled} />
          <span>Enabled</span>
        </label>

        <button class="btn btn-primary" type="button" onclick={saveProduct} disabled={saving}>
          {#if saving}
            <span class="loading loading-spinner loading-sm"></span>
          {/if}
          Save product
        </button>
      </div>
    </div>

    <div class="card min-w-0 border border-base-200 bg-base-100 shadow-sm">
      <div class="card-body min-w-0 gap-4 p-5">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <h2 class="card-title text-lg">Products</h2>
          <span class="badge badge-outline">{products.length}</span>
        </div>

        {#if products.length === 0}
          <div class="rounded-box border border-dashed border-base-200 p-6 text-center text-sm text-base-content/60">
            {loading ? 'Loading products...' : 'No products'}
          </div>
        {:else}
          <div class="grid gap-3 lg:hidden">
            {#each products as product}
              <div class="rounded-box border border-base-200 bg-base-100 p-4 {form.id === product.id ? 'ring-1 ring-primary/30' : ''}">
                <div class="flex items-start justify-between gap-3">
                  <div class="min-w-0">
                    <div class="truncate font-medium">{product.name}</div>
                    <div class="mt-1 text-xs text-base-content/60">{priceLabel(product)} - {billingLabel(product)}</div>
                  </div>
                  <span class="badge {statusClass(product)} shrink-0">{product.enabled ? 'enabled' : 'disabled'}</span>
                </div>

                <div class="mt-3 grid gap-2 text-xs text-base-content/60">
                  <div class="min-w-0">
                    <div class="font-medium text-base-content/70">Creem</div>
                    <div class="truncate font-mono">{product.creem_product_id || '--'}</div>
                  </div>
                  <div class="flex flex-wrap items-center gap-2">
                    <span class="badge badge-outline badge-sm">{membershipLabel(product.membership_level)}</span>
                    <span class="whitespace-nowrap">Updated {formatDate(product.updated_at)}</span>
                  </div>
                </div>

                <div class="mt-4 flex justify-end">
                  <button class="btn btn-outline btn-sm" type="button" onclick={() => editProduct(product)}>Edit</button>
                </div>
              </div>
            {/each}
          </div>

          <div class="max-w-full hidden overflow-x-auto rounded-box border border-base-200 lg:block">
            <table class="table table-zebra table-sm min-w-[46rem]">
              <thead>
                <tr>
                  <th>Product</th>
                  <th>Creem</th>
                  <th>Billing</th>
                  <th>Grant</th>
                  <th>Status</th>
                  <th class="text-right">Action</th>
                </tr>
              </thead>
              <tbody>
                {#each products as product}
                  <tr class={form.id === product.id ? 'bg-primary/5' : ''}>
                    <td class="min-w-52">
                      <div class="font-medium leading-5">{product.name}</div>
                      <div class="max-w-64 truncate text-xs text-base-content/50">{product.description || '--'}</div>
                      <div class="text-xs text-base-content/50">{priceLabel(product)}</div>
                    </td>
                    <td class="max-w-56 truncate font-mono text-xs text-base-content/70">{product.creem_product_id || '--'}</td>
                    <td class="whitespace-nowrap text-xs">{billingLabel(product)}</td>
                    <td>
                      <span class="badge badge-outline badge-sm">{membershipLabel(product.membership_level)}</span>
                    </td>
                    <td>
                      <div class="badge badge-sm {statusClass(product)}">
                        {product.enabled ? 'enabled' : 'disabled'}
                      </div>
                      <div class="mt-1 whitespace-nowrap text-xs text-base-content/50">{formatDate(product.updated_at)}</div>
                    </td>
                    <td class="text-right">
                      <button class="btn btn-outline btn-xs" type="button" onclick={() => editProduct(product)}>Edit</button>
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

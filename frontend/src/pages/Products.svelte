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

  <div class="grid gap-6 xl:grid-cols-[0.58fr_1.08fr]">
    <div class="card border border-base-300 bg-base-100 shadow-sm">
      <div class="card-body gap-4">
        <div class="flex items-center justify-between gap-3">
          <h2 class="card-title text-lg">{form.id ? 'Edit product' : 'Create product'}</h2>
          {#if form.id}
            <span class="badge badge-outline max-w-48 truncate font-mono text-xs">{form.id}</span>
          {/if}
        </div>

        <label class="form-control">
          <span class="label"><span class="label-text">Name</span></span>
          <input class="input input-bordered" bind:value={form.name} placeholder="Premium monthly" />
        </label>

        <label class="form-control">
          <span class="label"><span class="label-text">Creem product ID</span></span>
          <input class="input input-bordered font-mono text-sm" bind:value={form.creem_product_id} placeholder="prod_..." />
        </label>

        <div class="grid gap-3 sm:grid-cols-2">
          <label class="form-control">
            <span class="label"><span class="label-text">Billing</span></span>
            <select class="select select-bordered" bind:value={form.billing_type}>
              {#each billingTypes as type}
                <option value={type.value}>{type.label}</option>
              {/each}
            </select>
          </label>

          <label class="form-control">
            <span class="label"><span class="label-text">Membership</span></span>
            <select class="select select-bordered" bind:value={form.membership_level}>
              {#each membershipLevels as level}
                <option value={level.value}>{level.label}</option>
              {/each}
            </select>
          </label>
        </div>

        {#if form.billing_type === 'subscription'}
          <label class="form-control">
            <span class="label"><span class="label-text">Interval</span></span>
            <select class="select select-bordered" bind:value={form.subscription_interval}>
              {#each subscriptionIntervals as interval}
                <option value={interval.value}>{interval.label}</option>
              {/each}
            </select>
          </label>
        {/if}

        <div class="grid gap-3 sm:grid-cols-[1fr_0.6fr]">
          <label class="form-control">
            <span class="label"><span class="label-text">Display price cents</span></span>
            <input class="input input-bordered" inputmode="numeric" bind:value={form.price} placeholder="9900" />
          </label>

          <label class="form-control">
            <span class="label"><span class="label-text">Currency</span></span>
            <input class="input input-bordered uppercase" maxlength="3" bind:value={form.currency} placeholder="USD" />
          </label>
        </div>

        <label class="form-control">
          <span class="label"><span class="label-text">Description</span></span>
          <textarea class="textarea textarea-bordered min-h-24" bind:value={form.description}></textarea>
        </label>

        <label class="label cursor-pointer justify-start gap-3 rounded border border-base-300 px-3">
          <input class="toggle toggle-primary" type="checkbox" bind:checked={form.enabled} />
          <span class="label-text">Enabled</span>
        </label>

        <button class="btn btn-primary" type="button" onclick={saveProduct} disabled={saving}>
          {#if saving}
            <span class="loading loading-spinner loading-sm"></span>
          {/if}
          Save product
        </button>
      </div>
    </div>

    <div class="card border border-base-300 bg-base-100 shadow-sm">
      <div class="card-body gap-4">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <h2 class="card-title text-lg">Products</h2>
          <span class="badge badge-outline">{products.length}</span>
        </div>

        {#if products.length === 0}
          <div class="rounded border border-dashed border-base-300 p-6 text-center text-sm text-base-content/60">
            {loading ? 'Loading products...' : 'No products'}
          </div>
        {:else}
          <div class="overflow-x-auto">
            <table class="table table-sm">
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
                  <tr class:selected={form.id === product.id}>
                    <td>
                      <div class="font-medium">{product.name}</div>
                      <div class="max-w-64 truncate text-xs text-base-content/50">{product.description || '--'}</div>
                      <div class="text-xs text-base-content/50">{priceLabel(product)}</div>
                    </td>
                    <td class="max-w-52 truncate font-mono text-xs">{product.creem_product_id || '--'}</td>
                    <td class="text-xs">{billingLabel(product)}</td>
                    <td>
                      <span class="badge badge-outline">{membershipLabel(product.membership_level)}</span>
                    </td>
                    <td>
                      <div class="badge {product.enabled ? 'badge-success' : 'badge-outline'}">
                        {product.enabled ? 'enabled' : 'disabled'}
                      </div>
                      <div class="mt-1 whitespace-nowrap text-xs text-base-content/50">{formatDate(product.updated_at)}</div>
                    </td>
                    <td class="text-right">
                      <button class="btn btn-xs" type="button" onclick={() => editProduct(product)}>Edit</button>
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

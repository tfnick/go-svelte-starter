<script>
  import { createOrder, createOrderPaymentCheckout, getProducts } from '../api.js';
  import Notice from '../components/Notice.svelte';

  let { auth } = $props();

  let product = $state(null);
  let loading = $state(true);
  let busy = $state(false);
  let error = $state('');
  let message = $state('');
  let startedForProductId = $state('');

  $effect(() => {
    if (!auth.logged_in) {
      return;
    }
    const productId = selectedProductId();
    if (!productId) {
      loading = false;
      error = 'Missing product_id in checkout URL';
      return;
    }
    if (startedForProductId !== productId) {
      startedForProductId = productId;
      loadAndStartCheckout(productId);
    }
  });

  function selectedProductId() {
    return new URLSearchParams(globalThis.location?.search || '').get('product_id') || '';
  }

  async function loadAndStartCheckout(productId) {
    loading = true;
    busy = false;
    error = '';
    message = '';
    product = null;

    try {
      const result = await getProducts();
      const products = Array.isArray(result) ? result : [];
      const selected = products.find((item) => item.id === productId);
      if (!selected) {
        throw new Error('Selected product is no longer available');
      }
      if (!selected.enabled || !selected.creem_product_id) {
        throw new Error('Selected product is not enabled for checkout');
      }

      product = selected;
      await startCheckout(productId);
    } catch (err) {
      error = err.message || 'Failed to prepare checkout';
    } finally {
      loading = false;
    }
  }

  async function startCheckout(productId = selectedProductId()) {
    const userId = auth.user?.id || '';
    if (!userId) {
      error = 'Please sign in before checkout';
      return;
    }
    if (!productId) {
      error = 'Missing product_id in checkout URL';
      return;
    }

    busy = true;
    error = '';
    message = 'Creating your order...';
    try {
      const created = await createOrder({ user_id: userId, product_id: productId });
      const order = created?.order;
      if (!order?.id) {
        throw new Error('Order was created without an id');
      }

      message = 'Opening secure checkout...';
      const checkout = await createOrderPaymentCheckout(order.id);
      if (checkout.checkout_url) {
        globalThis.location.assign(checkout.checkout_url);
        return;
      }
      message = `Order ${checkout.order?.id || order.id} is waiting for payment confirmation`;
    } catch (err) {
      error = err.message || 'Failed to start checkout';
      message = '';
    } finally {
      busy = false;
    }
  }

  function priceLabel(value) {
    const price = Number(value?.price || 0);
    if (price <= 0) {
      return 'Contact us';
    }
    const currency = value?.currency || 'USD';
    return `${currency} ${(price / 100).toFixed(2)}`;
  }
</script>

<section class="space-y-6">
  <div class="rounded-lg border border-base-200 bg-base-100 p-6 shadow-sm">
    <div class="max-w-3xl">
      <p class="text-sm font-semibold uppercase text-base-content/50">Checkout</p>
      <h1 class="mt-2 text-3xl font-bold leading-tight">Preparing your secure checkout</h1>
      <p class="mt-3 text-base text-base-content/70">
        Your selected plan is connected to the same order and Creem payment flow used inside the dashboard.
      </p>
    </div>
  </div>

  <Notice type="success" message={message} />
  <Notice type="error" message={error} />

  <div class="grid gap-6 lg:grid-cols-[0.72fr_0.38fr]">
    <div class="rounded-lg border border-base-200 bg-base-100 p-6 shadow-sm">
      {#if loading}
        <div class="flex min-h-48 items-center justify-center">
          <span class="loading loading-spinner loading-md" aria-label="Loading"></span>
        </div>
      {:else if product}
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h2 class="text-2xl font-bold">{product.name}</h2>
            <p class="mt-2 max-w-2xl text-sm leading-6 text-base-content/70">{product.description}</p>
          </div>
          <div class="rounded-md border border-base-200 bg-base-200 px-4 py-3 text-right">
            <div class="text-xs font-semibold uppercase text-base-content/50">Plan</div>
            <div class="text-xl font-bold">{priceLabel(product)}</div>
          </div>
        </div>

        <div class="mt-6 flex flex-wrap gap-3">
          <button class="btn btn-primary" type="button" onclick={() => startCheckout()} disabled={busy}>
            {#if busy}
              <span class="loading loading-spinner loading-sm"></span>
            {/if}
            Continue checkout
          </button>
          <a class="btn btn-outline" href="/pricing">Back to pricing</a>
        </div>
      {:else}
        <div class="space-y-4">
          <p class="text-sm text-base-content/70">Choose a public plan before starting checkout.</p>
          <a class="btn btn-primary" href="/pricing">View pricing</a>
        </div>
      {/if}
    </div>

    <aside class="rounded-lg border border-base-200 bg-base-100 p-5 shadow-sm">
      <h2 class="text-lg font-bold">What happens next</h2>
      <ol class="mt-4 space-y-3 text-sm text-base-content/70">
        <li>1. A pending order ledger is created for your account.</li>
        <li>2. The payment channel creates a hosted checkout session.</li>
        <li>3. You return after the provider confirms payment.</li>
      </ol>
    </aside>
  </div>
</section>

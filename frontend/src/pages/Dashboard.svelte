<script>
  import { onDestroy } from 'svelte';

  import {
    createOrder,
    createOrderPaymentCheckout,
    getMyOrders,
    getMyPoints,
    getProducts,
    pointsSSEURL
  } from '../api.js';
  import Notice from '../components/Notice.svelte';
  import { orderStatusLabel } from '../enums/orderStatus.ts';
  import { formatLocalDateTime } from '../helpers/dateTime.js';
  import { dispatchRealtimeMessage } from '../helpers/realtimeMessages.js';

  let { auth } = $props();

  const orderPageSize = 10;
  const emptyOrderPagination = {
    page: 1,
    page_size: orderPageSize,
    total_items: 0,
    total_pages: 0,
    has_previous: false,
    has_next: false
  };

  let orders = $state([]);
  let products = $state([]);
  let selectedProductId = $state('');
  let orderPagination = $state({ ...emptyOrderPagination });
  let points = $state(null);
  let loadingOrders = $state(false);
  let creatingOrder = $state(false);
  let payingOrderId = $state('');
  let error = $state('');
  let message = $state('');
  let realtimeToasts = $state([]);
  let streamStatus = $state('Disconnected');
  let loadedUserId = $state('');
  let pointsStream;

  onDestroy(() => {
    closePointsStream();
  });

  $effect(() => {
    const userId = auth.logged_in ? auth.user?.id || '' : '';
    if (!userId) {
      resetOrderManagement();
      return;
    }

    if (userId !== loadedUserId) {
      loadedUserId = userId;
      orderPagination = { ...emptyOrderPagination };
      loadOrderManagement();
      connectPointsStream();
    }
  });

  function resetOrderManagement() {
    loadedUserId = '';
    orders = [];
    products = [];
    selectedProductId = '';
    orderPagination = { ...emptyOrderPagination };
    points = null;
    realtimeToasts = [];
    closePointsStream('Disconnected');
  }

  function activeUserId() {
    return auth.user?.id || '';
  }

  async function loadOrderManagement() {
    await Promise.all([
      loadOrders(orderPagination.page),
      loadProducts(),
      loadPoints()
    ]);
  }

  async function loadProducts() {
    error = '';
    try {
      const result = await getProducts();
      products = Array.isArray(result) ? result : [];
      if (!selectedProductId || !checkoutProducts().some((product) => product.id === selectedProductId)) {
        selectedProductId = checkoutProducts()[0]?.id || '';
      }
    } catch (err) {
      error = err.message || 'Failed to load products';
    }
  }

  async function loadOrders(page = orderPagination.page) {
    const userId = activeUserId();
    if (!userId) return;

    loadingOrders = true;
    error = '';
    try {
      const result = await getMyOrders({
        page,
        pageSize: orderPageSize
      });
      orders = Array.isArray(result?.items) ? result.items : [];
      orderPagination = {
        ...emptyOrderPagination,
        ...(result?.pagination || {})
      };
    } catch (err) {
      error = err.message || 'Failed to load orders';
    } finally {
      loadingOrders = false;
    }
  }

  async function loadPoints() {
    error = '';
    try {
      const result = await getMyPoints();
      points = result.balance;
    } catch (err) {
      error = err.message || 'Failed to load points';
    }
  }

  async function submitOrder() {
    const userId = activeUserId();
    if (!userId) {
      error = 'Please sign in before creating an order';
      return;
    }
    if (!selectedProductId) {
      error = 'Select an enabled product before creating an order';
      return;
    }

    creatingOrder = true;
    error = '';
    message = '';
    try {
      const result = await createOrder({ product_id: selectedProductId });
      const order = result?.order;
      if (!order?.id) {
        throw new Error('Order was created without an id');
      }
      await startPaymentCheckout(order, { refreshPage: 1 });
    } catch (err) {
      error = err.message || 'Failed to create order';
    } finally {
      creatingOrder = false;
    }
  }

  async function startPaymentCheckout(order, options = {}) {
    const refreshPage = options.refreshPage || orderPagination.page;

    payingOrderId = order.id;
    error = '';
    message = '';
    try {
      const result = await createOrderPaymentCheckout(order.id);
      message = `Order ${result.order.id} is waiting for Creem confirmation`;
      if (result.checkout_url) {
        globalThis.location.assign(result.checkout_url);
        return;
      }
      await loadOrders(refreshPage);
    } catch (err) {
      error = err.message || 'Failed to create checkout';
      await loadOrders(refreshPage);
    } finally {
      payingOrderId = '';
    }
  }

  async function goToOrdersPage(page) {
    if (loadingOrders || page < 1 || page === orderPagination.page) return;
    const totalPages = Number(orderPagination.total_pages || 0);
    if (totalPages > 0 && page > totalPages) return;
    await loadOrders(page);
  }

  function visibleOrderPages() {
    const totalPages = Number(orderPagination.total_pages || 0);
    const currentPage = Number(orderPagination.page || 1);
    if (totalPages <= 1) return [];

    const maxButtons = 5;
    const pages = [];
    let start = Math.max(1, currentPage - Math.floor(maxButtons / 2));
    let end = Math.min(totalPages, start + maxButtons - 1);
    start = Math.max(1, end - maxButtons + 1);

    for (let page = start; page <= end; page += 1) {
      pages.push(page);
    }
    return pages;
  }

  function connectPointsStream() {
    closePointsStream();
    streamStatus = 'Connecting';

    pointsStream = new EventSource(pointsSSEURL());
    pointsStream.onopen = () => {
      streamStatus = 'Connected';
    };
    pointsStream.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data);
        dispatchRealtimeMessage(payload, {
          refreshPoints(nextPoints) {
            if (Number.isFinite(Number(nextPoints.balance))) {
              points = Number(nextPoints.balance);
            }
          },
          toast: addRealtimeToast
        });
      } catch {
        // Ignore malformed realtime messages; HTTP refresh remains available.
      }
    };
    pointsStream.onerror = () => {
      streamStatus = 'Error';
    };
  }

  function closePointsStream(nextStatus = 'Disconnected') {
    if (pointsStream) {
      pointsStream.close();
      pointsStream = undefined;
    }
    streamStatus = nextStatus;
  }

  function addRealtimeToast(toast) {
    const id = `${toast.id || 'toast'}-${Date.now()}-${Math.random().toString(16).slice(2)}`;
    realtimeToasts = [
      ...realtimeToasts,
      {
        ...toast,
        id
      }
    ];

    setTimeout(() => {
      realtimeToasts = realtimeToasts.filter((item) => item.id !== id);
    }, 5000);
  }

  function realtimeToastClass(toast) {
    if (toast.level === 'success') return 'alert-success';
    if (toast.level === 'error') return 'alert-error';
    return 'alert-info';
  }

  function money(value) {
    return `$${(Number(value || 0) / 100).toFixed(2)}`;
  }

  function checkoutProducts() {
    return products.filter((product) => product.enabled && product.creem_product_id);
  }

  function selectedProduct() {
    return products.find((product) => product.id === selectedProductId) || null;
  }

  function productPriceLabel(product) {
    if (!product) return 'Creem checkout price';
    const price = Number(product.price || 0);
    if (price <= 0) return 'Creem checkout price';
    return `${product.currency || 'USD'} ${(price / 100).toFixed(2)}`;
  }

  function membershipLabel(value) {
    if (value === 'premium') return 'Premium';
    if (value === 'super') return 'Super';
    return 'Basic';
  }

  function subscriptionLabel(value) {
    if (value === 'active') return 'active';
    if (value === 'canceled') return 'canceled';
    return '--';
  }

  function formatDate(value) {
    return formatLocalDateTime(value);
  }

  function orderAmountLabel(order) {
    const amount = Number(order.amount || 0);
    return amount > 0 ? money(amount) : 'Creem';
  }
</script>

{#if realtimeToasts.length > 0}
  <div class="toast toast-top toast-end z-50">
    {#each realtimeToasts as toast (toast.id)}
      <div class="alert {realtimeToastClass(toast)}">
        <span>{toast.message}</span>
      </div>
    {/each}
  </div>
{/if}

<section class="grid gap-6 xl:grid-cols-[0.72fr_1.28fr]">
  <div class="space-y-6">
    <div class="card border border-base-300 bg-base-100 shadow-lg">
      <div class="card-body gap-4">
        <div class="flex items-start justify-between gap-4">
          <div>
            <p class="text-sm text-base-content/60">Current user</p>
            <h1 class="text-2xl font-bold leading-tight">{auth.user?.name || 'Not signed in'}</h1>
            <p class="mt-1 text-sm text-base-content/60">{auth.user?.id || 'Sign in to continue'}</p>
          </div>
          <span class="badge {streamStatus === 'Connected' ? 'badge-success' : 'badge-outline'}">{streamStatus}</span>
        </div>

        <div class="rounded-lg border border-base-300 bg-base-200/50 p-4">
          <div class="text-sm text-base-content/60">Points balance</div>
          <div class="mt-2 text-4xl font-bold">{points === null ? '--' : points}</div>
        </div>

        <div class="rounded-lg border border-base-300 bg-base-200/50 p-4">
          <div class="text-sm text-base-content/60">Membership</div>
          <div class="mt-2 text-2xl font-bold">{membershipLabel(auth.user?.membership_level)}</div>
          <div class="mt-1 text-sm text-base-content/60">{formatDate(auth.user?.membership_expires_at)}</div>
        </div>

        {#if !auth.logged_in}
          <Notice type="warning" message="Please sign in before managing orders" />
        {/if}
      </div>
    </div>

    <div class="card border border-base-300 bg-base-100 shadow-lg">
      <div class="card-body gap-4">
        <div>
          <h2 class="card-title text-xl">Creem Checkout</h2>
          <p class="text-sm text-base-content/60">Selected local product maps to Creem.</p>
        </div>

        <label class="form-control">
          <span class="label"><span class="label-text">Product</span></span>
          <select class="select select-bordered" bind:value={selectedProductId} disabled={checkoutProducts().length === 0}>
            {#if checkoutProducts().length === 0}
              <option value="">No enabled Creem products</option>
            {:else}
              {#each checkoutProducts() as product}
                <option value={product.id}>{product.name}</option>
              {/each}
            {/if}
          </select>
        </label>

        <div class="rounded-lg border border-base-300 bg-base-200/50 p-4">
          <div class="text-sm text-base-content/60">Amount</div>
          <div class="mt-1 text-lg font-semibold">{productPriceLabel(selectedProduct())}</div>
          {#if selectedProduct()}
            <div class="mt-1 text-sm text-base-content/60">
              {membershipLabel(selectedProduct().membership_level)} - {selectedProduct().billing_type === 'subscription' ? selectedProduct().subscription_interval : 'permanent'}
            </div>
          {/if}
        </div>

        <button class="btn btn-primary" type="button" onclick={submitOrder} disabled={!auth.logged_in || !selectedProductId || creatingOrder || Boolean(payingOrderId)}>
          {#if creatingOrder || payingOrderId}
            <span class="loading loading-spinner loading-sm"></span>
          {/if}
          Create and Pay
        </button>
      </div>
    </div>
  </div>

  <div class="card border border-base-300 bg-base-100 shadow-lg">
    <div class="card-body gap-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="card-title text-xl">Orders</h2>
          <p class="text-sm text-base-content/60">Webhook-confirmed payments receive 10 points</p>
        </div>
        <button class="btn btn-outline btn-sm" type="button" onclick={loadOrderManagement} disabled={!auth.logged_in || loadingOrders}>
          {#if loadingOrders}
            <span class="loading loading-spinner loading-xs"></span>
          {/if}
          Refresh
        </button>
      </div>

      <Notice type="success" message={message} />
      <Notice type="error" message={error} />

      {#if orders.length === 0}
        <div class="rounded-lg border border-dashed border-base-300 p-8 text-center text-base-content/60">
          {loadingOrders ? 'Loading orders...' : 'No orders yet'}
        </div>
      {:else}
        <div class="overflow-x-auto">
          <table class="table table-sm">
            <thead>
              <tr>
                <th>Order</th>
                <th>Product</th>
                <th>User</th>
                <th>Status</th>
                <th>Subscription</th>
                <th class="text-right">Amount</th>
                <th class="text-right">Action</th>
              </tr>
            </thead>
            <tbody>
              {#each orders as order}
                <tr>
                  <td class="max-w-48 truncate font-mono text-xs">{order.id}</td>
                  <td>
                    <div>{order.product_name || order.product_id || '--'}</div>
                    {#if order.provider_subscription_id}
                      <div class="max-w-44 truncate font-mono text-xs text-base-content/50">{order.provider_subscription_id}</div>
                    {/if}
                  </td>
                  <td>{order.user_name || order.user_id}</td>
                  <td>
                    <span class="badge {order.status === 'paid' ? 'badge-success' : 'badge-neutral'}">
                      {orderStatusLabel(order.status)}
                    </span>
                  </td>
                  <td>
                    <span class="badge {order.subscription_status === 'canceled' ? 'badge-outline' : order.subscription_status === 'active' ? 'badge-info' : 'badge-ghost'}">
                      {subscriptionLabel(order.subscription_status)}
                    </span>
                  </td>
                  <td class="text-right">{orderAmountLabel(order)}</td>
                  <td class="text-right">
                    {#if order.status === 'pending'}
                      <button class="btn btn-primary btn-xs" type="button" onclick={() => startPaymentCheckout(order)} disabled={payingOrderId === order.id || creatingOrder}>
                        {#if payingOrderId === order.id}
                          <span class="loading loading-spinner loading-xs"></span>
                        {/if}
                        Pay
                      </button>
                    {:else}
                      <span class="text-sm text-base-content/50">--</span>
                    {/if}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>

        <div class="flex flex-col gap-3 border-t border-base-300 pt-4 sm:flex-row sm:items-center sm:justify-between">
          <div class="text-sm text-base-content/60">
            {orderPagination.total_items} orders - Page {orderPagination.page} / {Math.max(orderPagination.total_pages, 1)}
          </div>
          {#if orderPagination.total_pages > 1}
            <div class="max-w-full overflow-x-auto">
              <div class="join">
                <button
                  class="btn join-item btn-sm"
                  type="button"
                  onclick={() => goToOrdersPage(orderPagination.page - 1)}
                  disabled={loadingOrders || !orderPagination.has_previous}
                >
                  Prev
                </button>
                {#each visibleOrderPages() as page}
                  <button
                    class="btn join-item btn-sm {page === orderPagination.page ? 'btn-active' : ''}"
                    type="button"
                    onclick={() => goToOrdersPage(page)}
                    disabled={loadingOrders}
                  >
                    {page}
                  </button>
                {/each}
                <button
                  class="btn join-item btn-sm"
                  type="button"
                  onclick={() => goToOrdersPage(orderPagination.page + 1)}
                  disabled={loadingOrders || !orderPagination.has_next}
                >
                  Next
                </button>
              </div>
            </div>
          {/if}
        </div>
      {/if}
    </div>
  </div>
</section>

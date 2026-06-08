<script>
  import { onDestroy } from 'svelte';

  import {
    createOrder,
    createOrderPaymentCheckout,
    getMyPoints,
    getProducts,
    getUserOrders,
    pointsSSEURL
  } from '../api.js';
  import Notice from '../components/Notice.svelte';
  import { orderStatusLabel } from '../enums/orderStatus.ts';
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
  let orderPagination = $state({ ...emptyOrderPagination });
  let products = $state([]);
  let selectedProductId = $state('');
  let quantity = $state(1);
  let points = $state(null);
  let loadingOrders = $state(false);
  let loadingProducts = $state(false);
  let creatingOrder = $state(false);
  let payingOrderId = $state('');
  let error = $state('');
  let message = $state('');
  let realtimeToasts = $state([]);
  let streamStatus = $state('未连接');
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
    orderPagination = { ...emptyOrderPagination };
    products = [];
    selectedProductId = '';
    points = null;
    realtimeToasts = [];
    closePointsStream('未连接');
  }

  function activeUserId() {
    return auth.user?.id || '';
  }

  async function loadOrderManagement() {
    await Promise.all([
      loadProducts(),
      loadOrders(orderPagination.page),
      loadPoints()
    ]);
  }

  async function loadProducts() {
    loadingProducts = true;
    error = '';
    try {
      products = await getProducts();
      if (!selectedProductId && products.length > 0) {
        selectedProductId = products[0].id;
      }
    } catch (err) {
      error = err.message || '加载商品失败';
    } finally {
      loadingProducts = false;
    }
  }

  async function loadOrders(page = orderPagination.page) {
    const userId = activeUserId();
    if (!userId) return;

    loadingOrders = true;
    error = '';
    try {
      const result = await getUserOrders(userId, {
        page,
        pageSize: orderPageSize
      });
      orders = Array.isArray(result?.items) ? result.items : [];
      orderPagination = {
        ...emptyOrderPagination,
        ...(result?.pagination || {})
      };
    } catch (err) {
      error = err.message || '加载订单失败';
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
      error = err.message || '加载积分失败';
    }
  }

  async function submitOrder() {
    const userId = activeUserId();
    const parsedQuantity = Number(quantity);
    if (!userId || !selectedProductId || parsedQuantity <= 0) {
      error = '请选择商品并填写有效数量';
      return;
    }

    creatingOrder = true;
    error = '';
    message = '';
    try {
      const result = await createOrder({
        user_id: userId,
        items: [{ product_id: selectedProductId, quantity: parsedQuantity }]
      });
      message = `订单 ${result.order.id} 已创建`;
      quantity = 1;
      await Promise.all([loadOrders(1), loadProducts()]);
    } catch (err) {
      error = err.message || '创建订单失败';
    } finally {
      creatingOrder = false;
    }
  }

  async function startPaymentCheckout(order) {
    payingOrderId = order.id;
    error = '';
    message = '';
    try {
      const result = await createOrderPaymentCheckout(order.id);
      message = `订单 ${result.order.id} 已创建 Creem 支付，等待回调确认。`;
      if (result.checkout_url) {
        globalThis.location.assign(result.checkout_url);
      }
      await loadOrders(orderPagination.page);
    } catch (err) {
      error = err.message || '创建支付链接失败';
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
    streamStatus = '连接中';

    pointsStream = new EventSource(pointsSSEURL());
    pointsStream.onopen = () => {
      streamStatus = '已连接';
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
      streamStatus = '连接异常';
    };
  }

  function closePointsStream(nextStatus = '已断开') {
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
    return `¥${(Number(value || 0) / 100).toFixed(2)}`;
  }

  function selectedProduct() {
    return products.find((product) => product.id === selectedProductId);
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
            <p class="text-sm text-base-content/60">当前用户</p>
            <h1 class="text-2xl font-bold leading-tight">{auth.user?.name || '未登录'}</h1>
            <p class="mt-1 text-sm text-base-content/60">{auth.user?.id || '请先登录'}</p>
          </div>
          <span class="badge {streamStatus === '已连接' ? 'badge-success' : 'badge-outline'}">{streamStatus}</span>
        </div>

        <div class="rounded-lg border border-base-300 bg-base-200/50 p-4">
          <div class="text-sm text-base-content/60">积分余额</div>
          <div class="mt-2 text-4xl font-bold">{points === null ? '--' : points}</div>
        </div>

        {#if !auth.logged_in}
          <Notice type="warning" message="请先登录后再管理订单。" />
        {/if}
      </div>
    </div>

    <div class="card border border-base-300 bg-base-100 shadow-lg">
      <div class="card-body gap-4">
        <div class="flex items-center justify-between gap-4">
          <h2 class="card-title text-xl">下订单</h2>
          {#if loadingProducts}
            <span class="loading loading-spinner loading-sm"></span>
          {/if}
        </div>

        <label class="form-control">
          <span class="label">
            <span class="label-text">商品</span>
          </span>
          <select class="select select-bordered" bind:value={selectedProductId} disabled={!auth.logged_in || products.length === 0}>
            {#each products as product}
              <option value={product.id}>{product.name} · {money(product.price)} · 库存 {product.stock}</option>
            {/each}
          </select>
        </label>

        <label class="form-control">
          <span class="label">
            <span class="label-text">数量</span>
          </span>
          <input
            class="input input-bordered"
            type="number"
            min="1"
            bind:value={quantity}
            disabled={!auth.logged_in}
          />
        </label>

        {#if selectedProduct()}
          <div class="rounded-lg bg-base-200/60 p-3 text-sm text-base-content/70">
            小计：{money(selectedProduct().price * Number(quantity || 0))}
          </div>
        {/if}

        <button class="btn btn-primary" type="button" onclick={submitOrder} disabled={!auth.logged_in || creatingOrder || products.length === 0}>
          {#if creatingOrder}
            <span class="loading loading-spinner loading-sm"></span>
          {/if}
          创建订单
        </button>
      </div>
    </div>
  </div>

  <div class="card border border-base-300 bg-base-100 shadow-lg">
    <div class="card-body gap-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="card-title text-xl">订单列表</h2>
          <p class="text-sm text-base-content/60">Creem 回调确认支付后才会赠送 {10} 积分。</p>
        </div>
        <button class="btn btn-outline btn-sm" type="button" onclick={loadOrderManagement} disabled={!auth.logged_in || loadingOrders}>
          {#if loadingOrders}
            <span class="loading loading-spinner loading-xs"></span>
          {/if}
          刷新
        </button>
      </div>

      <Notice type="success" message={message} />
      <Notice type="error" message={error} />

      {#if orders.length === 0}
        <div class="rounded-lg border border-dashed border-base-300 p-8 text-center text-base-content/60">
          {loadingOrders ? '正在加载订单...' : '暂无订单'}
        </div>
      {:else}
        <div class="overflow-x-auto">
          <table class="table table-sm">
            <thead>
              <tr>
                <th>订单</th>
                <th>用户</th>
                <th>状态</th>
                <th class="text-right">金额</th>
                <th class="text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              {#each orders as order}
                <tr>
                  <td class="max-w-48 truncate font-mono text-xs">{order.id}</td>
                  <td>{order.user_name || order.user_id}</td>
                  <td>
                    <span class="badge {order.status === 'paid' ? 'badge-success' : 'badge-neutral'}">
                      {orderStatusLabel(order.status)}
                    </span>
                  </td>
                  <td class="text-right">{money(order.amount)}</td>
                  <td class="text-right">
                    {#if order.status === 'pending'}
                      <button class="btn btn-primary btn-xs" type="button" onclick={() => startPaymentCheckout(order)} disabled={payingOrderId === order.id}>
                        {#if payingOrderId === order.id}
                          <span class="loading loading-spinner loading-xs"></span>
                        {/if}
                        去支付
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
            {orderPagination.total_items} orders · Page {orderPagination.page} / {Math.max(orderPagination.total_pages, 1)}
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

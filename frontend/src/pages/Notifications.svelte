<script>
  import { onMount } from 'svelte';

  import { getDictionaries, listNotifications } from '../api.js';
  import Notice from '../components/Notice.svelte';
  import { formatLocalDateTime } from '../helpers/dateTime.js';

  const notificationPageSize = 10;
  const emptyNotificationPagination = {
    page: 1,
    page_size: notificationPageSize,
    total_items: 0,
    total_pages: 0,
    has_previous: false,
    has_next: false
  };

  let notifications = $state([]);
  let typeOptions = $state([]);
  let notificationPagination = $state({ ...emptyNotificationPagination });
  let filters = $state({
    type: '',
    email: '',
    phone: ''
  });
  let loadingNotifications = $state(false);
  let loadingDictionaries = $state(false);
  let error = $state('');

  onMount(() => {
    loadDictionaries();
    loadNotifications(1);
  });

  async function loadDictionaries() {
    loadingDictionaries = true;
    try {
      const result = await getDictionaries(['notification_type']);
      typeOptions = Array.isArray(result?.dictionaries?.notification_type)
        ? result.dictionaries.notification_type
        : [];
    } catch (err) {
      error = err.message || 'Failed to load notification types';
    } finally {
      loadingDictionaries = false;
    }
  }

  async function loadNotifications(page = notificationPagination.page) {
    loadingNotifications = true;
    error = '';
    try {
      const result = await listNotifications({
        page,
        pageSize: notificationPageSize,
        type: filters.type,
        email: filters.email.trim(),
        phone: filters.phone.trim()
      });
      notifications = Array.isArray(result?.items) ? result.items : [];
      notificationPagination = {
        ...emptyNotificationPagination,
        ...(result?.pagination || {})
      };
    } catch (err) {
      error = err.message || 'Failed to load notifications';
    } finally {
      loadingNotifications = false;
    }
  }

  async function applyFilters() {
    await loadNotifications(1);
  }

  async function clearFilters() {
    filters = {
      type: '',
      email: '',
      phone: ''
    };
    await loadNotifications(1);
  }

  async function goToNotificationsPage(page) {
    if (loadingNotifications || page < 1 || page === notificationPagination.page) return;
    const totalPages = Number(notificationPagination.total_pages || 0);
    if (totalPages > 0 && page > totalPages) return;
    await loadNotifications(page);
  }

  function visibleNotificationPages() {
    const totalPages = Number(notificationPagination.total_pages || 0);
    const currentPage = Number(notificationPagination.page || 1);
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

  function formatDate(value) {
    return formatLocalDateTime(value);
  }

  function typeLabel(notification) {
    return notification.notification_type_label || notification.notification_type || '--';
  }

  function sourceText(notification) {
    if (!notification.source_type && !notification.source_id) return '--';
    if (!notification.source_id) return notification.source_type;
    if (!notification.source_type) return notification.source_id;
    return `${notification.source_type}:${notification.source_id}`;
  }

  function recipientText(notification) {
    if (notification.recipient_email && notification.recipient_phone) {
      return `${notification.recipient_email} / ${notification.recipient_phone}`;
    }
    return notification.recipient_email || notification.recipient_phone || notification.user_id || '--';
  }

  function statusClass(status) {
    if (status === 'sent') return 'badge-success';
    if (status === 'failed') return 'badge-error';
    if (status === 'pending') return 'badge-warning';
    if (status === 'skipped') return 'badge-neutral';
    return 'badge-outline';
  }
</script>

<section class="space-y-6">
  <div class="flex flex-wrap items-start justify-between gap-4">
    <div>
      <h1 class="text-2xl font-bold leading-tight">Notification</h1>
      <p class="mt-1 text-sm text-base-content/60">Business notification ledger and realtime delivery slice.</p>
    </div>
    <button class="btn btn-outline btn-sm" type="button" onclick={() => loadNotifications(notificationPagination.page)} disabled={loadingNotifications}>
      {#if loadingNotifications}
        <span class="loading loading-spinner loading-xs"></span>
      {/if}
      Refresh
    </button>
  </div>

  <Notice type="error" message={error} />

  <div class="card border border-base-300 bg-base-100 shadow-sm">
    <div class="card-body gap-4">
      <div class="flex flex-wrap items-end gap-3">
        <label class="form-control w-full sm:w-52">
          <span class="label-text">Type</span>
          <select class="select select-bordered select-sm" bind:value={filters.type} disabled={loadingDictionaries}>
            <option value="">All</option>
            {#each typeOptions as option}
              <option value={option.value}>{option.label}</option>
            {/each}
          </select>
        </label>
        <label class="form-control w-full sm:w-64">
          <span class="label-text">Email</span>
          <input class="input input-bordered input-sm" type="search" bind:value={filters.email} placeholder="user@example.com" />
        </label>
        <label class="form-control w-full sm:w-56">
          <span class="label-text">Phone</span>
          <input class="input input-bordered input-sm" type="search" bind:value={filters.phone} placeholder="13800000000" />
        </label>
        <div class="flex gap-2">
          <button class="btn btn-primary btn-sm" type="button" onclick={applyFilters} disabled={loadingNotifications}>Filter</button>
          <button class="btn btn-ghost btn-sm" type="button" onclick={clearFilters} disabled={loadingNotifications}>Clear</button>
        </div>
      </div>
    </div>
  </div>

  <div class="card border border-base-300 bg-base-100 shadow-sm">
    <div class="card-body gap-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <h2 class="card-title text-lg">Ledger</h2>
        <span class="badge badge-outline">{notificationPagination.total_items}</span>
      </div>

      {#if notifications.length === 0}
        <div class="rounded border border-dashed border-base-300 p-6 text-center text-sm text-base-content/60">
          {loadingNotifications ? 'Loading notifications...' : 'No notifications'}
        </div>
      {:else}
        <div class="overflow-x-auto">
          <table class="table table-sm">
            <thead>
              <tr>
                <th>Notification</th>
                <th>Type</th>
                <th>Status</th>
                <th>Recipient</th>
                <th>Source</th>
                <th>Created</th>
                <th>Sent</th>
              </tr>
            </thead>
            <tbody>
              {#each notifications as notification}
                <tr>
                  <td>
                    <div class="font-medium">{notification.title}</div>
                    <div class="max-w-72 truncate text-xs text-base-content/60">{notification.summary || '--'}</div>
                    <div class="max-w-56 truncate font-mono text-xs text-base-content/40">{notification.id}</div>
                  </td>
                  <td>
                    <span class="badge badge-outline">{typeLabel(notification)}</span>
                  </td>
                  <td>
                    <span class="badge {statusClass(notification.status)}">{notification.status}</span>
                    {#if notification.last_error}
                      <div class="mt-1 max-w-48 truncate text-xs text-error">{notification.last_error}</div>
                    {/if}
                  </td>
                  <td>
                    <div class="max-w-64 truncate text-sm">{recipientText(notification)}</div>
                    {#if notification.user_id}
                      <div class="max-w-48 truncate font-mono text-xs text-base-content/50">{notification.user_id}</div>
                    {/if}
                  </td>
                  <td class="max-w-48 truncate font-mono text-xs">{sourceText(notification)}</td>
                  <td class="whitespace-nowrap text-xs">{formatDate(notification.created_at)}</td>
                  <td class="whitespace-nowrap text-xs">{formatDate(notification.sent_at)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>

        <div class="flex flex-col gap-3 border-t border-base-300 pt-4 sm:flex-row sm:items-center sm:justify-between">
          <div class="text-sm text-base-content/60">
            {notificationPagination.total_items} notifications - Page {notificationPagination.page} / {Math.max(notificationPagination.total_pages, 1)}
          </div>
          {#if notificationPagination.total_pages > 1}
            <div class="max-w-full overflow-x-auto">
              <div class="join">
                <button
                  class="btn join-item btn-sm"
                  type="button"
                  onclick={() => goToNotificationsPage(notificationPagination.page - 1)}
                  disabled={loadingNotifications || !notificationPagination.has_previous}
                >
                  Prev
                </button>
                {#each visibleNotificationPages() as page}
                  <button
                    class="btn join-item btn-sm {page === notificationPagination.page ? 'btn-active' : ''}"
                    type="button"
                    onclick={() => goToNotificationsPage(page)}
                    disabled={loadingNotifications}
                  >
                    {page}
                  </button>
                {/each}
                <button
                  class="btn join-item btn-sm"
                  type="button"
                  onclick={() => goToNotificationsPage(notificationPagination.page + 1)}
                  disabled={loadingNotifications || !notificationPagination.has_next}
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

<script>
  import { onMount } from 'svelte';

  import { listUsers, setUserActive } from '../api.js';
  import Notice from '../components/Notice.svelte';
  import { formatLocalDateTime } from '../helpers/dateTime.js';

  const userPageSize = 10;
  const emptyUserPagination = {
    page: 1,
    page_size: userPageSize,
    total_items: 0,
    total_pages: 0,
    has_previous: false,
    has_next: false
  };

  let { auth = { user: null } } = $props();

  let users = $state([]);
  let userPagination = $state({ ...emptyUserPagination });
  let loadingUsers = $state(false);
  let updatingUserId = $state('');
  let error = $state('');
  let message = $state('');

  onMount(() => {
    loadUsers(1);
  });

  async function loadUsers(page = userPagination.page) {
    loadingUsers = true;
    error = '';
    try {
      const result = await listUsers({
        page,
        pageSize: userPageSize
      });
      users = Array.isArray(result?.items) ? result.items : [];
      userPagination = {
        ...emptyUserPagination,
        ...(result?.pagination || {})
      };
    } catch (err) {
      error = err.message || 'Failed to load users';
    } finally {
      loadingUsers = false;
    }
  }

  async function toggleUser(user) {
    if (!user?.id || updatingUserId) return;

    updatingUserId = user.id;
    error = '';
    message = '';
    try {
      const updated = await setUserActive(user.id, !user.is_active);
      message = updated.is_active ? 'User enabled' : 'User disabled';
      await loadUsers(userPagination.page);
    } catch (err) {
      error = err.message || 'Failed to update user';
    } finally {
      updatingUserId = '';
    }
  }

  async function goToUsersPage(page) {
    if (loadingUsers || page < 1 || page === userPagination.page) return;
    const totalPages = Number(userPagination.total_pages || 0);
    if (totalPages > 0 && page > totalPages) return;
    await loadUsers(page);
  }

  function visibleUserPages() {
    const totalPages = Number(userPagination.total_pages || 0);
    const currentPage = Number(userPagination.page || 1);
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

  function isCurrentUser(user) {
    return Boolean(user?.id && auth?.user?.id === user.id);
  }

  function formatDate(value) {
    return formatLocalDateTime(value);
  }
</script>

<section class="space-y-6">
  <div class="flex flex-wrap items-start justify-between gap-4">
    <div>
      <h1 class="text-2xl font-bold leading-tight">User</h1>
      <p class="mt-1 text-sm text-base-content/60">Account status and access control.</p>
    </div>
    <button class="btn btn-outline btn-sm" type="button" onclick={() => loadUsers(userPagination.page)} disabled={loadingUsers}>
      {#if loadingUsers}
        <span class="loading loading-spinner loading-xs"></span>
      {/if}
      Refresh
    </button>
  </div>

  <Notice type="success" message={message} />
  <Notice type="error" message={error} />

  <div class="card border border-base-300 bg-base-100 shadow-sm">
    <div class="card-body gap-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <h2 class="card-title text-lg">Users</h2>
        <span class="badge badge-outline">{userPagination.total_items}</span>
      </div>

      {#if users.length === 0}
        <div class="rounded border border-dashed border-base-300 p-6 text-center text-sm text-base-content/60">
          {loadingUsers ? 'Loading users...' : 'No users'}
        </div>
      {:else}
        <div class="overflow-x-auto">
          <table class="table table-sm">
            <thead>
              <tr>
                <th>User</th>
                <th>Email</th>
                <th>Verified</th>
                <th>Status</th>
                <th>Created</th>
                <th class="text-right">Actions</th>
              </tr>
            </thead>
            <tbody>
              {#each users as user}
                <tr>
                  <td>
                    <div class="font-medium">{user.name || '--'}</div>
                    <div class="flex max-w-56 items-center gap-2">
                      <span class="truncate font-mono text-xs text-base-content/50">{user.id}</span>
                      {#if isCurrentUser(user)}
                        <span class="badge badge-outline badge-xs">you</span>
                      {/if}
                    </div>
                  </td>
                  <td class="max-w-60 truncate">{user.email}</td>
                  <td>
                    <span class="badge {user.email_verified ? 'badge-success' : 'badge-outline'}">
                      {user.email_verified ? 'verified' : 'unverified'}
                    </span>
                  </td>
                  <td>
                    <span class="badge {user.is_active ? 'badge-success' : 'badge-error'}">
                      {user.is_active ? 'active' : 'disabled'}
                    </span>
                  </td>
                  <td class="whitespace-nowrap text-xs">{formatDate(user.created_at)}</td>
                  <td class="text-right">
                    <button
                      class="btn btn-xs {user.is_active ? 'btn-outline' : 'btn-primary'}"
                      type="button"
                      onclick={() => toggleUser(user)}
                      disabled={updatingUserId === user.id || (isCurrentUser(user) && user.is_active)}
                    >
                      {#if updatingUserId === user.id}
                        <span class="loading loading-spinner loading-xs"></span>
                      {/if}
                      {user.is_active ? 'Disable' : 'Enable'}
                    </button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>

        <div class="flex flex-col gap-3 border-t border-base-300 pt-4 sm:flex-row sm:items-center sm:justify-between">
          <div class="text-sm text-base-content/60">
            {userPagination.total_items} users - Page {userPagination.page} / {Math.max(userPagination.total_pages, 1)}
          </div>
          {#if userPagination.total_pages > 1}
            <div class="max-w-full overflow-x-auto">
              <div class="join">
                <button
                  class="btn join-item btn-sm"
                  type="button"
                  onclick={() => goToUsersPage(userPagination.page - 1)}
                  disabled={loadingUsers || !userPagination.has_previous}
                >
                  Prev
                </button>
                {#each visibleUserPages() as page}
                  <button
                    class="btn join-item btn-sm {page === userPagination.page ? 'btn-active' : ''}"
                    type="button"
                    onclick={() => goToUsersPage(page)}
                    disabled={loadingUsers}
                  >
                    {page}
                  </button>
                {/each}
                <button
                  class="btn join-item btn-sm"
                  type="button"
                  onclick={() => goToUsersPage(userPagination.page + 1)}
                  disabled={loadingUsers || !userPagination.has_next}
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

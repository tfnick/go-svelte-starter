<script>
  import { listMyTasks } from '../api.js';

  let { refreshTrigger = 0 } = $props();
  let open = $state(false);
  let tasks = $state([]);
  let loading = $state(false);

  async function loadTasks() {
    loading = true;
    try {
      tasks = (await listMyTasks({ page: 1, pageSize: 20 }))?.items || [];
    } catch {
      tasks = [];
    } finally {
      loading = false;
    }
  }

  function toggle() {
    open = !open;
    if (open) {
      loadTasks();
    }
  }

  function statusBadge(status) {
    switch (status) {
      case 'queued': return 'badge-ghost';
      case 'processing': return 'badge-info';
      case 'completed': return 'badge-success';
      case 'failed': return 'badge-error';
      default: return 'badge-ghost';
    }
  }

  $effect(() => {
    if (refreshTrigger > 0 && open) {
      loadTasks();
    }
  });
</script>

<div class="fixed bottom-16 left-4 z-50">
  {#if open}
    <div class="card min-w-0 border border-base-200 bg-base-100 shadow-xl w-96 max-h-96 flex flex-col">
      <div class="card-body p-3 gap-2 overflow-y-auto">
        <div class="flex items-center justify-between">
          <h3 class="card-title text-sm">Tasks</h3>
          <div class="flex items-center gap-1">
            <button class="btn btn-ghost btn-xs" onclick={loadTasks} disabled={loading}>
              {#if loading}
                <span class="loading loading-spinner loading-xs"></span>
              {:else}
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
              {/if}
            </button>
            <button class="btn btn-ghost btn-xs" onclick={toggle}>&times;</button>
          </div>
        </div>
        {#if tasks.length === 0}
          <div class="text-sm text-base-content/50 py-4 text-center">No tasks</div>
        {:else}
          {#each tasks as task (task.id)}
            <div class="flex items-center justify-between gap-2 border-b border-base-200 pb-2">
              <div class="flex-1 min-w-0">
                <div class="text-xs font-medium truncate">{task.task_type}</div>
                {#if task.error_message}
                  <div class="text-xs text-error truncate">{task.error_message}</div>
                {/if}
              </div>
              <span class="badge badge-xs {statusBadge(task.status)}">{task.status}</span>
            </div>
          {/each}
        {/if}
      </div>
    </div>
  {/if}

  <button class="btn btn-circle btn-ghost" onclick={toggle} aria-label="Tasks">
    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
    </svg>
  </button>
</div>

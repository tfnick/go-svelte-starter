<script>
  import { ClipboardCheck, RefreshCw, X } from 'lucide-svelte';

  import { listMyTasks } from '../api.js';

  let { refreshTrigger = 0, docked = false } = $props();
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

<div class={docked ? '' : 'fixed bottom-16 left-4 z-50'}>
  {#if open}
    <div class={docked ? 'absolute bottom-12 left-0 z-50 flex max-h-96 w-[min(24rem,calc(100vw-2rem))] min-w-0 flex-col rounded-box border border-base-200 bg-base-100 shadow-xl' : 'card flex max-h-96 w-96 min-w-0 flex-col border border-base-200 bg-base-100 shadow-xl'}>
      <div class="card-body gap-2 overflow-y-auto p-3">
        <div class="flex items-center justify-between">
          <h3 class="card-title text-sm">Tasks</h3>
          <div class="flex items-center gap-1">
            <button class="btn btn-square btn-ghost btn-xs" type="button" aria-label="Refresh tasks" onclick={loadTasks} disabled={loading}>
              {#if loading}
                <span class="loading loading-spinner loading-xs"></span>
              {:else}
                <RefreshCw size={14} />
              {/if}
            </button>
            <button class="btn btn-square btn-ghost btn-xs" type="button" aria-label="Close tasks" onclick={toggle}>
              <X size={14} />
            </button>
          </div>
        </div>
        {#if tasks.length === 0}
          <div class="py-4 text-center text-sm text-base-content/50">No tasks</div>
        {:else}
          {#each tasks as task (task.id)}
            <div class="flex items-center justify-between gap-2 border-b border-base-200 pb-2">
              <div class="min-w-0 flex-1">
                <div class="truncate text-xs font-medium">{task.task_type}</div>
                {#if task.error_message}
                  <div class="truncate text-xs text-error">{task.error_message}</div>
                {/if}
              </div>
              <span class="badge badge-xs {statusBadge(task.status)}">{task.status}</span>
            </div>
          {/each}
        {/if}
      </div>
    </div>
  {/if}

  <button class={docked ? 'btn btn-square btn-ghost' : 'btn btn-circle btn-ghost'} type="button" onclick={toggle} aria-label="Tasks">
    <ClipboardCheck size={18} />
  </button>
</div>

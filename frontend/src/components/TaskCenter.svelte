<script>
  import { ClipboardCheck, Download, RefreshCw, X } from 'lucide-svelte';

  import { getTaskDownload, listMyTasks } from '../api.js';
  import { canDownloadTask, taskFilename, taskTitle } from '../helpers/tasks.js';

  let { refreshTrigger = 0, docked = false } = $props();
  let open = $state(false);
  let tasks = $state([]);
  let loading = $state(false);
  let downloadingTaskId = $state('');
  let downloadError = $state('');

  async function loadTasks() {
    loading = true;
    downloadError = '';
    try {
      tasks = (await listMyTasks({ page: 1, pageSize: 20 }))?.items || [];
    } catch {
      tasks = [];
    } finally {
      loading = false;
    }
  }

  async function downloadTask(task) {
    if (!task?.id || downloadingTaskId) return;

    downloadingTaskId = task.id;
    downloadError = '';
    try {
      const result = await getTaskDownload(task.id);
      if (result?.url) {
        globalThis.location?.assign(result.url);
      }
    } catch (err) {
      downloadError = err.message || 'Failed to prepare task download';
    } finally {
      downloadingTaskId = '';
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
    <div class={docked ? 'absolute inset-x-0 bottom-12 z-50 flex max-h-80 max-w-full min-w-0 flex-col rounded-box border border-base-200 bg-base-100 shadow-xl' : 'card flex max-h-96 w-96 min-w-0 flex-col border border-base-200 bg-base-100 shadow-xl'}>
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
        {#if downloadError}
          <div class="alert alert-error py-2 text-xs">{downloadError}</div>
        {/if}
        {#if tasks.length === 0}
          <div class="py-4 text-center text-sm text-base-content/50">No tasks</div>
        {:else}
          {#each tasks as task (task.id)}
            <div class="flex items-center justify-between gap-2 border-b border-base-200 pb-2">
              <div class="min-w-0 flex-1">
                <div class="truncate text-xs font-medium">{taskTitle(task)}</div>
                {#if canDownloadTask(task)}
                  <div class="truncate text-xs text-base-content/50">{taskFilename(task)}</div>
                {/if}
                {#if task.error_message}
                  <div class="truncate text-xs text-error">{task.error_message}</div>
                {/if}
              </div>
              <div class="flex shrink-0 items-center gap-1">
                <span class="badge badge-xs {statusBadge(task.status)}">{task.status}</span>
                {#if canDownloadTask(task)}
                  <button class="btn btn-ghost btn-xs gap-1" type="button" aria-label="Download task result" onclick={() => downloadTask(task)} disabled={downloadingTaskId === task.id}>
                    {#if downloadingTaskId === task.id}
                      <span class="loading loading-spinner loading-xs"></span>
                    {:else}
                      <Download size={14} />
                    {/if}
                    <span>Download</span>
                  </button>
                {/if}
              </div>
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

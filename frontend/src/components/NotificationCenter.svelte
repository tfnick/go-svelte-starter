<script>
  import { ArchiveX, Bell, X } from 'lucide-svelte';

  import { clearMyNotifications } from '../api.js';

  let { notifications = [], docked = false, onCleared } = $props();
  let open = $state(false);
  let clearing = $state(false);
  let notificationError = $state('');

  function count() {
    return notifications.length;
  }

  function toggle() {
    open = !open;
  }

  async function clearNotifications() {
    if (clearing || notifications.length === 0) return;

    clearing = true;
    notificationError = '';
    try {
      await clearMyNotifications();
      onCleared?.();
    } catch (err) {
      notificationError = err.message || 'Failed to clear notifications';
    } finally {
      clearing = false;
    }
  }

  function levelClass(level) {
    switch (level) {
      case 'success': return 'alert-success';
      case 'error': return 'alert-error';
      case 'info': return 'alert-info';
      default: return 'alert-info';
    }
  }
</script>

<div class={docked ? '' : 'fixed bottom-4 left-4 z-50'}>
  {#if open}
    <div class={docked ? 'absolute inset-x-0 bottom-12 z-50 flex max-h-80 max-w-full min-w-0 flex-col rounded-box border border-base-200 bg-base-100 shadow-xl' : 'card flex max-h-96 w-80 min-w-0 flex-col border border-base-200 bg-base-100 shadow-xl'}>
      <div class="card-body gap-2 overflow-y-auto p-3">
        <div class="flex items-center justify-between">
          <h3 class="card-title text-sm">Notifications</h3>
          <div class="flex items-center gap-1">
            <button class="btn btn-square btn-ghost btn-xs" type="button" aria-label="Clear notifications" onclick={clearNotifications} disabled={clearing || notifications.length === 0}>
              {#if clearing}
                <span class="loading loading-spinner loading-xs"></span>
              {:else}
                <ArchiveX size={14} />
              {/if}
            </button>
            <button class="btn btn-square btn-ghost btn-xs" type="button" aria-label="Close notifications" onclick={toggle}>
              <X size={14} />
            </button>
          </div>
        </div>
        {#if notificationError}
          <div class="alert alert-error py-2 text-xs">{notificationError}</div>
        {/if}
        {#if notifications.length === 0}
          <div class="py-4 text-center text-sm text-base-content/50">No notifications</div>
        {:else}
          {#each notifications as notification (notification.id)}
            <div class="alert {levelClass(notification.level)} min-w-0 p-2 text-xs">
              <span class="min-w-0 break-words">{notification.message}</span>
            </div>
          {/each}
        {/if}
      </div>
    </div>
  {/if}

  <button class={docked ? 'btn btn-square btn-ghost relative' : 'btn btn-circle btn-ghost relative'} type="button" onclick={toggle} aria-label="Notifications">
    <Bell size={18} />
    {#if count() > 0}
      <span class="badge badge-error badge-sm absolute -right-1 -top-1">{count()}</span>
    {/if}
  </button>
</div>

<script>
  let { notifications = [] } = $props();
  let open = $state(false);

  function count() {
    return notifications.length;
  }

  function toggle() {
    open = !open;
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

<div class="fixed bottom-4 left-4 z-50">
  {#if open}
    <div class="card border border-base-300 bg-base-100 shadow-xl w-80 max-h-96 flex flex-col">
      <div class="card-body p-3 gap-2 overflow-y-auto">
        <div class="flex items-center justify-between">
          <h3 class="card-title text-sm">Notifications</h3>
          <button class="btn btn-ghost btn-xs" onclick={toggle}>&times;</button>
        </div>
        {#if notifications.length === 0}
          <div class="text-sm text-base-content/50 py-4 text-center">No notifications</div>
        {:else}
          {#each notifications as notification (notification.id)}
            <div class="alert {levelClass(notification.level)} text-xs p-2">
              <span>{notification.message}</span>
            </div>
          {/each}
        {/if}
      </div>
    </div>
  {/if}

  <button class="btn btn-circle btn-ghost relative" onclick={toggle} aria-label="Notifications">
    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
    </svg>
    {#if count() > 0}
      <span class="badge badge-sm badge-error absolute -top-1 -right-1">{count()}</span>
    {/if}
  </button>
</div>

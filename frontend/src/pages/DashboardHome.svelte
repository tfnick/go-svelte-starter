<script>
  let { auth } = $props();
</script>

<section class="space-y-6">
  <div class="rounded-lg border border-base-300 bg-base-100 p-6 shadow-sm">
    <div class="max-w-3xl">
      <p class="text-sm font-semibold uppercase text-base-content/50">Dashboard</p>
      <h1 class="mt-2 text-3xl font-bold leading-tight">Welcome back, {auth.user?.name || 'user'}</h1>
      <p class="mt-3 text-base text-base-content/70">
        Your workspace is ready.
      </p>
    </div>
  </div>

  <div class="grid gap-4 md:grid-cols-3">
    <div class="rounded-lg border border-base-300 bg-base-100 p-5 shadow-sm">
      <div class="text-sm font-semibold text-base-content/60">Order</div>
      <div class="mt-2 text-2xl font-bold">Ready</div>
      <p class="mt-2 text-sm text-base-content/60">Order tools are available.</p>
    </div>

    <div class="rounded-lg border border-base-300 bg-base-100 p-5 shadow-sm">
      <div class="text-sm font-semibold text-base-content/60">Realtime</div>
      <div class="mt-2 text-2xl font-bold">Connected</div>
      <p class="mt-2 text-sm text-base-content/60">Notifications are online.</p>
    </div>

    <div class="rounded-lg border border-base-300 bg-base-100 p-5 shadow-sm">
      <div class="text-sm font-semibold text-base-content/60">Scheduler</div>
      <div class="mt-2 text-2xl font-bold">Ready</div>
      <p class="mt-2 text-sm text-base-content/60">Task and queue tools are available.</p>
    </div>
  </div>
</section>

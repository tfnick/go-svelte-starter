<script>
  import { navigate, normalizePath, visibleAppRoutes } from '../router.js';

  let { path, auth, children } = $props();
  let drawerOpen = $state(false);

  function activePath() {
    return normalizePath(path);
  }

  function go(routePath) {
    drawerOpen = false;
    navigate(routePath);
  }

  function routes() {
    return visibleAppRoutes(auth.user);
  }
</script>

<div class="drawer lg:drawer-open">
  <input id="app-sidebar-drawer" class="drawer-toggle" type="checkbox" bind:checked={drawerOpen} />

  <div class="drawer-content flex min-h-[calc(100vh-4rem)] min-w-0 flex-col">
    <div class="sticky top-0 z-20 flex items-center justify-between border-b border-base-200 bg-base-100/95 px-4 py-3 backdrop-blur lg:hidden">
      <button class="btn btn-square btn-ghost" type="button" aria-label="Open menu" onclick={() => { drawerOpen = true; }}>
        <span class="text-xs font-semibold">Menu</span>
      </button>
      <div class="text-sm font-semibold">{auth.user?.name || 'Menu'}</div>
      <div class="w-10"></div>
    </div>

    <div class="min-w-0 flex-1 px-4 py-6 sm:px-6 lg:px-8">
      {@render children?.()}
    </div>
  </div>

  <div class="drawer-side z-30">
    <label for="app-sidebar-drawer" aria-label="Close menu" class="drawer-overlay"></label>
    <aside class="flex min-h-full w-72 flex-col border-r border-base-200 bg-base-100">
      <div class="border-b border-base-200 px-5 py-5">
        <div class="text-xs font-semibold uppercase tracking-wide text-base-content/50">Workspace</div>
        <div class="mt-1 truncate text-lg font-bold">{auth.user?.name || 'Svelte Go Starter'}</div>
        <div class="truncate text-sm text-base-content/60">{auth.user?.id || 'Signed in'}</div>
      </div>

      <nav class="flex-1 p-3">
        <ul class="menu gap-1">
          {#each routes() as route}
            <li>
              <button
                class={activePath() === route.path ? 'active' : ''}
                type="button"
                onclick={() => go(route.path)}
              >
                <span>{route.label}</span>
                <span class="text-xs text-base-content/50">{route.description}</span>
              </button>
            </li>
          {/each}
        </ul>
      </nav>
    </aside>
  </div>
</div>

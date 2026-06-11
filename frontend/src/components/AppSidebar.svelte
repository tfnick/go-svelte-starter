<script>
  import {
    Bell,
    BookOpen,
    Boxes,
    CalendarClock,
    CreditCard,
    FlaskConical,
    Gauge,
    Home,
    ListChecks,
    LogOut,
    Menu,
    Package,
    Settings,
    SlidersHorizontal,
    Users,
    Workflow
  } from 'lucide-svelte';

  import { logout } from '../api.js';
  import { appHomePath, navigate, normalizePath, routeTitle, visibleAppRoutes } from '../router.js';
  import NotificationCenter from './NotificationCenter.svelte';
  import TaskCenter from './TaskCenter.svelte';

  let {
    path,
    auth,
    siteSettings,
    notifications = [],
    taskRefreshTrigger = 0,
    onAuthChanged,
    children
  } = $props();

  const routeIcons = {
    checkout: CreditCard,
    dashboard: Gauge,
    dictionary: BookOpen,
    events: Workflow,
    experiments: FlaskConical,
    notifications: Bell,
    orders: ListChecks,
    parameters: SlidersHorizontal,
    products: Package,
    scheduler: CalendarClock,
    settings: Settings,
    users: Users,
    variables: Boxes
  };

  let drawerOpen = $state(false);
  let busy = $state(false);
  let error = $state('');
  let logoURL = $state('/logo.png');

  $effect(() => {
    logoURL = siteSettings?.logo_url || '/logo.png';
  });

  function activePath() {
    return normalizePath(path);
  }

  function activeTitle() {
    return routeTitle(activePath());
  }

  function go(routePath) {
    drawerOpen = false;
    navigate(routePath);
  }

  function routes() {
    return visibleAppRoutes(auth.user);
  }

  function iconFor(route) {
    return routeIcons[route.icon] || Home;
  }

  function userInitial() {
    const name = auth.user?.name || auth.user?.email || auth.user?.id || 'U';
    return String(name).trim().slice(0, 1).toUpperCase() || 'U';
  }

  function profileLabel() {
    return auth.user?.name || auth.user?.email || 'Signed in';
  }

  async function handleLogout() {
    busy = true;
    error = '';
    try {
      await logout();
      onAuthChanged?.();
      drawerOpen = false;
      navigate(appHomePath);
    } catch (err) {
      error = err.message || 'Failed to sign out';
    } finally {
      busy = false;
    }
  }
</script>

<div class="drawer min-h-screen lg:drawer-open">
  <input id="app-sidebar-drawer" class="drawer-toggle" type="checkbox" bind:checked={drawerOpen} />

  <div class="drawer-content flex min-h-screen min-w-0 flex-col bg-base-200/40">
    <div class="sticky top-0 z-20 flex h-14 items-center justify-between border-b border-base-200 bg-base-100/95 px-3 backdrop-blur lg:hidden">
      <button class="btn btn-square btn-ghost" type="button" aria-label="Open menu" onclick={() => { drawerOpen = true; }}>
        <Menu size={20} />
      </button>
      <button class="btn btn-ghost min-w-0 flex-1 justify-center px-2" type="button" onclick={() => go(appHomePath)}>
        <img
          alt="Svelte Go Starter"
          class="h-[22px] w-[96px] object-contain"
          height="22"
          src={logoURL}
          width="96"
          onerror={() => {
            if (logoURL !== '/logo.png') {
              logoURL = '/logo.png';
            }
          }}
        />
      </button>
      <div class="badge badge-ghost max-w-28 truncate">{activeTitle()}</div>
    </div>

    <main class="min-w-0 flex-1 px-4 py-5 sm:px-6 lg:px-8 lg:py-7">
      {#if error}
        <div class="alert alert-error mb-4 py-2 text-sm">{error}</div>
      {/if}
      {@render children?.()}
    </main>
  </div>

  <div class="drawer-side z-30">
    <label for="app-sidebar-drawer" aria-label="Close menu" class="drawer-overlay"></label>
    <aside class="flex min-h-full w-72 flex-col border-r border-base-200 bg-base-100">
      <div class="flex h-20 items-center border-b border-base-200 px-5">
        <button class="btn btn-ghost h-auto min-h-0 px-0" type="button" onclick={() => go(appHomePath)}>
          <img
            alt="Svelte Go Starter"
            class="h-[25px] w-[120px] object-contain"
            height="25"
            src={logoURL}
            width="120"
            onerror={() => {
              if (logoURL !== '/logo.png') {
                logoURL = '/logo.png';
              }
            }}
          />
        </button>
      </div>

      <nav class="min-h-0 flex-1 overflow-y-auto p-3">
        <div class="flex flex-col gap-1">
          {#each routes() as route}
            {@const Icon = iconFor(route)}
            <button
              class="btn h-11 min-h-11 w-full justify-start gap-3 rounded-box border-0 {activePath() === route.path ? 'btn-primary' : 'btn-ghost'}"
              type="button"
              aria-current={activePath() === route.path ? 'page' : undefined}
              onclick={() => go(route.path)}
            >
              <Icon size={18} />
              <span class="truncate">{route.label}</span>
            </button>
          {/each}
        </div>
      </nav>

      <div class="border-t border-base-200 p-3">
        <div class="relative rounded-box bg-base-200/60 p-3">
          <div class="flex min-w-0 items-center gap-3">
            <div class="avatar placeholder">
              <div class="w-10 rounded-full bg-primary text-primary-content">
                <span class="text-sm font-semibold">{userInitial()}</span>
              </div>
            </div>
            <div class="min-w-0 flex-1">
              <div class="truncate text-sm font-semibold">{profileLabel()}</div>
              <div class="truncate text-xs text-base-content/55">{auth.user?.id || 'Current user'}</div>
            </div>
            {#if auth.user?.is_admin}
              <span class="badge badge-primary badge-sm">Admin</span>
            {/if}
          </div>

          <div class="mt-3 grid grid-cols-3 gap-2">
            <NotificationCenter {notifications} docked />
            <TaskCenter refreshTrigger={taskRefreshTrigger} docked />
            <button class="btn btn-square btn-ghost" type="button" aria-label="Sign out" onclick={handleLogout} disabled={busy}>
              {#if busy}
                <span class="loading loading-spinner loading-xs"></span>
              {:else}
                <LogOut size={18} />
              {/if}
            </button>
          </div>
        </div>
      </div>
    </aside>
  </div>
</div>

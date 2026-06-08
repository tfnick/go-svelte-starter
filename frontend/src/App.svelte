<script>
  import AppSidebar from './components/AppSidebar.svelte';
  import Header from './components/Header.svelte';
  import Dashboard from './pages/Dashboard.svelte';
  import DashboardHome from './pages/DashboardHome.svelte';
  import Dictionary from './pages/Dictionary.svelte';
  import Events from './pages/Events.svelte';
  import Experiments from './pages/Experiments.svelte';
  import ForgotPassword from './pages/ForgotPassword.svelte';
  import Login from './pages/Login.svelte';
  import Notifications from './pages/Notifications.svelte';
  import Parameters from './pages/Parameters.svelte';
  import Products from './pages/Products.svelte';
  import Register from './pages/Register.svelte';
  import ResetPassword from './pages/ResetPassword.svelte';
  import Scheduler from './pages/Scheduler.svelte';
  import Settings from './pages/Settings.svelte';
  import Users from './pages/Users.svelte';
  import Variables from './pages/Variables.svelte';
  import { getAuthStatus, getSiteSettings } from './api.js';
  import { isAuthRoute, normalizePath, routeTitle, visibleAppRoutes } from './router.js';
  import { onMount } from 'svelte';

  let path = $state(normalizePath());
  let auth = $state({ loading: true, logged_in: false, user: null });
  let siteSettings = $state({ logo_url: '/logo.png', logo_configured: false, logo_updated_at: '' });

  async function refreshAuth() {
    auth = { ...auth, loading: true };
    try {
      const status = await getAuthStatus();
      auth = {
        loading: false,
        logged_in: Boolean(status.logged_in),
        user: status.user || null
      };
    } catch {
      auth = { loading: false, logged_in: false, user: null };
    }
  }

  function syncRoute() {
    path = normalizePath();
    document.title = `${routeTitle(path)} - Svelte Go Starter`;
  }

  function handleAuthChanged() {
    refreshAuth();
  }

  async function refreshSiteSettings() {
    try {
      const settings = await getSiteSettings();
      siteSettings = {
        logo_url: settings?.logo_url || '/logo.png',
        logo_configured: Boolean(settings?.logo_configured),
        logo_updated_at: settings?.logo_updated_at || ''
      };
    } catch {
      siteSettings = { logo_url: '/logo.png', logo_configured: false, logo_updated_at: '' };
    }
  }

  function canAccessCurrentPath() {
    return visibleAppRoutes(auth.user).some((route) => route.path === path);
  }

  onMount(() => {
    syncRoute();
    refreshAuth();
    refreshSiteSettings();
    window.addEventListener('popstate', syncRoute);

    return () => {
      window.removeEventListener('popstate', syncRoute);
    };
  });
</script>

<div class="app-shell">
  <Header {auth} {siteSettings} onAuthChanged={handleAuthChanged} />

  {#if isAuthRoute(path)}
    <main class="page-wrap py-8">
      {#if path === '/login'}
        <Login onSuccess={handleAuthChanged} />
      {:else if path === '/register'}
        <Register onSuccess={handleAuthChanged} />
      {:else if path === '/forgot-password'}
        <ForgotPassword />
      {:else if path === '/reset-password'}
        <ResetPassword />
      {/if}
    </main>
  {:else if auth.loading}
    <main class="page-wrap py-8">
      <div class="flex min-h-64 items-center justify-center">
        <span class="loading loading-spinner loading-md" aria-label="Loading"></span>
      </div>
    </main>
  {:else if !auth.logged_in}
    <main class="page-wrap py-8">
      <Login onSuccess={handleAuthChanged} />
    </main>
  {:else}
    <AppSidebar path={canAccessCurrentPath() ? path : '/'} {auth}>
      {#snippet children()}
        {#if !canAccessCurrentPath()}
          <DashboardHome {auth} />
        {:else if path === '/orders'}
          <Dashboard {auth} />
        {:else if path === '/products'}
          <Products />
        {:else if path === '/users'}
          <Users {auth} />
        {:else if path === '/scheduler'}
          <Scheduler />
        {:else if path === '/events'}
          <Events />
        {:else if path === '/experiments'}
          <Experiments {auth} />
        {:else if path === '/dictionary'}
          <Dictionary />
        {:else if path === '/parameters'}
          <Parameters />
        {:else if path === '/notifications'}
          <Notifications />
        {:else if path === '/settings'}
          <Settings settings={siteSettings} onSettingsChanged={refreshSiteSettings} />
        {:else if path === '/variables'}
          <Variables />
        {:else}
          <DashboardHome {auth} />
        {/if}
      {/snippet}
    </AppSidebar>
  {/if}
</div>

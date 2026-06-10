<script>
  import AppSidebar from './components/AppSidebar.svelte';
  import Header from './components/Header.svelte';
  import AppCheckout from './pages/AppCheckout.svelte';
  import NotificationCenter from './components/NotificationCenter.svelte';
  import TaskCenter from './components/TaskCenter.svelte';
  import Dashboard from './pages/Dashboard.svelte';
  import DashboardHome from './pages/DashboardHome.svelte';
  import Dictionary from './pages/Dictionary.svelte';
  import Events from './pages/Events.svelte';
  import Experiments from './pages/Experiments.svelte';
  import ForgotPassword from './pages/ForgotPassword.svelte';
  import Login from './pages/Login.svelte';
  import OAuthCallback from './pages/OAuthCallback.svelte';
  import Notifications from './pages/Notifications.svelte';
  import Parameters from './pages/Parameters.svelte';
  import Products from './pages/Products.svelte';
  import Register from './pages/Register.svelte';
  import ResetPassword from './pages/ResetPassword.svelte';
  import Scheduler from './pages/Scheduler.svelte';
  import Settings from './pages/Settings.svelte';
  import Users from './pages/Users.svelte';
  import Variables from './pages/Variables.svelte';
  import { getAuthStatus, getSiteSettings, realtimeWebSocketURL } from './api.js';
  import { appHomePath, canAccessAppRoute, isAuthRoute, normalizePath, routeTitle, visibleAppRoutes } from './router.js';
  import { normalizeRealtimeMessage, toastFromRealtimeMessage } from './helpers/realtimeMessages.js';
  import { createRealtimeWebSocketClient } from './helpers/realtimeWebSocket.js';
  import { onMount } from 'svelte';

  let path = $state(normalizePath());
  let auth = $state({ loading: true, logged_in: false, user: null });

  function defaultSiteSettings() {
    return {
      logo_url: '/logo.png',
      logo_configured: false,
      logo_updated_at: '',
      logo_upload_available: false,
      logo_upload_unavailable_reason: 'Primary OSS provider is not configured'
    };
  }

  let siteSettings = $state(defaultSiteSettings());
  let notifications = $state([]);
  let taskRefreshTrigger = $state(0);
  const realtimeClient = createRealtimeWebSocketClient({
    url: realtimeWebSocketURL,
    shouldReconnect: () => auth.logged_in,
    onMessage(data) {
      const message = normalizeRealtimeMessage(data);
      if (!message) return;
      if (message.presentation === 'toast') {
        const toast = toastFromRealtimeMessage(message);
        addNotification(toast);
      }
      if (message.type === 'heavy_task') {
        taskRefreshTrigger++;
      }
    },
    onMalformedMessage() {
      // ignore malformed messages
    }
  });

  function addNotification(msg) {
    notifications = [{ ...msg, id: msg.id || Date.now() }, ...notifications].slice(0, 50);
  }

  async function refreshAuth() {
    auth = { ...auth, loading: true };
    try {
      const status = await getAuthStatus();
      auth = {
        loading: false,
        logged_in: Boolean(status.logged_in),
        user: status.user || null
      };
      if (auth.logged_in) {
        realtimeClient.connect();
      } else {
        realtimeClient.disconnect();
      }
    } catch {
      auth = { loading: false, logged_in: false, user: null };
      realtimeClient.disconnect();
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
        logo_updated_at: settings?.logo_updated_at || '',
        logo_upload_available: Boolean(settings?.logo_upload_available),
        logo_upload_unavailable_reason: settings?.logo_upload_unavailable_reason || ''
      };
    } catch {
      siteSettings = defaultSiteSettings();
    }
  }

  function canAccessCurrentPath() {
    return canAccessAppRoute(path, auth.user);
  }

  onMount(() => {
    syncRoute();
    refreshAuth();
    refreshSiteSettings();
    window.addEventListener('popstate', syncRoute);

    return () => {
      window.removeEventListener('popstate', syncRoute);
      realtimeClient.disconnect();
    };
  });
</script>

<div class="app-shell">
  <Header {auth} {siteSettings} onAuthChanged={handleAuthChanged} />

  {#if auth.logged_in}
    <NotificationCenter {notifications} />
    <TaskCenter refreshTrigger={taskRefreshTrigger} />
  {/if}

  {#if isAuthRoute(path)}
    <main class="page-wrap py-8">
      {#if path === '/app/login'}
        <Login onSuccess={handleAuthChanged} />
      {:else if path === '/app/login/oauth/callback'}
        <OAuthCallback onSuccess={handleAuthChanged} />
      {:else if path === '/app/register'}
        <Register onSuccess={handleAuthChanged} />
      {:else if path === '/app/forgot-password'}
        <ForgotPassword />
      {:else if path === '/app/reset-password'}
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
    <AppSidebar path={canAccessCurrentPath() ? path : appHomePath} {auth}>
      {#snippet children()}
        {#if !canAccessCurrentPath()}
          <DashboardHome {auth} />
        {:else if path === '/app/orders'}
          <Dashboard {auth} />
        {:else if path === '/app/products'}
          <Products />
        {:else if path === '/app/users'}
          <Users {auth} />
        {:else if path === '/app/scheduler'}
          <Scheduler />
        {:else if path === '/app/events'}
          <Events />
        {:else if path === '/app/experiments'}
          <Experiments {auth} />
        {:else if path === '/app/dictionary'}
          <Dictionary />
        {:else if path === '/app/parameters'}
          <Parameters />
        {:else if path === '/app/notifications'}
          <Notifications />
        {:else if path === '/app/settings'}
          <Settings settings={siteSettings} onSettingsChanged={refreshSiteSettings} />
        {:else if path === '/app/variables'}
          <Variables />
        {:else if path === '/app/checkout'}
          <AppCheckout {auth} />
        {:else}
          <DashboardHome {auth} />
        {/if}
      {/snippet}
    </AppSidebar>
  {/if}
</div>

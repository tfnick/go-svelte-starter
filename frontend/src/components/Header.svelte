<script>
  import { logout } from '../api.js';
  import { appHomePath, navigate } from '../router.js';

  let { auth, siteSettings, onAuthChanged } = $props();
  let busy = $state(false);
  let error = $state('');
  let logoURL = $state('/logo.png');

  $effect(() => {
    logoURL = siteSettings?.logo_url || '/logo.png';
  });

  async function handleLogout() {
    busy = true;
    error = '';
    try {
      await logout();
      onAuthChanged?.();
      navigate(appHomePath);
    } catch (err) {
      error = err.message || '登出失败';
    } finally {
      busy = false;
    }
  }

</script>

<header class="border-b border-base-200 bg-base-100/90 shadow-sm">
  <div class="page-wrap navbar px-0">
    <div class="navbar-start">
      <button class="btn btn-ghost px-2 text-lg font-bold" type="button" onclick={() => navigate(appHomePath)}>
        <img
          alt="Svelte Go Starter"
          class="h-[25px] w-[110px] object-contain"
          height="25"
          src={logoURL}
          width="110"
          onerror={() => {
            if (logoURL !== '/logo.png') {
              logoURL = '/logo.png';
            }
          }}
        />
      </button>
    </div>

    <div class="navbar-end gap-2">
      {#if auth.loading}
        <span class="loading loading-spinner loading-sm" aria-label="加载认证状态"></span>
      {:else if auth.logged_in}
        <span class="hidden text-sm text-base-content/70 sm:inline">你好，{auth.user?.name}</span>
        <button class="btn btn-outline btn-sm" type="button" onclick={handleLogout} disabled={busy}>
          {#if busy}
            <span class="loading loading-spinner loading-xs"></span>
          {/if}
          登出
        </button>
      {:else}
        <button class="btn btn-ghost btn-sm" type="button" onclick={() => navigate('/app/login')}>登录</button>
        <button class="btn btn-primary btn-sm" type="button" onclick={() => navigate('/app/register')}>注册</button>
      {/if}
    </div>
  </div>

  {#if error}
    <div class="page-wrap pb-3">
      <div class="alert alert-error py-2 text-sm">{error}</div>
    </div>
  {/if}
</header>

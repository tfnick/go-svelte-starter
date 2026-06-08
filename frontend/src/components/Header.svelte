<script>
  import { logout } from '../api.js';
  import { navigate } from '../router.js';

  let { auth, onAuthChanged } = $props();
  let busy = $state(false);
  let error = $state('');

  async function handleLogout() {
    busy = true;
    error = '';
    try {
      await logout();
      onAuthChanged?.();
      navigate('/');
    } catch (err) {
      error = err.message || '登出失败';
    } finally {
      busy = false;
    }
  }

</script>

<header class="border-b border-base-300 bg-base-100/90 shadow-sm">
  <div class="page-wrap navbar px-0">
    <div class="navbar-start">
      <button class="btn btn-ghost px-2 text-lg font-bold" type="button" onclick={() => navigate('/')}>
        Svelte Go Starter
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
        <button class="btn btn-ghost btn-sm" type="button" onclick={() => navigate('/login')}>登录</button>
        <button class="btn btn-primary btn-sm" type="button" onclick={() => navigate('/register')}>注册</button>
      {/if}
    </div>
  </div>

  {#if error}
    <div class="page-wrap pb-3">
      <div class="alert alert-error py-2 text-sm">{error}</div>
    </div>
  {/if}
</header>

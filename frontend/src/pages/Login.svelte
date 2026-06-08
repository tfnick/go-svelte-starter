<script>
  import { login, startOAuthLogin } from '../api.js';
  import AuthCard from '../components/AuthCard.svelte';
  import Notice from '../components/Notice.svelte';
  import { navigate } from '../router.js';
  import { onMount } from 'svelte';

  let { onSuccess } = $props();
  let email = $state('');
  let password = $state('');
  let busy = $state(false);
  let error = $state('');

  async function submit() {
    busy = true;
    error = '';
    try {
      await login({ email, password });
      onSuccess?.();
      navigate('/');
    } catch (err) {
      error = err.message || '登录失败';
    } finally {
      busy = false;
    }
  }

  function redirectPath() {
    const pathname = globalThis.location?.pathname || '/';
    if (['/login', '/register', '/forgot-password', '/reset-password', '/login/oauth/callback'].includes(pathname)) {
      return '/';
    }
    const search = globalThis.location?.search || '';
    return `${pathname}${search}`;
  }

  function continueWith(provider) {
    error = '';
    startOAuthLogin(provider, redirectPath());
  }

  onMount(() => {
    const oauthError = new URLSearchParams(globalThis.location?.search || '').get('oauth_error');
    if (oauthError) {
      error = oauthError;
    }
  });
</script>

<AuthCard title="用户登录" subtitle="使用你的邮箱和密码进入控制台。">
  <div class="grid gap-2 sm:grid-cols-2">
    <button class="btn btn-outline w-full" type="button" onclick={() => continueWith('google')}>
      Google
    </button>
    <button class="btn btn-outline w-full" type="button" onclick={() => continueWith('github')}>
      GitHub
    </button>
  </div>

  <div class="divider my-1">or</div>

  <form class="space-y-4" onsubmit={(event) => { event.preventDefault(); submit(); }}>
    <label class="form-control">
      <span class="label">
        <span class="label-text">邮箱</span>
      </span>
      <input class="input input-bordered" type="email" bind:value={email} placeholder="your@email.com" required />
    </label>

    <label class="form-control">
      <span class="label">
        <span class="label-text">密码</span>
      </span>
      <input class="input input-bordered" type="password" bind:value={password} placeholder="至少 6 位" required />
    </label>

    <Notice type="error" message={error} />

    <button class="btn btn-primary w-full" type="submit" disabled={busy}>
      {#if busy}
        <span class="loading loading-spinner loading-sm"></span>
      {/if}
      登录
    </button>
  </form>

  <div class="flex items-center justify-between text-sm">
    <button class="link link-hover" type="button" onclick={() => navigate('/register')}>注册账号</button>
    <button class="link link-hover" type="button" onclick={() => navigate('/forgot-password')}>忘记密码？</button>
  </div>
</AuthCard>

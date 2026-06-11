<script>
  import { register } from '../api.js';
  import AuthCard from '../components/AuthCard.svelte';
  import Notice from '../components/Notice.svelte';
  import { appHomePath, isAuthRoute, navigate, normalizeRouteTarget } from '../router.js';

  let { onSuccess } = $props();
  let name = $state('');
  let email = $state('');
  let password = $state('');
  let busy = $state(false);
  let error = $state('');

  async function submit() {
    busy = true;
    error = '';
    try {
      await register({ name, email, password });
      onSuccess?.();
      navigate(redirectPath());
    } catch (err) {
      error = err.message || '注册失败';
    } finally {
      busy = false;
    }
  }

  function redirectPath() {
    const pathname = globalThis.location?.pathname || '/';
    const search = globalThis.location?.search || '';
    const params = new URLSearchParams(search);
    const explicitRedirect = safeRedirectPath(params.get('redirect_path') || params.get('redirect') || '');
    if (explicitRedirect) {
      return explicitRedirect;
    }

    if (isAuthRoute(pathname)) {
      return appHomePath;
    }
    return `${pathname}${search}`;
  }

  function safeRedirectPath(value) {
    if (!value || !value.startsWith('/') || value.startsWith('//')) {
      return '';
    }
    return isAuthRoute(value) ? appHomePath : normalizeRouteTarget(value);
  }
</script>

<AuthCard title="注册账号" subtitle="创建账号后会自动登录。">
  <form class="space-y-4" onsubmit={(event) => { event.preventDefault(); submit(); }}>
    <fieldset class="fieldset">
          <legend class="fieldset-legend">姓名</legend>
      <input class="input w-full" type="text" bind:value={name} placeholder="你的姓名" required />
        </fieldset>

    <fieldset class="fieldset">
          <legend class="fieldset-legend">邮箱</legend>
      <input class="input w-full" type="email" bind:value={email} placeholder="your@email.com" required />
        </fieldset>

    <fieldset class="fieldset">
          <legend class="fieldset-legend">密码</legend>
      <input class="input w-full" type="password" bind:value={password} placeholder="至少 6 位" required />
        </fieldset>

    <Notice type="error" message={error} />

    <button class="btn btn-primary w-full" type="submit" disabled={busy}>
      {#if busy}
        <span class="loading loading-spinner loading-sm"></span>
      {/if}
      注册
    </button>
  </form>

  <button class="link link-hover self-start text-sm" type="button" onclick={() => navigate('/app/login')}>
    已有账号？登录
  </button>
</AuthCard>

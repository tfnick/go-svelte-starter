<script>
  import { login } from '../api.js';
  import AuthCard from '../components/AuthCard.svelte';
  import Notice from '../components/Notice.svelte';
  import { navigate } from '../router.js';

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
</script>

<AuthCard title="用户登录" subtitle="使用你的邮箱和密码进入控制台。">
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

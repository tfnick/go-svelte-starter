<script>
  import { resetPassword } from '../api.js';
  import AuthCard from '../components/AuthCard.svelte';
  import Notice from '../components/Notice.svelte';
  import { navigate } from '../router.js';

  const params = new URLSearchParams(window.location.search);
  let token = $state(params.get('token') || '');
  let password = $state('');
  let busy = $state(false);
  let error = $state('');
  let message = $state('');

  async function submit() {
    busy = true;
    error = '';
    message = '';
    try {
      const result = await resetPassword({ token, password });
      message = result.message || '密码重置成功，请重新登录。';
      setTimeout(() => navigate('/app/login'), 900);
    } catch (err) {
      error = err.message || '重置失败';
    } finally {
      busy = false;
    }
  }
</script>

<AuthCard title="重置密码" subtitle="输入新密码后即可重新登录。">
  <form class="space-y-4" onsubmit={(event) => { event.preventDefault(); submit(); }}>
    <label class="form-control">
      <span class="label">
        <span class="label-text">重置 Token</span>
      </span>
      <input class="input input-bordered" type="text" bind:value={token} placeholder="来自重置链接的 token" required />
    </label>

    <label class="form-control">
      <span class="label">
        <span class="label-text">新密码</span>
      </span>
      <input class="input input-bordered" type="password" bind:value={password} placeholder="至少 6 位" required />
    </label>

    <Notice type="success" message={message} />
    <Notice type="error" message={error} />

    <button class="btn btn-primary w-full" type="submit" disabled={busy}>
      {#if busy}
        <span class="loading loading-spinner loading-sm"></span>
      {/if}
      确认重置
    </button>
  </form>
</AuthCard>

<script>
  import { forgotPassword } from '../api.js';
  import AuthCard from '../components/AuthCard.svelte';
  import Notice from '../components/Notice.svelte';
  import { navigate } from '../router.js';

  let email = $state('');
  let busy = $state(false);
  let error = $state('');
  let message = $state('');

  async function submit() {
    busy = true;
    error = '';
    message = '';
    try {
      const result = await forgotPassword({ email });
      message = result.message || '如果该邮箱已注册，重置链接已发送。';
    } catch (err) {
      error = err.message || '发送失败';
    } finally {
      busy = false;
    }
  }
</script>

<AuthCard title="找回密码" subtitle="输入注册邮箱。开发环境会在后端日志里打印重置链接。">
  <form class="space-y-4" onsubmit={(event) => { event.preventDefault(); submit(); }}>
    <label class="form-control">
      <span class="label">
        <span class="label-text">邮箱</span>
      </span>
      <input class="input input-bordered" type="email" bind:value={email} placeholder="your@email.com" required />
    </label>

    <Notice type="success" message={message} />
    <Notice type="error" message={error} />

    <button class="btn btn-primary w-full" type="submit" disabled={busy}>
      {#if busy}
        <span class="loading loading-spinner loading-sm"></span>
      {/if}
      发送重置链接
    </button>
  </form>

  <button class="link link-hover self-start text-sm" type="button" onclick={() => navigate('/app/login')}>
    返回登录
  </button>
</AuthCard>

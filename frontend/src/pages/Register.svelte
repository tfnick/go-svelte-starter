<script>
  import { register } from '../api.js';
  import AuthCard from '../components/AuthCard.svelte';
  import Notice from '../components/Notice.svelte';
  import { navigate } from '../router.js';

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
      navigate('/');
    } catch (err) {
      error = err.message || '注册失败';
    } finally {
      busy = false;
    }
  }
</script>

<AuthCard title="注册账号" subtitle="创建账号后会自动登录。">
  <form class="space-y-4" onsubmit={(event) => { event.preventDefault(); submit(); }}>
    <label class="form-control">
      <span class="label">
        <span class="label-text">姓名</span>
      </span>
      <input class="input input-bordered" type="text" bind:value={name} placeholder="你的姓名" required />
    </label>

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
      注册
    </button>
  </form>

  <button class="link link-hover self-start text-sm" type="button" onclick={() => navigate('/login')}>
    已有账号？登录
  </button>
</AuthCard>

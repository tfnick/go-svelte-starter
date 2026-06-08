<script>
  import { onMount } from 'svelte';
  import { exchangeOAuthLoginResult } from '../api.js';
  import AuthCard from '../components/AuthCard.svelte';
  import Notice from '../components/Notice.svelte';
  import { navigate } from '../router.js';

  let { onSuccess } = $props();
  let error = $state('');
  let busy = $state(true);

  function safeRedirectPath(value) {
    if (!value || !value.startsWith('/') || value.startsWith('//')) {
      return '/';
    }
    return value === '/login/oauth/callback' ? '/' : value;
  }

  onMount(async () => {
    const params = new URLSearchParams(globalThis.location?.search || '');
    const token = params.get('token') || '';
    const redirectPath = safeRedirectPath(params.get('redirect_path') || '/');
    if (!token) {
      error = 'OAuth login token is missing.';
      busy = false;
      return;
    }

    try {
      await exchangeOAuthLoginResult(token);
      onSuccess?.();
      navigate(redirectPath);
    } catch (err) {
      error = err.message || 'OAuth login failed.';
      busy = false;
    }
  });
</script>

<AuthCard title="OAuth Login" subtitle="Completing your login.">
  {#if busy}
    <div class="flex min-h-24 items-center justify-center">
      <span class="loading loading-spinner loading-md" aria-label="Loading"></span>
    </div>
  {/if}

  <Notice type="error" message={error} />

  {#if error}
    <button class="btn btn-primary w-full" type="button" onclick={() => navigate('/login')}>
      Back to login
    </button>
  {/if}
</AuthCard>

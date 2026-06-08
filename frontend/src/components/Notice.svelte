<script>
  let { type = 'info', message = '' } = $props();

  const classes = {
    info: 'alert-info',
    success: 'alert-success',
    error: 'alert-error',
    warning: 'alert-warning'
  };
</script>

{#if message}
  <div class={`alert ${classes[type] || classes.info} text-sm`}>
    <span>{message}</span>
  </div>
{/if}

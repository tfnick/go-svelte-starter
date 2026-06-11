<script>
  let { title, subtitle = '', children } = $props();
</script>

<section class="flex justify-center px-4 py-10">
  <div class="auth-panel card border border-base-200 bg-base-100 shadow-xl">
    <div class="card-body gap-6 p-6 sm:p-8">
      <div class="space-y-2">
        <h1 class="text-2xl font-semibold tracking-normal">{title}</h1>
        {#if subtitle}
          <p class="text-sm leading-6 text-base-content/65">{subtitle}</p>
        {/if}
      </div>

      {@render children?.()}
    </div>
  </div>
</section>

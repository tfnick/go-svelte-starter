<script>
  let { title, subtitle = '', children } = $props();
</script>

<section class="flex justify-center py-8">
  <div class="auth-panel card border border-base-300 bg-base-100 shadow-xl">
    <div class="card-body gap-5">
      <div>
        <h1 class="card-title text-2xl">{title}</h1>
        {#if subtitle}
          <p class="mt-2 text-sm leading-6 text-base-content/70">{subtitle}</p>
        {/if}
      </div>

      {@render children?.()}
    </div>
  </div>
</section>

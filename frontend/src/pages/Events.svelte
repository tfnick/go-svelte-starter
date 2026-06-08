<script>
  import { onMount } from 'svelte';

  import { listEventDeliveries, listEvents } from '../api.js';
  import Notice from '../components/Notice.svelte';
  import { formatLocalDateTime } from '../helpers/dateTime.js';

  const eventPageSize = 10;
  const emptyEventPagination = {
    page: 1,
    page_size: eventPageSize,
    total_items: 0,
    total_pages: 0,
    has_previous: false,
    has_next: false
  };

  let events = $state([]);
  let deliveries = $state([]);
  let eventPagination = $state({ ...emptyEventPagination });
  let selectedEventId = $state('');
  let loadingEvents = $state(false);
  let loadingDeliveries = $state(false);
  let error = $state('');

  onMount(() => {
    loadEvents(1);
  });

  async function loadEvents(page = eventPagination.page) {
    loadingEvents = true;
    error = '';
    try {
      const result = await listEvents({
        page,
        pageSize: eventPageSize
      });
      events = Array.isArray(result?.items) ? result.items : [];
      eventPagination = {
        ...emptyEventPagination,
        ...(result?.pagination || {})
      };

      if (selectedEventId && events.some((event) => event.id === selectedEventId)) {
        return;
      }

      deliveries = [];
      selectedEventId = '';
      if (events.length > 0) {
        await selectEvent(events[0]);
      }
    } catch (err) {
      error = err.message || 'Failed to load events';
    } finally {
      loadingEvents = false;
    }
  }

  async function selectEvent(event) {
    if (!event?.id) return;

    selectedEventId = event.id;
    loadingDeliveries = true;
    error = '';
    try {
      deliveries = await listEventDeliveries(event.id);
    } catch (err) {
      error = err.message || 'Failed to load event deliveries';
    } finally {
      loadingDeliveries = false;
    }
  }

  async function goToEventsPage(page) {
    if (loadingEvents || page < 1 || page === eventPagination.page) return;
    const totalPages = Number(eventPagination.total_pages || 0);
    if (totalPages > 0 && page > totalPages) return;
    await loadEvents(page);
  }

  function visibleEventPages() {
    const totalPages = Number(eventPagination.total_pages || 0);
    const currentPage = Number(eventPagination.page || 1);
    if (totalPages <= 1) return [];

    const maxButtons = 5;
    const pages = [];
    let start = Math.max(1, currentPage - Math.floor(maxButtons / 2));
    let end = Math.min(totalPages, start + maxButtons - 1);
    start = Math.max(1, end - maxButtons + 1);

    for (let page = start; page <= end; page += 1) {
      pages.push(page);
    }
    return pages;
  }

  function selectedEvent() {
    return events.find((event) => event.id === selectedEventId);
  }

  function formatDate(value) {
    return formatLocalDateTime(value);
  }

  function statusClass(status) {
    if (status === 'succeeded') return 'badge-success';
    if (status === 'failed') return 'badge-error';
    if (status === 'running') return 'badge-warning';
    return 'badge-neutral';
  }

  function compactJSON(value, fallback = '--') {
    if (!value) return fallback;
    try {
      return JSON.stringify(JSON.parse(value));
    } catch {
      return value;
    }
  }

  function prettyJSON(value) {
    if (!value) return '--';
    try {
      return JSON.stringify(JSON.parse(value), null, 2);
    } catch {
      return value;
    }
  }

  function truncate(value, maxLength = 96) {
    const text = String(value || '');
    if (text.length <= maxLength) return text;
    return `${text.slice(0, maxLength)}...`;
  }
</script>

<section class="space-y-6">
  <div class="flex flex-wrap items-start justify-between gap-4">
    <div>
      <h1 class="text-2xl font-bold leading-tight">Event</h1>
      <p class="mt-1 text-sm text-base-content/60">Domain event facts and per-event delivery records.</p>
    </div>
    <button class="btn btn-outline btn-sm" type="button" onclick={() => loadEvents(eventPagination.page)} disabled={loadingEvents}>
      {#if loadingEvents}
        <span class="loading loading-spinner loading-xs"></span>
      {/if}
      Refresh
    </button>
  </div>

  <Notice type="error" message={error} />

  <div class="grid gap-6 xl:grid-cols-[1.12fr_0.88fr]">
    <div class="card border border-base-300 bg-base-100 shadow-sm">
      <div class="card-body gap-4">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <h2 class="card-title text-lg">Events</h2>
          <span class="badge badge-outline">{eventPagination.total_items}</span>
        </div>

        {#if events.length === 0}
          <div class="rounded border border-dashed border-base-300 p-6 text-center text-sm text-base-content/60">
            {loadingEvents ? 'Loading events...' : 'No events'}
          </div>
        {:else}
          <div class="overflow-x-auto">
            <table class="table table-sm">
              <thead>
                <tr>
                  <th>Topic</th>
                  <th>Aggregate</th>
                  <th>Occurred</th>
                  <th>Payload</th>
                  <th class="text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                {#each events as event}
                  <tr class:selected={selectedEventId === event.id}>
                    <td>
                      <div class="font-medium">{event.topic}</div>
                      <div class="max-w-48 truncate font-mono text-xs text-base-content/50">{event.id}</div>
                    </td>
                    <td>
                      <div class="font-mono text-xs">{event.aggregate_type || '--'}</div>
                      <div class="max-w-44 truncate font-mono text-xs text-base-content/60">{event.aggregate_id || '--'}</div>
                    </td>
                    <td class="whitespace-nowrap text-xs">{formatDate(event.occurred_at || event.created_at)}</td>
                    <td class="max-w-72 truncate font-mono text-xs">{truncate(compactJSON(event.payload_json))}</td>
                    <td class="text-right">
                      <button class="btn btn-xs" type="button" onclick={() => selectEvent(event)} disabled={loadingDeliveries && selectedEventId === event.id}>
                        {#if loadingDeliveries && selectedEventId === event.id}
                          <span class="loading loading-spinner loading-xs"></span>
                        {/if}
                        Deliveries
                      </button>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>

          <div class="flex flex-col gap-3 border-t border-base-300 pt-4 sm:flex-row sm:items-center sm:justify-between">
            <div class="text-sm text-base-content/60">
              {eventPagination.total_items} events - Page {eventPagination.page} / {Math.max(eventPagination.total_pages, 1)}
            </div>
            {#if eventPagination.total_pages > 1}
              <div class="max-w-full overflow-x-auto">
                <div class="join">
                  <button
                    class="btn join-item btn-sm"
                    type="button"
                    onclick={() => goToEventsPage(eventPagination.page - 1)}
                    disabled={loadingEvents || !eventPagination.has_previous}
                  >
                    Prev
                  </button>
                  {#each visibleEventPages() as page}
                    <button
                      class="btn join-item btn-sm {page === eventPagination.page ? 'btn-active' : ''}"
                      type="button"
                      onclick={() => goToEventsPage(page)}
                      disabled={loadingEvents}
                    >
                      {page}
                    </button>
                  {/each}
                  <button
                    class="btn join-item btn-sm"
                    type="button"
                    onclick={() => goToEventsPage(eventPagination.page + 1)}
                    disabled={loadingEvents || !eventPagination.has_next}
                  >
                    Next
                  </button>
                </div>
              </div>
            {/if}
          </div>
        {/if}
      </div>
    </div>

    <div class="space-y-6">
      <div class="card border border-base-300 bg-base-100 shadow-sm">
        <div class="card-body gap-4">
          <div class="flex items-center justify-between gap-3">
            <h2 class="card-title text-lg">Selected event</h2>
            {#if loadingDeliveries}
              <span class="loading loading-spinner loading-sm"></span>
            {/if}
          </div>

          {#if selectedEvent()}
            <div class="space-y-3 text-sm">
              <div>
                <div class="text-xs font-semibold uppercase text-base-content/50">ID</div>
                <div class="break-all font-mono text-xs">{selectedEvent().id}</div>
              </div>
              <div class="grid gap-3 sm:grid-cols-2">
                <div>
                  <div class="text-xs font-semibold uppercase text-base-content/50">Topic</div>
                  <div>{selectedEvent().topic}</div>
                </div>
                <div>
                  <div class="text-xs font-semibold uppercase text-base-content/50">Occurred</div>
                  <div>{formatDate(selectedEvent().occurred_at || selectedEvent().created_at)}</div>
                </div>
              </div>
              <div>
                <div class="text-xs font-semibold uppercase text-base-content/50">Payload</div>
                <pre class="mt-2 max-h-44 overflow-auto rounded bg-base-200 p-3 font-mono text-xs whitespace-pre-wrap break-all">{prettyJSON(selectedEvent().payload_json)}</pre>
              </div>
              <div>
                <div class="text-xs font-semibold uppercase text-base-content/50">Metadata</div>
                <pre class="mt-2 max-h-32 overflow-auto rounded bg-base-200 p-3 font-mono text-xs whitespace-pre-wrap break-all">{prettyJSON(selectedEvent().metadata_json)}</pre>
              </div>
            </div>
          {:else}
            <div class="rounded border border-dashed border-base-300 p-6 text-center text-sm text-base-content/60">
              Select an event to view details
            </div>
          {/if}
        </div>
      </div>

      <div class="card border border-base-300 bg-base-100 shadow-sm">
        <div class="card-body gap-4">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <h2 class="card-title text-lg">Delivery records</h2>
            <span class="badge badge-outline">{deliveries.length}</span>
          </div>

          {#if deliveries.length === 0}
            <div class="rounded border border-dashed border-base-300 p-6 text-center text-sm text-base-content/60">
              {selectedEventId ? 'No delivery records' : 'Select an event to view deliveries'}
            </div>
          {:else}
            <div class="overflow-x-auto">
              <table class="table table-sm">
                <thead>
                  <tr>
                    <th>Subscriber</th>
                    <th>Status</th>
                    <th class="text-right">Attempts</th>
                    <th>Updated</th>
                  </tr>
                </thead>
                <tbody>
                  {#each deliveries as delivery}
                    <tr>
                      <td>
                        <div class="max-w-56 truncate font-mono text-xs">{delivery.subscriber}</div>
                        <div class="max-w-56 truncate font-mono text-xs text-base-content/50">{delivery.message_id || '--'}</div>
                      </td>
                      <td><span class="badge {statusClass(delivery.status)}">{delivery.status}</span></td>
                      <td class="text-right">{delivery.attempts}</td>
                      <td class="whitespace-nowrap text-xs">{formatDate(delivery.updated_at || delivery.created_at)}</td>
                    </tr>
                    {#if delivery.last_error}
                      <tr>
                        <td colspan="4" class="text-error">
                          <div class="max-w-full break-words font-mono text-xs">{delivery.last_error}</div>
                        </td>
                      </tr>
                    {/if}
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        </div>
      </div>
    </div>
  </div>
</section>

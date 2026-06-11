<script>
  import { onMount } from 'svelte';

  import {
    createScheduledTask,
    listMessages,
    listScheduledTaskHistory,
    listScheduledTasks,
    setScheduledTaskEnabled,
    updateScheduledTask
  } from '../api.js';
  import Notice from '../components/Notice.svelte';
  import { formatLocalDateTime } from '../helpers/dateTime.js';

  const emptyForm = {
    id: '',
    name: '',
    job_name: 'scheduler.noop',
    schedule_type: 'cron',
    schedule_value: '*/5 * * * *',
    payload_json: '{}',
    enabled: true
  };

  let tasks = $state([]);
  let history = $state([]);
  let messages = $state([]);
  let queueFilter = $state('');
  let selectedTaskId = $state('');
  let form = $state({ ...emptyForm });
  let loadingTasks = $state(false);
  let loadingHistory = $state(false);
  let loadingMessages = $state(false);
  let saving = $state(false);
  let error = $state('');
  let message = $state('');

  onMount(() => {
    loadPage();
  });

  async function loadPage() {
    await Promise.all([loadTasks(), loadMessagesTable()]);
  }

  async function loadTasks() {
    loadingTasks = true;
    error = '';
    try {
      tasks = await listScheduledTasks();
      if (selectedTaskId && !tasks.some((task) => task.id === selectedTaskId)) {
        selectedTaskId = '';
        history = [];
      }
    } catch (err) {
      error = err.message || 'Failed to load scheduled tasks';
    } finally {
      loadingTasks = false;
    }
  }

  async function loadMessagesTable() {
    loadingMessages = true;
    error = '';
    try {
      messages = await listMessages(queueFilter.trim());
    } catch (err) {
      error = err.message || 'Failed to load queue messages';
    } finally {
      loadingMessages = false;
    }
  }

  async function loadHistory(task) {
    if (!task?.id) return;

    selectedTaskId = task.id;
    loadingHistory = true;
    error = '';
    try {
      history = await listScheduledTaskHistory(task.id);
    } catch (err) {
      error = err.message || 'Failed to load task history';
    } finally {
      loadingHistory = false;
    }
  }

  function editTask(task) {
    form = {
      id: task.id,
      name: task.name,
      job_name: task.job_name,
      schedule_type: task.schedule_type,
      schedule_value: task.schedule_value,
      payload_json: task.payload_json || '{}',
      enabled: Boolean(task.enabled)
    };
  }

  function resetForm() {
    form = { ...emptyForm };
  }

  async function saveTask() {
    saving = true;
    error = '';
    message = '';
    const payload = {
      name: form.name,
      job_name: form.job_name,
      schedule_type: form.schedule_type,
      schedule_value: form.schedule_value,
      payload_json: form.payload_json,
      enabled: form.enabled
    };

    try {
      const saved = form.id
        ? await updateScheduledTask(form.id, payload)
        : await createScheduledTask(payload);
      message = form.id ? 'Scheduled task updated' : 'Scheduled task created';
      form = {
        id: saved.id,
        name: saved.name,
        job_name: saved.job_name,
        schedule_type: saved.schedule_type,
        schedule_value: saved.schedule_value,
        payload_json: saved.payload_json || '{}',
        enabled: Boolean(saved.enabled)
      };
      await loadTasks();
    } catch (err) {
      error = err.message || 'Failed to save scheduled task';
    } finally {
      saving = false;
    }
  }

  async function toggleTask(task) {
    error = '';
    message = '';
    try {
      await setScheduledTaskEnabled(task.id, !task.enabled);
      message = task.enabled ? 'Scheduled task disabled' : 'Scheduled task enabled';
      await loadTasks();
    } catch (err) {
      error = err.message || 'Failed to update scheduled task';
    }
  }

  function selectedTaskName() {
    return tasks.find((task) => task.id === selectedTaskId)?.name || 'Execution history';
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
</script>

<section class="space-y-6">
  <div class="flex flex-wrap items-start justify-between gap-4">
    <div>
      <h1 class="text-2xl font-bold leading-tight">Scheduler</h1>
      <p class="mt-1 text-sm text-base-content/60">Scheduled task definitions, execution history, and durable queue messages.</p>
    </div>
    <div class="flex gap-2">
      <button class="btn btn-outline btn-sm" type="button" onclick={loadPage} disabled={loadingTasks || loadingMessages}>
        {#if loadingTasks || loadingMessages}
          <span class="loading loading-spinner loading-xs"></span>
        {/if}
        Refresh
      </button>
      <button class="btn btn-primary btn-sm" type="button" onclick={resetForm}>New task</button>
    </div>
  </div>

  <Notice type="success" message={message} />
  <Notice type="error" message={error} />

  <div class="grid gap-6 xl:grid-cols-[0.84fr_1.16fr]">
    <div class="card min-w-0 border border-base-200 bg-base-100 shadow-sm">
      <div class="card-body gap-4 p-5">
        <div class="flex items-center justify-between gap-3">
          <h2 class="card-title text-lg">{form.id ? 'Edit task' : 'Create task'}</h2>
          {#if form.id}
            <span class="badge badge-outline max-w-48 truncate font-mono text-xs">{form.id}</span>
          {/if}
        </div>

        <fieldset class="fieldset">
          <legend class="fieldset-legend">Name</legend>
          <input class="input w-full" bind:value={form.name} placeholder="Nightly export" />
        </fieldset>

        <fieldset class="fieldset">
          <legend class="fieldset-legend">Job name</legend>
          <input class="input font-mono text-sm w-full" bind:value={form.job_name} />
        </fieldset>

        <div class="grid gap-3 sm:grid-cols-[0.42fr_0.58fr]">
          <fieldset class="fieldset">
          <legend class="fieldset-legend">Schedule type</legend>
            <select class="select w-full" bind:value={form.schedule_type}>
              <option value="cron">cron</option>
              <option value="once_at">once_at</option>
            </select>
        </fieldset>

          <fieldset class="fieldset">
          <legend class="fieldset-legend">Schedule value</legend>
            <input class="input font-mono text-sm w-full" bind:value={form.schedule_value} />
        </fieldset>
        </div>

        <fieldset class="fieldset">
          <legend class="fieldset-legend">Payload JSON</legend>
          <textarea class="textarea min-h-28 font-mono text-sm w-full" bind:value={form.payload_json}></textarea>
        </fieldset>

        <label class="fieldset-label cursor-pointer justify-start gap-3 rounded-box border border-base-200 bg-base-200/40 px-3 py-3">
          <input class="toggle toggle-primary" type="checkbox" bind:checked={form.enabled} />
          <span>Enabled</span>
        </label>

        <button class="btn btn-primary" type="button" onclick={saveTask} disabled={saving}>
          {#if saving}
            <span class="loading loading-spinner loading-sm"></span>
          {/if}
          Save task
        </button>
      </div>
    </div>

    <div class="card min-w-0 border border-base-200 bg-base-100 shadow-sm">
      <div class="card-body gap-4 p-5">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <h2 class="card-title text-lg">Task definitions</h2>
          <span class="badge badge-outline">{tasks.length}</span>
        </div>

        {#if tasks.length === 0}
          <div class="rounded-box border border-dashed border-base-200 p-6 text-center text-sm text-base-content/60">
            {loadingTasks ? 'Loading scheduled tasks...' : 'No scheduled tasks'}
          </div>
        {:else}
          <div class="max-w-full overflow-x-auto rounded-box border border-base-200">
            <table class="table table-zebra table-sm min-w-[44rem]">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Schedule</th>
                  <th>Next run</th>
                  <th>Status</th>
                  <th class="text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                {#each tasks as task}
                  <tr class={selectedTaskId === task.id ? 'bg-primary/5' : ''}>
                    <td>
                      <div class="font-medium">{task.name}</div>
                      <div class="max-w-56 truncate font-mono text-xs text-base-content/50">{task.job_name}</div>
                    </td>
                    <td>
                      <div class="font-mono text-xs">{task.schedule_type}</div>
                      <div class="max-w-48 truncate font-mono text-xs text-base-content/60">{task.schedule_value}</div>
                    </td>
                    <td>{formatDate(task.next_run_at)}</td>
                    <td>
                      <span class="badge {task.enabled ? 'badge-success' : 'badge-outline'}">
                        {task.enabled ? 'enabled' : 'disabled'}
                      </span>
                    </td>
                    <td class="text-right">
                      <div class="join">
                        <button class="btn join-item btn-xs" type="button" onclick={() => editTask(task)}>Edit</button>
                        <button class="btn join-item btn-xs" type="button" onclick={() => loadHistory(task)}>History</button>
                        <button class="btn join-item btn-xs" type="button" onclick={() => toggleTask(task)}>
                          {task.enabled ? 'Disable' : 'Enable'}
                        </button>
                      </div>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>
    </div>
  </div>

  <div class="grid gap-6 xl:grid-cols-2">
    <div class="card min-w-0 border border-base-200 bg-base-100 shadow-sm">
      <div class="card-body gap-4 p-5">
        <div class="flex items-center justify-between gap-3">
          <h2 class="card-title text-lg">{selectedTaskName()}</h2>
          {#if loadingHistory}
            <span class="loading loading-spinner loading-sm"></span>
          {/if}
        </div>

        {#if history.length === 0}
          <div class="rounded-box border border-dashed border-base-200 p-6 text-center text-sm text-base-content/60">
            {selectedTaskId ? 'No execution history' : 'Select a task to view history'}
          </div>
        {:else}
          <div class="max-w-full overflow-x-auto rounded-box border border-base-200">
            <table class="table table-zebra table-sm min-w-[44rem]">
              <thead>
                <tr>
                  <th>Status</th>
                  <th>Scheduled</th>
                  <th>Finished</th>
                  <th>Message</th>
                </tr>
              </thead>
              <tbody>
                {#each history as item}
                  <tr>
                    <td><span class="badge {statusClass(item.status)}">{item.status}</span></td>
                    <td>{formatDate(item.scheduled_at)}</td>
                    <td>{formatDate(item.finished_at)}</td>
                    <td class="max-w-40 truncate font-mono text-xs">{item.message_id || '--'}</td>
                  </tr>
                  {#if item.error_message}
                    <tr>
                      <td colspan="4" class="text-error">{item.error_message}</td>
                    </tr>
                  {/if}
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>
    </div>

    <div class="card min-w-0 border border-base-200 bg-base-100 shadow-sm">
      <div class="card-body gap-4 p-5">
        <div class="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h2 class="card-title text-lg">Queue messages</h2>
            <p class="text-xs text-base-content/60">Read-only pending, delayed, in-flight, and retryable work.</p>
          </div>
          <div class="join">
            <input class="input join-item input-sm w-44" bind:value={queueFilter} placeholder="queue name" />
            <button class="btn join-item btn-sm" type="button" onclick={loadMessagesTable} disabled={loadingMessages}>
              {#if loadingMessages}
                <span class="loading loading-spinner loading-xs"></span>
              {/if}
              Filter
            </button>
          </div>
        </div>

        {#if messages.length === 0}
          <div class="rounded-box border border-dashed border-base-200 p-6 text-center text-sm text-base-content/60">
            {loadingMessages ? 'Loading queue messages...' : 'No queue messages'}
          </div>
        {:else}
          <div class="max-w-full overflow-x-auto rounded-box border border-base-200">
            <table class="table table-zebra table-sm min-w-[44rem]">
              <thead>
                <tr>
                  <th>Queue</th>
                  <th>Message</th>
                  <th>Available</th>
                  <th class="text-right">Receive</th>
                </tr>
              </thead>
              <tbody>
                {#each messages as item}
                  <tr>
                    <td class="font-mono text-xs">{item.queue}</td>
                    <td>
                      <div class="max-w-56 truncate font-mono text-xs">{item.id}</div>
                      <div class="max-w-72 truncate text-xs text-base-content/60">{item.body_preview}</div>
                    </td>
                    <td>
                      <div class="text-xs">{formatDate(item.timeout)}</div>
                      <div class="text-xs text-base-content/50">{formatDate(item.updated)}</div>
                    </td>
                    <td class="text-right">
                      <div>{item.received}</div>
                      <div class="text-xs text-base-content/50">p{item.priority}</div>
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </div>
    </div>
  </div>
</section>

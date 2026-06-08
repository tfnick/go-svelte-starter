<script>
  import { onDestroy } from 'svelte';

  import {
    pointsSSEURL,
    summarizeTextWithLLM,
    triggerExportToast
  } from '../api.js';
  import Notice from '../components/Notice.svelte';
  import { dispatchRealtimeMessage } from '../helpers/realtimeMessages.js';

  let { auth } = $props();

  const tabs = [
    { key: 'llm', label: 'LLM' },
    { key: 'sse', label: 'SSE' }
  ];
  const defaultPrompt = 'Summarize the text in Chinese. Keep the result concise and list the most important conclusions.';

  let activeTab = $state('llm');
  let originalText = $state('');
  let requirementPrompt = $state(defaultPrompt);
  let chatMessages = $state([
    {
      id: 'welcome',
      role: 'assistant',
      text: 'Submit source text and a requirement prompt to generate a summary.',
      meta: ''
    }
  ]);
  let summarizing = $state(false);
  let llmError = $state('');
  let llmMessage = $state('');

  let streamStatus = $state('Disconnected');
  let streamEvents = $state([]);
  let sseError = $state('');
  let sseMessage = $state('');
  let triggeringExportToast = $state(false);
  let pointsStream;
  let loadedUserId = $state('');

  onDestroy(() => {
    closeSSEStream();
  });

  $effect(() => {
    const userId = auth.logged_in ? auth.user?.id || '' : '';
    if (!userId) {
      loadedUserId = '';
      streamEvents = [];
      closeSSEStream('Disconnected');
      return;
    }

    if (userId !== loadedUserId) {
      loadedUserId = userId;
      streamEvents = [];
      connectSSEStream();
    }
  });

  async function submitSummary() {
    const text = originalText.trim();
    const prompt = requirementPrompt.trim();
    if (!text) {
      llmError = 'Original text is required';
      return;
    }
    if (!prompt) {
      llmError = 'Requirement prompt is required';
      return;
    }

    const requestId = `llm-${Date.now()}-${Math.random().toString(16).slice(2)}`;
    chatMessages = [
      ...chatMessages,
      {
        id: `${requestId}-user`,
        role: 'user',
        text: `${prompt}\n\n${text}`,
        meta: 'Original text + requirement prompt'
      }
    ];

    summarizing = true;
    llmError = '';
    llmMessage = '';
    try {
      const result = await summarizeTextWithLLM({
        text,
        prompt,
        dimensions: ['summary']
      });
      const summary = result?.summary?.summary || '';
      chatMessages = [
        ...chatMessages,
        {
          id: `${requestId}-assistant`,
          role: 'assistant',
          text: summary || 'No summary returned',
          meta: metadataText(result)
        }
      ];
      llmMessage = 'Summary generated';
    } catch (err) {
      llmError = err.message || 'Failed to generate summary';
      chatMessages = [
        ...chatMessages,
        {
          id: `${requestId}-error`,
          role: 'assistant',
          text: llmError,
          meta: 'Request failed'
        }
      ];
    } finally {
      summarizing = false;
    }
  }

  function metadataText(result) {
    const parts = [];
    if (result?.channel_code) parts.push(`channel ${result.channel_code}`);
    if (result?.model_code) parts.push(`model ${result.model_code}`);
    if (result?.invocation_id) parts.push(`invocation ${result.invocation_id}`);
    return parts.join(' | ');
  }

  function clearChat() {
    chatMessages = [];
    llmError = '';
    llmMessage = '';
  }

  function connectSSEStream() {
    closeSSEStream();
    if (!auth.logged_in) {
      return;
    }

    streamStatus = 'Connecting';
    sseError = '';

    pointsStream = new EventSource(pointsSSEURL());
    pointsStream.onopen = () => {
      streamStatus = 'Connected';
      appendStreamEvent('system', 'SSE stream connected');
    };
    pointsStream.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data);
        dispatchRealtimeMessage(payload, {
          refreshPoints(nextPoints) {
            appendStreamEvent('points', `Points balance refreshed to ${nextPoints.balance ?? '--'}`);
          },
          toast(toast) {
            appendStreamEvent('toast', toast.message || 'Realtime toast received');
          }
        });
      } catch {
        appendStreamEvent('error', 'Malformed realtime message ignored');
      }
    };
    pointsStream.onerror = () => {
      streamStatus = 'Error';
      sseError = 'SSE stream disconnected or failed to connect';
    };
  }

  function closeSSEStream(nextStatus = 'Disconnected') {
    if (pointsStream) {
      pointsStream.close();
      pointsStream = undefined;
    }
    streamStatus = nextStatus;
  }

  async function handleTriggerExportToast() {
    triggeringExportToast = true;
    sseError = '';
    sseMessage = '';
    try {
      await triggerExportToast();
      sseMessage = 'Export completion event requested';
    } catch (err) {
      sseError = err.message || 'Failed to trigger export notification';
    } finally {
      triggeringExportToast = false;
    }
  }

  function appendStreamEvent(type, message) {
    streamEvents = [
      {
        id: `${Date.now()}-${Math.random().toString(16).slice(2)}`,
        type,
        message,
        time: new Date().toLocaleTimeString()
      },
      ...streamEvents
    ].slice(0, 12);
  }

  function streamStatusClass() {
    if (streamStatus === 'Connected') return 'badge-success';
    if (streamStatus === 'Error') return 'badge-error';
    return 'badge-outline';
  }
</script>

<section class="space-y-6">
  <div class="flex flex-wrap items-start justify-between gap-4">
    <div>
      <h1 class="text-2xl font-bold leading-tight">Experiment</h1>
      <p class="mt-1 text-sm text-base-content/60">Functional research demos for LLM summaries and server-sent events.</p>
    </div>
  </div>

  <div class="grid gap-6 xl:grid-cols-[0.5fr_1.1fr]">
    <div class="space-y-6">
      <div class="card border border-base-300 bg-base-100 shadow-sm">
        <div class="card-body gap-4">
          <div>
            <h2 class="card-title text-lg">Workspace</h2>
            <p class="text-sm text-base-content/60">{auth.user?.name || 'Signed in user'}</p>
          </div>

          <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-1">
            <div class="rounded border border-base-300 p-3">
              <div class="text-xs font-semibold uppercase tracking-wide text-base-content/50">LLM</div>
              <div class="mt-1 text-sm">DeepSeek summary channel</div>
            </div>
            <div class="rounded border border-base-300 p-3">
              <div class="flex items-center justify-between gap-3">
                <div>
                  <div class="text-xs font-semibold uppercase tracking-wide text-base-content/50">SSE</div>
                  <div class="mt-1 text-sm">Realtime notification stream</div>
                </div>
                <span class="badge {streamStatusClass()}">{streamStatus}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="card border border-base-300 bg-base-100 shadow-sm">
        <div class="card-body gap-4">
          <h2 class="card-title text-lg">Controls</h2>
          <button class="btn btn-outline btn-sm justify-start" type="button" onclick={() => (activeTab = 'llm')}>Open LLM</button>
          <button class="btn btn-outline btn-sm justify-start" type="button" onclick={() => (activeTab = 'sse')}>Open SSE</button>
        </div>
      </div>
    </div>

    <div class="min-w-0">
      <div class="tabs tabs-lift">
        {#each tabs as tab}
          <input
            type="radio"
            name="experiment_tabs"
            class="tab"
            aria-label={tab.label}
            checked={activeTab === tab.key}
            onchange={() => (activeTab = tab.key)}
          />
          <div class="tab-content border-base-300 bg-base-100 p-4">
            {#if tab.key === 'llm'}
              <div class="grid gap-4 lg:grid-cols-[0.85fr_1.15fr]">
                <div class="space-y-4">
                  <Notice type="success" message={llmMessage} />
                  <Notice type="error" message={llmError} />

                  <label class="form-control">
                    <span class="label"><span class="label-text">Original text</span></span>
                    <textarea
                      class="textarea textarea-bordered min-h-52 text-sm"
                      bind:value={originalText}
                      placeholder="Paste source text here"
                    ></textarea>
                  </label>

                  <label class="form-control">
                    <span class="label"><span class="label-text">Requirement prompt</span></span>
                    <textarea
                      class="textarea textarea-bordered min-h-28 text-sm"
                      bind:value={requirementPrompt}
                      placeholder="Describe the summary style and focus"
                    ></textarea>
                  </label>

                  <div class="flex flex-wrap gap-2">
                    <button class="btn btn-primary" type="button" onclick={submitSummary} disabled={summarizing}>
                      {#if summarizing}
                        <span class="loading loading-spinner loading-sm"></span>
                      {/if}
                      Submit
                    </button>
                    <button class="btn btn-outline" type="button" onclick={clearChat} disabled={summarizing}>Clear</button>
                  </div>
                </div>

                <div class="rounded border border-base-300 bg-base-200/30 p-3">
                  <div class="mb-3 flex items-center justify-between gap-3">
                    <h2 class="text-lg font-semibold">Chat</h2>
                    <span class="badge badge-ghost">{chatMessages.length} messages</span>
                  </div>
                  <div class="max-h-[34rem] min-h-96 space-y-3 overflow-y-auto pr-1">
                    {#if chatMessages.length === 0}
                      <div class="rounded border border-dashed border-base-300 p-8 text-center text-sm text-base-content/60">
                        No messages
                      </div>
                    {:else}
                      {#each chatMessages as message (message.id)}
                        <div class="chat {message.role === 'user' ? 'chat-end' : 'chat-start'}">
                          <div class="chat-header text-xs uppercase tracking-wide text-base-content/50">{message.role}</div>
                          <div class="chat-bubble max-w-[92%] whitespace-pre-wrap {message.role === 'user' ? 'chat-bubble-primary' : ''}">
                            {message.text}
                          </div>
                          {#if message.meta}
                            <div class="chat-footer max-w-[92%] truncate text-xs text-base-content/50">{message.meta}</div>
                          {/if}
                        </div>
                      {/each}
                    {/if}
                  </div>
                </div>
              </div>
            {:else}
              <div class="space-y-4">
                <div class="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <h2 class="text-lg font-semibold">Server-Sent Events</h2>
                    <p class="text-sm text-base-content/60">Simulate backend event delivery through the existing realtime stream.</p>
                  </div>
                  <span class="badge {streamStatusClass()}">{streamStatus}</span>
                </div>

                <Notice type="success" message={sseMessage} />
                <Notice type="error" message={sseError} />

                <div class="flex flex-wrap gap-2">
                  <button class="btn btn-secondary" type="button" onclick={handleTriggerExportToast} disabled={triggeringExportToast || streamStatus !== 'Connected'}>
                    {#if triggeringExportToast}
                      <span class="loading loading-spinner loading-sm"></span>
                    {/if}
                    Trigger export completed
                  </button>
                  <button class="btn btn-outline" type="button" onclick={connectSSEStream}>Reconnect</button>
                  <button class="btn btn-outline" type="button" onclick={() => (streamEvents = [])}>Clear log</button>
                </div>

                <div class="overflow-x-auto rounded border border-base-300">
                  <table class="table table-sm">
                    <thead>
                      <tr>
                        <th>Time</th>
                        <th>Type</th>
                        <th>Message</th>
                      </tr>
                    </thead>
                    <tbody>
                      {#if streamEvents.length === 0}
                        <tr>
                          <td colspan="3" class="py-8 text-center text-base-content/60">No realtime events yet</td>
                        </tr>
                      {:else}
                        {#each streamEvents as event (event.id)}
                          <tr>
                            <td class="font-mono text-xs">{event.time}</td>
                            <td><span class="badge badge-outline">{event.type}</span></td>
                            <td>{event.message}</td>
                          </tr>
                        {/each}
                      {/if}
                    </tbody>
                  </table>
                </div>
              </div>
            {/if}
          </div>
        {/each}
      </div>
    </div>
  </div>
</section>

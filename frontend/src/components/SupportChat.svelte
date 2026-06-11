<script>
  import { startSupportConversation, sendSupportMessage, getSupportMessages, submitSupportLead } from '../api.js';

  const VISITOR_ID_KEY = 'support_visitor_id';

  let { } = $props();

  let open = $state(false);
  let minimized = $state(false);
  let visitorId = $state('');
  let conversationId = $state('');
  let messages = $state([]);
  let inputText = $state('');
  let loading = $state(false);
  let error = $state('');
  let showLeadForm = $state(false);
  let leadSubmitted = $state(false);
  let leadForm = $state({ name: '', company: '', phone: '', email: '', need_description: '' });
  let leadLoading = $state(false);
  let leadError = $state('');

  function getVisitorId() {
    try {
      return localStorage.getItem(VISITOR_ID_KEY) || '';
    } catch {
      return '';
    }
  }

  function setVisitorId(id) {
    try {
      localStorage.setItem(VISITOR_ID_KEY, id);
    } catch {
      // storage unavailable
    }
  }

  function getSourcePage() {
    try {
      return window.location.pathname + window.location.search;
    } catch {
      return '';
    }
  }

  function getSourceReferrer() {
    try {
      return document.referrer || '';
    } catch {
      return '';
    }
  }

  async function initConversation() {
    visitorId = getVisitorId();
    try {
      const result = await startSupportConversation(
        visitorId,
        getSourcePage(),
        getSourceReferrer()
      );
      conversationId = result.conversation_id;
      visitorId = result.visitor_id;
      setVisitorId(visitorId);
      await loadMessages();
    } catch (err) {
      error = err.message || 'Failed to start conversation';
    }
  }

  async function loadMessages() {
    if (!conversationId) return;
    try {
      const msgList = await getSupportMessages(conversationId);
      messages = Array.isArray(msgList) ? msgList : [];
    } catch (err) {
      // silently fail, will retry on next open
    }
  }

  async function handleSend() {
    const text = inputText.trim();
    if (!text || loading || !conversationId) return;

    // Add visitor message optimistically
    messages = [...messages, {
      id: 'temp-' + Date.now(),
      role: 'visitor',
      content: text,
      created_at: new Date().toISOString()
    }];
    inputText = '';
    loading = true;
    error = '';

    try {
      const result = await sendSupportMessage(conversationId, text);
      await loadMessages();

      // Check for lead capture trigger
      if (result && result.lead_capture_prompt && !leadSubmitted) {
        showLeadForm = true;
      }
    } catch (err) {
      error = err.message || 'Failed to send message';
    } finally {
      loading = false;
    }
  }

  function handleKeyPress(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }

  function toggleChat() {
    if (!open) {
      open = true;
      minimized = false;
      if (!conversationId) {
        initConversation();
      } else {
        loadMessages();
      }
    } else {
      minimized = !minimized;
    }
  }

  function closeChat() {
    open = false;
    minimized = false;
  }

  async function handleLeadSubmit() {
    if (leadLoading || leadSubmitted) return;
    leadLoading = true;
    leadError = '';
    try {
      await submitSupportLead(conversationId, {
        name: leadForm.name,
        company: leadForm.company,
        phone: leadForm.phone,
        email: leadForm.email,
        need_description: leadForm.need_description
      });
      leadSubmitted = true;
      showLeadForm = false;
    } catch (err) {
      leadError = err.message || 'Failed to submit contact info';
    } finally {
      leadLoading = false;
    }
  }

  function isValidLeadForm() {
    const email = (leadForm.email || '').trim();
    const phone = (leadForm.phone || '').trim();
    const desc = (leadForm.need_description || '').trim();
    return (email || phone) && desc;
  }

  function formatTime(dateStr) {
    if (!dateStr) return '';
    try {
      const date = new Date(dateStr);
      return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    } catch {
      return '';
    }
  }
</script>

{#if !open}
  <!-- Floating launcher button -->
  <button
    class="fixed bottom-6 right-6 z-50 btn btn-primary btn-circle shadow-lg h-14 w-14"
    onclick={toggleChat}
    aria-label="Open support chat"
  >
    <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
    </svg>
  </button>
{:else}
  <!-- Chat panel -->
  <div class="fixed bottom-6 right-6 z-50 flex flex-col bg-base-100 rounded-lg shadow-2xl border border-base-300 transition-all duration-300 {minimized ? 'h-14 w-72' : 'h-[500px] w-[380px] max-w-[calc(100vw-2rem)]'}">
    <!-- Header -->
    <div class="flex items-center justify-between px-4 py-3 border-b border-base-200 bg-primary text-primary-content rounded-t-lg shrink-0">
      <div class="flex items-center gap-2">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
        </svg>
        <span class="font-semibold text-sm">Support</span>
      </div>
      <div class="flex items-center gap-1">
        <button
          class="btn btn-ghost btn-xs text-primary-content"
          onclick={toggleChat}
          aria-label={minimized ? 'Expand' : 'Minimize'}
        >
          {minimized ? '+' : '-'}
        </button>
        <button
          class="btn btn-ghost btn-xs text-primary-content"
          onclick={closeChat}
          aria-label="Close chat"
        >
          x
        </button>
      </div>
    </div>

    {#if !minimized}
      <!-- Messages area -->
      <div class="flex-1 overflow-y-auto px-4 py-3 space-y-3 bg-base-200/50">
        {#if messages.length === 0 && !loading}
          <div class="text-center text-base-content/60 py-8 text-sm">
            <p class="mb-2">Welcome! How can we help you today?</p>
            <p>Ask us about our products and services.</p>
          </div>
        {/if}

        {#each messages as msg (msg.id)}
          <div class="flex {msg.role === 'visitor' ? 'justify-end' : 'justify-start'}">
            <div class="max-w-[80%] rounded-lg px-3 py-2 text-sm {msg.role === 'visitor' ? 'bg-primary text-primary-content' : 'bg-base-100 text-base-content shadow-sm border border-base-200'}">
              <p class="whitespace-pre-wrap break-words">{msg.content}</p>
              <span class="text-xs opacity-60 mt-1 block">{formatTime(msg.created_at)}</span>
            </div>
          </div>
        {/each}

        {#if loading}
          <div class="flex justify-start">
            <div class="bg-base-100 rounded-lg px-3 py-2 shadow-sm border border-base-200">
              <span class="loading loading-dots loading-sm"></span>
            </div>
          </div>
        {/if}

        {#if showLeadForm && !leadSubmitted}
          <!-- Lead capture form -->
          <div class="mx-2 mb-2 p-3 rounded-lg bg-primary/10 border border-primary/30">
            <p class="text-sm font-medium mb-2">Would you like us to follow up? Please leave your contact info.</p>
            {#if leadError}
              <p class="text-error text-xs mb-2">{leadError}</p>
            {/if}
            <div class="space-y-2">
              <input type="text" class="input input-bordered input-xs w-full" placeholder="Your name" bind:value={leadForm.name} />
              <input type="text" class="input input-bordered input-xs w-full" placeholder="Company" bind:value={leadForm.company} />
              <div class="flex gap-1">
                <input type="email" class="input input-bordered input-xs flex-1" placeholder="Email" bind:value={leadForm.email} />
                <input type="tel" class="input input-bordered input-xs flex-1" placeholder="Phone" bind:value={leadForm.phone} />
              </div>
              <textarea class="textarea textarea-bordered textarea-xs w-full" rows="2" placeholder="What do you need help with?" bind:value={leadForm.need_description}></textarea>
              <div class="flex gap-1 justify-end">
                <button class="btn btn-ghost btn-xs" onclick={() => { showLeadForm = false; }}>Not now</button>
                <button class="btn btn-primary btn-xs" onclick={handleLeadSubmit} disabled={leadLoading || !isValidLeadForm()}>
                  {#if leadLoading}
                    <span class="loading loading-spinner loading-xs"></span>
                  {:else}
                    Submit
                  {/if}
                </button>
              </div>
            </div>
          </div>
        {/if}

        {#if error}
          <div class="text-center text-error text-xs">{error}</div>
        {/if}
      </div>

      <!-- Input area -->
      {#if !leadSubmitted || !showLeadForm}
      <div class="px-4 py-3 border-t border-base-200 bg-base-100 rounded-b-lg shrink-0">
        {#if leadSubmitted}
          <p class="text-sm text-success text-center mb-2">Thank you! We will follow up with you soon.</p>
        {/if}
        <div class="flex gap-2">
          <input
            type="text"
            class="input input-bordered input-sm flex-1"
            placeholder="Type your message..."
            bind:value={inputText}
            onkeypress={handleKeyPress}
            disabled={loading}
          />
          <button
            class="btn btn-primary btn-sm"
            onclick={handleSend}
            disabled={loading || !inputText.trim()}
          >
            {#if loading}
              <span class="loading loading-spinner loading-xs"></span>
            {:else}
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
              </svg>
            {/if}
          </button>
        </div>
      </div>
      {/if}
    {/if}
  </div>
{/if}

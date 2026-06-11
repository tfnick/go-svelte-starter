<script>
  import { listSupportConversations, getSupportConversation, listSupportLeads, getSupportLead } from '../api.js';

  let activeTab = $state('conversations');
  let conversations = $state({ items: [], total: 0 });
  let leads = $state({ items: [], total: 0 });
  let loading = $state(false);
  let error = $state('');
  let page = $state(1);
  let pageSize = $state(20);
  // Detail expansion
  let expandedConv = $state(null);
  let expandedLead = $state(null);
  let detailLoading = $state(false);

  async function loadConversations() {
    loading = true;
    error = '';
    try {
      const result = await listSupportConversations(page, pageSize);
      conversations = result;
    } catch (err) {
      error = err.message || 'Failed to load conversations';
    } finally {
      loading = false;
    }
  }

  async function loadLeads() {
    loading = true;
    error = '';
    try {
      const result = await listSupportLeads(page, pageSize);
      leads = result;
    } catch (err) {
      error = err.message || 'Failed to load leads';
    } finally {
      loading = false;
    }
  }

  async function viewConversation(id) {
    detailLoading = true;
    error = '';
    try {
      expandedConv = await getSupportConversation(id);
    } catch (err) {
      error = err.message || 'Failed to load conversation';
    } finally {
      detailLoading = false;
    }
  }

  async function viewLead(id) {
    detailLoading = true;
    error = '';
    try {
      expandedLead = await getSupportLead(id);
    } catch (err) {
      error = err.message || 'Failed to load lead';
    } finally {
      detailLoading = false;
    }
  }

  function closeDetail() {
    expandedConv = null;
    expandedLead = null;
  }

  function switchTab(tab) {
    activeTab = tab;
    closeDetail();
    page = 1;
    if (tab === 'conversations') {
      loadConversations();
    } else {
      loadLeads();
    }
  }

  function statusClass(status) {
    switch (status) {
      case 'open': return 'badge-info';
      case 'lead_captured': return 'badge-success';
      case 'closed': return 'badge-ghost';
      default: return 'badge-ghost';
    }
  }

  // Load initial data
  $effect(() => {
    loadConversations();
  });
</script>

<div>
  <h1 class="text-2xl font-bold mb-6">Support Console</h1>

  {#if error}
    <div class="alert alert-error mb-4">{error}</div>
  {/if}

  <!-- Tabs -->
  <div class="tabs tabs-bordered mb-4">
    <button
      class="tab {activeTab === 'conversations' ? 'tab-active' : ''}"
      onclick={() => switchTab('conversations')}
    >Conversations</button>
    <button
      class="tab {activeTab === 'leads' ? 'tab-active' : ''}"
      onclick={() => switchTab('leads')}
    >Leads</button>
  </div>

  {#if loading && activeTab === 'conversations'}
    <div class="flex justify-center py-8">
      <span class="loading loading-spinner loading-md"></span>
    </div>
  {:else if activeTab === 'conversations'}
    <!-- Conversations Table -->
    <div class="overflow-x-auto">
      <table class="table table-zebra table-sm">
        <thead>
          <tr>
            <th>Created</th>
            <th>Source</th>
            <th>Status</th>
            <th>Messages</th>
            <th>Lead</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {#if conversations.items && conversations.items.length > 0}
            {#each conversations.items as conv}
              <tr>
                <td class="text-xs">{new Date(conv.created_at).toLocaleDateString()}</td>
                <td class="text-xs max-w-40 truncate">{conv.source_page || '/'}</td>
                <td><span class="badge badge-xs {statusClass(conv.status)}">{conv.status}</span></td>
                <td class="text-center">{conv.message_count}</td>
                <td>{#if conv.has_lead}<span class="badge badge-success badge-xs">Yes</span>{:else}-{/if}</td>
                <td>
                  <button class="btn btn-ghost btn-xs" onclick={() => viewConversation(conv.id)}>
                    View
                  </button>
                </td>
              </tr>
            {/each}
          {:else}
            <tr><td colspan="6" class="text-center text-base-content/60 py-4">No conversations found</td></tr>
          {/if}
        </tbody>
      </table>
    </div>

    <!-- Conversation Detail Panel -->
    {#if expandedConv}
      <div class="card bg-base-100 shadow-lg border border-base-300 mt-4">
        <div class="card-body p-4">
          <div class="flex items-center justify-between mb-3">
            <h3 class="font-semibold">Conversation Detail</h3>
            <button class="btn btn-ghost btn-xs" onclick={closeDetail}>Close</button>
          </div>

          <div class="grid grid-cols-2 gap-2 text-sm mb-4">
            <div><span class="text-base-content/60">ID:</span> {expandedConv.conversation.id}</div>
            <div>
              <span class="text-base-content/60">Status:</span>
              <span class="badge badge-xs ml-1 {statusClass(expandedConv.conversation.status)}">{expandedConv.conversation.status}</span>
            </div>
            <div><span class="text-base-content/60">Source:</span> {expandedConv.conversation.source_page || '/'}</div>
            <div><span class="text-base-content/60">Visitor:</span> {expandedConv.conversation.visitor_id?.substring(0, 12)}...</div>
            <div><span class="text-base-content/60">Created:</span> {new Date(expandedConv.conversation.created_at).toLocaleString()}</div>
            <div><span class="text-base-content/60">Messages:</span> {expandedConv.conversation.message_count}</div>
          </div>

          <!-- Messages -->
          <h4 class="font-medium text-sm mb-2">Messages</h4>
          <div class="space-y-2 max-h-80 overflow-y-auto mb-4">
            {#each expandedConv.messages || [] as msg}
              <div class="chat {msg.role === 'visitor' ? 'chat-end' : 'chat-start'}">
                <div class="chat-bubble text-sm {msg.role === 'visitor' ? 'chat-bubble-primary' : 'chat-bubble-secondary'}">
                  {msg.content}
                </div>
                <div class="chat-footer text-xs opacity-50">
                  {msg.role} - {new Date(msg.created_at).toLocaleTimeString()}
                </div>
              </div>
            {/each}
          </div>

          <!-- Citations -->
          {#if expandedConv.citations && expandedConv.citations.length > 0}
            <h4 class="font-medium text-sm mb-2">Citations</h4>
            <div class="space-y-1 max-h-40 overflow-y-auto">
              {#each expandedConv.citations as cit}
                <div class="text-xs p-2 bg-base-200 rounded">
                  <span class="font-medium">{cit.source_name}</span>
                  <span class="text-base-content/60 ml-2">({cit.source_type}) - distance: {cit.distance.toFixed(3)}</span>
                  <p class="mt-1 italic">{cit.snippet}</p>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    {/if}
  {/if}

  {#if loading && activeTab === 'leads'}
    <div class="flex justify-center py-8">
      <span class="loading loading-spinner loading-md"></span>
    </div>
  {:else if activeTab === 'leads'}
    <!-- Leads Table -->
    <div class="overflow-x-auto">
      <table class="table table-zebra table-sm">
        <thead>
          <tr>
            <th>Created</th>
            <th>Name</th>
            <th>Company</th>
            <th>Email</th>
            <th>Phone</th>
            <th>Need</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {#if leads.items && leads.items.length > 0}
            {#each leads.items as lead}
              <tr>
                <td class="text-xs">{new Date(lead.created_at).toLocaleDateString()}</td>
                <td class="text-sm">{lead.name || '-'}</td>
                <td class="text-sm">{lead.company || '-'}</td>
                <td class="text-xs">{lead.email || '-'}</td>
                <td class="text-xs">{lead.phone || '-'}</td>
                <td class="text-xs max-w-40 truncate">{lead.need_description || '-'}</td>
                <td>
                  <button class="btn btn-ghost btn-xs" onclick={() => viewLead(lead.id)}>
                    View
                  </button>
                </td>
              </tr>
            {/each}
          {:else}
            <tr><td colspan="7" class="text-center text-base-content/60 py-4">No leads captured yet</td></tr>
          {/if}
        </tbody>
      </table>
    </div>

    <!-- Lead Detail Panel -->
    {#if expandedLead}
      <div class="card bg-base-100 shadow-lg border border-base-300 mt-4">
        <div class="card-body p-4">
          <div class="flex items-center justify-between mb-3">
            <h3 class="font-semibold">Lead Detail</h3>
            <button class="btn btn-ghost btn-xs" onclick={closeDetail}>Close</button>
          </div>

          <div class="grid grid-cols-2 gap-2 text-sm mb-4">
            <div><span class="text-base-content/60">ID:</span> {expandedLead.lead.id}</div>
            <div><span class="text-base-content/60">Created:</span> {new Date(expandedLead.lead.created_at).toLocaleString()}</div>
            <div><span class="text-base-content/60">Name:</span> {expandedLead.lead.name || '-'}</div>
            <div><span class="text-base-content/60">Company:</span> {expandedLead.lead.company || '-'}</div>
            <div><span class="text-base-content/60">Email:</span> {expandedLead.lead.email || '-'}</div>
            <div><span class="text-base-content/60">Phone:</span> {expandedLead.lead.phone || '-'}</div>
            <div><span class="text-base-content/60">Source:</span> {expandedLead.lead.source_page || '-'}</div>
            <div><span class="text-base-content/60">Intent:</span> {expandedLead.lead.detected_intent || '-'}</div>
          </div>

          <div class="mb-4">
            <h4 class="text-sm font-medium mb-1">Need Description</h4>
            <p class="text-sm bg-base-200 rounded p-2">{expandedLead.lead.need_description || 'None provided'}</p>
          </div>

          {#if expandedLead.lead.conversation_summary}
            <div class="mb-4">
              <h4 class="text-sm font-medium mb-1">Conversation Summary</h4>
              <p class="text-xs bg-base-200 rounded p-2 whitespace-pre-wrap">{expandedLead.lead.conversation_summary}</p>
            </div>
          {/if}

          {#if expandedLead.conversation && expandedLead.conversation.id}
            <div class="border-t border-base-200 pt-3">
              <h4 class="text-sm font-medium mb-2">Linked Conversation</h4>
              <div class="grid grid-cols-2 gap-2 text-sm">
                <div><span class="text-base-content/60">ID:</span> {expandedLead.conversation.id}</div>
                <div>
                  <span class="text-base-content/60">Status:</span>
                  <span class="badge badge-xs ml-1 {statusClass(expandedLead.conversation.status)}">{expandedLead.conversation.status}</span>
                </div>
                <div><span class="text-base-content/60">Messages:</span> {expandedLead.conversation.message_count}</div>
                <div>
                  <button class="btn btn-ghost btn-xs" onclick={() => viewConversation(expandedLead.conversation.id)}>
                    View Conversation
                  </button>
                </div>
              </div>
            </div>
          {/if}
        </div>
      </div>
    {/if}
  {/if}
</div>

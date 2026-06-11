<script>
  import {
    listKBSources,
    createKBSource,
    updateKBSource,
    setKBSourceEnabled,
    listKBDocuments,
    createKBDocument,
    updateKBDocument,
    setKBDocumentEnabled,
    reindexKBDocument
  } from '../api.js';

  let sources = $state([]);
  let expandedSource = $state(null);
  let documents = $state([]);
  let loading = $state(false);
  let error = $state('');
  let message = $state('');

  // Create/Edit modal state
  let showSourceModal = $state(false);
  let showDocModal = $state(false);
  let editingSource = $state(null);
  let editingDoc = $state(null);
  let sourceForm = $state({ title: '', description: '', source_type: 'manual' });

  // Form state for documents
  let docForm = $state({ title: '', content: '' });
  let docSourceId = $state('');

  async function loadSources() {
    loading = true;
    error = '';
    try {
      sources = await listKBSources();
    } catch (err) {
      error = err.message || 'Failed to load sources';
    } finally {
      loading = false;
    }
  }

  async function toggleSource(sourceId) {
    if (expandedSource === sourceId) {
      expandedSource = null;
      return;
    }
    expandedSource = sourceId;
    documents = [];
    try {
      documents = await listKBDocuments(sourceId);
    } catch (err) {
      error = err.message || 'Failed to load documents';
    }
  }

  function openNewSource() {
    editingSource = null;
    sourceForm = { title: '', description: '', source_type: 'manual' };
    showSourceModal = true;
  }

  function openEditSource(source) {
    editingSource = source;
    sourceForm = {
      title: source.title || '',
      description: source.description || '',
      source_type: source.source_type || 'manual'
    };
    showSourceModal = true;
  }

  async function handleSaveSource() {
    error = '';
    message = '';
    try {
      const payload = {
        title: sourceForm.title,
        description: sourceForm.description,
        source_type: sourceForm.source_type
      };
      if (editingSource) {
        await updateKBSource(editingSource.id, payload);
        message = 'Source updated successfully';
      } else {
        await createKBSource(payload);
        message = 'Source created successfully';
      }
      showSourceModal = false;
      await loadSources();
    } catch (err) {
      error = err.message || 'Failed to save source';
    }
  }

  async function toggleSourceEnabled(sourceId, enabled) {
    try {
      await setKBSourceEnabled(sourceId, enabled);
      await loadSources();
    } catch (err) {
      error = err.message || 'Failed to toggle source';
    }
  }

  function openNewDocument(sourceId) {
    editingDoc = null;
    docSourceId = sourceId;
    docForm = { title: '', content: '' };
    showDocModal = true;
  }

  function openEditDocument(doc) {
    editingDoc = doc;
    docSourceId = doc.source_id;
    docForm = { title: doc.title || '', content: doc.content || '' };
    showDocModal = true;
  }

  async function handleSaveDocument() {
    error = '';
    message = '';
    try {
      const payload = { title: docForm.title, content: docForm.content };
      if (editingDoc) {
        await updateKBDocument(docSourceId, editingDoc.id, payload);
        message = 'Document updated successfully';
      } else {
        await createKBDocument(docSourceId, payload);
        message = 'Document created successfully';
      }
      showDocModal = false;
      if (expandedSource) {
        documents = await listKBDocuments(expandedSource);
      }
    } catch (err) {
      error = err.message || 'Failed to save document';
    }
  }

  async function toggleDocEnabled(sourceId, docId, enabled) {
    try {
      await setKBDocumentEnabled(sourceId, docId, enabled);
      if (expandedSource) {
        documents = await listKBDocuments(expandedSource);
      }
    } catch (err) {
      error = err.message || 'Failed to toggle document';
    }
  }

  async function reindexDoc(docId) {
    try {
      await reindexKBDocument(docId);
      if (expandedSource) {
        documents = await listKBDocuments(expandedSource);
      }
    } catch (err) {
      error = err.message || 'Failed to reindex document';
    }
  }

  function indexStatusClass(status) {
    switch (status) {
      case 'indexed': return 'badge-success';
      case 'pending': return 'badge-warning';
      case 'processing': return 'badge-info';
      case 'failed': return 'badge-error';
      default: return 'badge-ghost';
    }
  }

  function formatDate(dateStr) {
    if (!dateStr) return '-';
    try {
      return new Date(dateStr).toLocaleDateString();
    } catch {
      return dateStr;
    }
  }

  $effect(() => {
    loadSources();
  });
</script>

<div>
  <div class="flex items-center justify-between mb-6">
    <h1 class="text-2xl font-bold">Knowledge Base</h1>
    <button class="btn btn-primary btn-sm" onclick={openNewSource}>New Source</button>
  </div>

  {#if message}
    <div class="alert alert-success mb-4">{message}</div>
  {/if}
  {#if error}
    <div class="alert alert-error mb-4">{error}</div>
  {/if}

  <!-- Source Creation/Edit Modal -->
  {#if showSourceModal}
    <div class="modal modal-open">
      <div class="modal-box">
        <h3 class="font-bold text-lg mb-4">{editingSource ? 'Edit Source' : 'New Source'}</h3>
        <div class="form-control mb-3">
          <label class="label"><span class="label-text">Title *</span></label>
          <input type="text" class="input input-bordered" bind:value={sourceForm.title} />
        </div>
        <div class="form-control mb-3">
          <label class="label"><span class="label-text">Description</span></label>
          <textarea class="textarea textarea-bordered" rows="3" bind:value={sourceForm.description}></textarea>
        </div>
        <div class="form-control mb-4">
          <label class="label"><span class="label-text">Source Type</span></label>
          <select class="select select-bordered" bind:value={sourceForm.source_type}>
            <option value="manual">Manual</option>
            <option value="markdown">Markdown</option>
            <option value="url">URL</option>
          </select>
        </div>
        <div class="modal-action">
          <button class="btn btn-ghost btn-sm" onclick={() => { showSourceModal = false; }}>Cancel</button>
          <button class="btn btn-primary btn-sm" onclick={handleSaveSource} disabled={!sourceForm.title.trim()}>
            {editingSource ? 'Update' : 'Create'}
          </button>
        </div>
      </div>
      <div class="modal-backdrop" onclick={() => { showSourceModal = false; }}></div>
    </div>
  {/if}

  <!-- Document Create/Edit Modal -->
  {#if showDocModal}
    <div class="modal modal-open">
      <div class="modal-box max-w-2xl">
        <h3 class="font-bold text-lg mb-4">{editingDoc ? 'Edit Document' : 'New Document'}</h3>
        <div class="form-control mb-3">
          <label class="label"><span class="label-text">Title *</span></label>
          <input type="text" class="input input-bordered" bind:value={docForm.title} />
        </div>
        <div class="form-control mb-4">
          <label class="label"><span class="label-text">Content *</span></label>
          <textarea class="textarea textarea-bordered h-40" bind:value={docForm.content} placeholder="Enter document content..."></textarea>
        </div>
        <div class="modal-action">
          <button class="btn btn-ghost btn-sm" onclick={() => { showDocModal = false; }}>Cancel</button>
          <button class="btn btn-primary btn-sm" onclick={handleSaveDocument}
            disabled={!docForm.title.trim() || !docForm.content.trim()}>
            {editingDoc ? 'Update' : 'Create'}
          </button>
        </div>
      </div>
      <div class="modal-backdrop" onclick={() => { showDocModal = false; }}></div>
    </div>
  {/if}

  {#if loading}
    <div class="flex justify-center py-8">
      <span class="loading loading-spinner loading-md"></span>
    </div>
  {:else if sources && sources.length > 0}
    <div class="space-y-2">
      {#each sources as source}
        <div class="card bg-base-100 shadow-sm border border-base-200">
          <div class="card-body p-4">
            <div class="flex items-center justify-between">
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <span class="font-semibold truncate">{source.title}</span>
                  <span class="badge badge-xs">{source.source_type}</span>
                  <span class="badge badge-xs {indexStatusClass(source.index_status)}">{source.index_status}</span>
                </div>
                {#if source.last_index_error}
                  <p class="text-xs text-error mt-1 truncate">{source.last_index_error}</p>
                {/if}
              </div>
              <div class="flex items-center gap-1">
                <input
                  type="checkbox"
                  class="toggle toggle-sm"
                  checked={source.enabled}
                  onchange={() => toggleSourceEnabled(source.id, !source.enabled)}
                />
                <button class="btn btn-ghost btn-xs" onclick={() => openEditSource(source)}>Edit</button>
                <button class="btn btn-ghost btn-xs" onclick={() => toggleSource(source.id)}>
                  {expandedSource === source.id ? 'Collapse' : 'Docs'}
                </button>
              </div>
            </div>

            <!-- Documents list -->
            {#if expandedSource === source.id}
              <div class="mt-3 border-t border-base-200 pt-3">
                <div class="flex items-center justify-between mb-2">
                  <span class="text-sm font-medium">Documents</span>
                  <button class="btn btn-ghost btn-xs" onclick={() => openNewDocument(source.id)}>+ Add Document</button>
                </div>

                {#if documents && documents.length > 0}
                  <div class="space-y-1">
                    {#each documents as doc}
                      <div class="flex items-center justify-between p-2 bg-base-200/50 rounded text-sm">
                        <div class="flex items-center gap-2 min-w-0">
                          <span class="truncate">{doc.title}</span>
                          <span class="badge badge-xs {indexStatusClass(doc.index_status)}">{doc.index_status}</span>
                          {#if doc.last_index_error}
                            <span class="text-xs text-error truncate max-w-40">{doc.last_index_error}</span>
                          {/if}
                        </div>
                        <div class="flex items-center gap-1 shrink-0">
                          <input
                            type="checkbox"
                            class="toggle toggle-xs"
                            checked={doc.enabled}
                            onchange={() => toggleDocEnabled(doc.source_id, doc.id, !doc.enabled)}
                          />
                          {#if doc.index_status === 'failed' || doc.index_status === 'pending'}
                            <button class="btn btn-ghost btn-xs text-warning" onclick={() => reindexDoc(doc.id)}>Reindex</button>
                          {/if}
                          <button class="btn btn-ghost btn-xs" onclick={() => openEditDocument(doc)}>Edit</button>
                        </div>
                      </div>
                    {/each}
                  </div>
                {:else}
                  <p class="text-sm text-base-content/60 py-2">No documents yet.</p>
                {/if}
              </div>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <div class="text-center text-base-content/60 py-8">
      <p class="mb-2">No knowledge base sources yet.</p>
      <button class="btn btn-primary btn-sm" onclick={openNewSource}>Create Your First Source</button>
    </div>
  {/if}
</div>

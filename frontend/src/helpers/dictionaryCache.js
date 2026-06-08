export function createDictionaryCache(loadDictionaries, onChange = () => {}) {
  const state = {
    dictionaries: {},
    loading: false,
    error: ''
  };
  const pendingLoads = new Map();

  function snapshot() {
    return {
      dictionaries: { ...state.dictionaries },
      loading: state.loading,
      error: state.error
    };
  }

  function notify() {
    onChange(snapshot());
  }

  function cacheKey(types) {
    return [...types].sort().join(',');
  }

  function missingTypes(types) {
    const uniqueTypes = [...new Set((types || []).filter(Boolean))];
    return uniqueTypes.filter((type) => !(type in state.dictionaries));
  }

  async function load(types) {
    const missing = missingTypes(types);
    if (missing.length === 0) {
      return state.dictionaries;
    }

    const key = cacheKey(missing);
    const pending = pendingLoads.get(key);
    if (pending) {
      await pending;
      return state.dictionaries;
    }

    state.loading = true;
    state.error = '';
    notify();

    const promise = loadDictionaries(missing).then((response) => {
      const dictionaries = response?.dictionaries || {};
      state.dictionaries = {
        ...state.dictionaries,
        ...dictionaries
      };
      notify();
      return dictionaries;
    });

    pendingLoads.set(key, promise);

    try {
      await promise;
    } catch (error) {
      state.error = error instanceof Error ? error.message : 'Failed to load dictionaries';
      notify();
      throw error;
    } finally {
      pendingLoads.delete(key);
      state.loading = pendingLoads.size > 0;
      notify();
    }

    return state.dictionaries;
  }

  function options(type) {
    return state.dictionaries[type] || [];
  }

  function reset() {
    state.dictionaries = {};
    state.loading = false;
    state.error = '';
    pendingLoads.clear();
    notify();
  }

  return {
    state,
    load,
    options,
    reset
  };
}

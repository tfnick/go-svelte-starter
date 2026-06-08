import { readable } from 'svelte/store';

import { getDictionaries } from '../api.js';
import { createDictionaryCache } from '../helpers/dictionaryCache.js';

export type DictionaryOption = {
  value: string;
  label: string;
};

export type DictionaryState = {
  dictionaries: Record<string, DictionaryOption[]>;
  loading: boolean;
  error: string;
};

const initialState: DictionaryState = {
  dictionaries: {},
  loading: false,
  error: ''
};

let setState: (state: DictionaryState) => void = () => {};
const cache = createDictionaryCache(getDictionaries, (state: DictionaryState) => {
  setState(state);
});

export const dictionaryStore = readable<DictionaryState>(initialState, (set) => {
  setState = set;
  set({ ...cache.state });

  return () => {
    setState = () => {};
  };
});

export function loadDictionaries(types: string[]) {
  return cache.load(types);
}

export function dictionaryOptions(type: string): DictionaryOption[] {
  return cache.options(type);
}

export function resetDictionariesForTest() {
  cache.reset();
}

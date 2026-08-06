/* App State — simple reactive store for demo */
(function(global) {
  const listeners = [];
  const state = {
    currentCat: 'cat-whit',
    currentPage: 'today',
    isOnline: navigator.onLine !== false,
    isAIEnabled: true,
    isOffline: false,
    draftRecords: [],
    completedReminders: [],
    aiMode: 'normal' // normal | disabled | error
  };

  function getState() { return state; }

  function setState(updates) {
    Object.assign(state, updates);
    listeners.forEach(fn => fn(state));
  }

  function subscribe(fn) {
    listeners.push(fn);
    return () => {
      const idx = listeners.indexOf(fn);
      if (idx > -1) listeners.splice(idx, 1);
    };
  }

  // Router
  function navigate(page, params) {
    state.currentPage = page;
    state.currentParams = params || {};
    listeners.forEach(fn => fn(state));
    // Update URL hash
    if (params && Object.keys(params).length) {
      const qs = Object.entries(params).map(([k,v]) => k + '=' + encodeURIComponent(v)).join('&');
      location.hash = page + '?' + qs;
    } else {
      location.hash = page;
    }
    // Re-render current page
    if (global.renderPage) {
      global.renderPage(page, params);
    }
  }

  // Init from hash
  function initFromHash() {
    const hash = location.hash.replace('#', '') || 'today';
    const [page, qs] = hash.split('?');
    const params = {};
    if (qs) {
      qs.split('&').forEach(pair => {
        const [k, v] = pair.split('=');
        params[k] = decodeURIComponent(v);
      });
    }
    state.currentPage = page || 'today';
    state.currentParams = params;
    if (global.renderPage) {
      global.renderPage(page, params);
    }
  }

  global.AppState = { getState, setState, subscribe, navigate, initFromHash };
})(typeof window !== 'undefined' ? window : global);

/* Router — hash-based navigation for demo */
(function(global) {
  const routes = {
    '': 'today',
    'today': 'today',
    'onboarding': 'onboarding',
    'records': 'records',
    'records/quick': 'quick-record',
    'records/ai': 'ai-input',
    'records/ai/confirm': 'ai-confirm',
    'cats': 'cats',
    'cats/:catId': 'cat-detail',
    'cats/:catId/trends': 'trends',
    'cats/:catId/medical': 'medical',
    'medical/upload': 'medical-upload',
    'reminders': 'reminders',
    'moments': 'moments',
    'family': 'family',
    'family/inventory': 'inventory',
    'family/expenses': 'expenses',
    'settings': 'settings'
  };

  function resolve(hash) {
    const path = hash.replace('#', '') || 'today';
    // Try exact match first
    if (routes[path]) return { page: routes[path], params: {} };
    // Try param match
    for (const [pattern, page] of Object.entries(routes)) {
      if (pattern.includes(':')) {
        const regexStr = '^' + pattern.replace(/:[^/]+/g, '([^/]+)') + '$';
        const regex = new RegExp(regexStr);
        const match = path.match(regex);
        if (match) {
          const params = {};
          const keys = pattern.match(/:[^/]+/g).map(k => k.slice(1));
          keys.forEach((k, i) => { params[k] = match[i + 1]; });
          return { page, params };
        }
      }
    }
    return { page: 'today', params: {} };
  }

  global.AppRouter = { routes, resolve };
})(typeof window !== 'undefined' ? window : global);

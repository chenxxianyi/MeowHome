/* App.js — render dispatcher, bottom nav, shell */
(function(global) {
  const root = document.getElementById('app-root');

  // Bottom navigation config
  const navItems = [
    { id: 'today', label: '今日', icon: 'home' },
    { id: 'records', label: '记录', icon: 'record' },
    { id: 'cats', label: '猫咪', icon: 'cat' },
    { id: 'moments', label: '时光', icon: 'timeline' },
    { id: 'family', label: '家庭', icon: 'family' }
  ];

  function renderShell(page, params) {
    const state = global.AppState.getState();
    const activeNav = page;

    root.innerHTML = `
      <div class="page" id="page-container">
        <div id="page-header"></div>
        <div id="page-content" class="page-content"></div>
      </div>
      <div class="bottom-nav" role="navigation" aria-label="主导航">
        ${navItems.map(item => `
          <button class="nav-item ${activeNav === item.id ? 'active' : ''}" data-nav="${item.id}" aria-label="${item.label}" aria-current="${activeNav === item.id ? 'page' : 'false'}">
            ${global.AppIcons.icon(item.icon, 24)}
            <span>${item.label}</span>
          </button>
        `).join('')}
      </div>
    `;

    // Nav click handlers
    root.querySelectorAll('.nav-item').forEach(btn => {
      btn.addEventListener('click', () => {
        const pageId = btn.dataset.nav;
        global.AppState.navigate(pageId);
      });
    });
  }

  function renderPage(page, params) {
    renderShell(page, params);
    const headerEl = document.getElementById('page-header');
    const contentEl = document.getElementById('page-content');

    if (!headerEl || !contentEl) return;

    // Dispatch to page controller
    switch (page) {
      case 'today':
        renderTodayPage(headerEl, contentEl, params);
        break;
      case 'onboarding':
        renderOnboardingPage(headerEl, contentEl, params);
        break;
      case 'records':
        renderRecordsPage(headerEl, contentEl, params);
        break;
      case 'quick-record':
        renderQuickRecordPage(headerEl, contentEl, params);
        break;
      case 'ai-input':
        renderAIInputPage(headerEl, contentEl, params);
        break;
      case 'ai-confirm':
        renderAIConfirmPage(headerEl, contentEl, params);
        break;
      case 'cats':
        renderCatsPage(headerEl, contentEl, params);
        break;
      case 'cat-detail':
        renderCatDetailPage(headerEl, contentEl, params);
        break;
      case 'trends':
        renderTrendsPage(headerEl, contentEl, params);
        break;
      case 'medical-upload':
        renderMedicalUploadPage(headerEl, contentEl, params);
        break;
      case 'reminders':
        renderRemindersPage(headerEl, contentEl, params);
        break;
      case 'moments':
        renderMomentsPage(headerEl, contentEl, params);
        break;
      case 'family':
        renderFamilyPage(headerEl, contentEl, params);
        break;
      case 'inventory':
        renderInventoryPage(headerEl, contentEl, params);
        break;
      case 'expenses':
        renderExpensesPage(headerEl, contentEl, params);
        break;
      default:
        renderTodayPage(headerEl, contentEl, params);
    }
  }

  global.renderPage = renderPage;

  // Init
  window.addEventListener('hashchange', () => {
    const resolved = global.AppRouter.resolve(location.hash);
    global.AppState.setState({ currentPage: resolved.page, currentParams: resolved.params });
    renderPage(resolved.page, resolved.params);
  });

  // Initial render
  const initial = global.AppRouter.resolve(location.hash);
  global.AppState.setState({ currentPage: initial.page, currentParams: initial.params });
  renderPage(initial.page, initial.params);

  // Offline detection
  window.addEventListener('online', () => {
    global.AppState.setState({ isOnline: true, isOffline: false });
    global.UI.toast('网络已恢复', 'success');
  });
  window.addEventListener('offline', () => {
    global.AppState.setState({ isOnline: false, isOffline: true });
    global.UI.toast('网络已断开，离线模式', 'info');
  });

  if (!navigator.onLine) {
    global.AppState.setState({ isOnline: false, isOffline: true });
  }
})(typeof window !== 'undefined' ? window : global);

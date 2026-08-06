/* Render utils — shared helpers for demo */
(function(global) {
  const I = () => global.AppIcons;

  // Toast
  let toastTimer = null;
  function toast(message, type) {
    let el = document.getElementById('toast-el');
    if (!el) {
      el = document.createElement('div');
      el.id = 'toast-el';
      el.className = 'toast';
      document.body.appendChild(el);
    }
    el.className = 'toast ' + (type || 'info');
    el.textContent = message;
    el.classList.add('show');
    if (toastTimer) clearTimeout(toastTimer);
    toastTimer = setTimeout(() => el.classList.remove('show'), 2200);
  }

  // Cat avatar
  function catAvatar(cat, size) {
    if (cat && cat.avatar) {
      return `<img class="cat-avatar-img" src="${cat.avatar}" alt="${cat.name}" style="width:${size}px;height:${size}px;border-radius:50%;object-fit:cover;">`;
    }
    return `<span class="cat-avatar-sm" style="width:${size}px;height:${size}px;">${I().icon('cat', size * 0.5)}</span>`;
  }

  // Cat switcher
  function catSwitcher(currentCat) {
    const cats = global.Mock.cats;
    const all = ['cat-whit', 'cat-oran', 'all'];
    return `
      <div class="cat-switcher" role="tablist" aria-label="切换猫咪">
        <button class="cat-switcher-btn ${currentCat === 'all' ? 'active' : ''}" data-cat="all" aria-pressed="${currentCat === 'all'}">
          ${I().icon('cat', 20)} 全部
        </button>
        ${cats.map(c => `
          <button class="cat-switcher-btn ${currentCat === c.id ? 'active' : ''}" data-cat="${c.id}" aria-pressed="${currentCat === c.id}">
            ${catAvatar(c, 28)} ${c.name}
          </button>
        `).join('')}
      </div>
    `;
  }

  // Status row
  function statusRow(label, value, state) {
    const cls = state === 'normal' || state === 'none' ? 'normal' : state;
    return `
      <div class="status-row">
        <span class="status-label"></span>
        <span class="status-value ${cls}">${value}</span>
      </div>
    `;
  }

  // Reminder item
  function reminderHtml(item, completed) {
    const done = completed.includes(item.id) || item.state === 'done';
    return `
      <div class="reminder-item ${done ? 'is-done' : ''}" data-reminder-id="${item.id}" data-reminder-type="${item.type}">
        <div class="reminder-left">
          <div class="reminder-icon ${item.icon}">${I().icon(item.icon === 'water' ? 'water' : item.icon, 18)}</div>
          <div class="reminder-text">
            <div class="reminder-title">${item.title}</div>
            <div class="reminder-subtitle">${item.subtitle} · ${item.time}</div>
          </div>
        </div>
        ${done ? `
          <span class="reminder-done-badge"><span class="done-check">${I().icon('check', 14)}</span> 已完成</span>
        ` : `
          <div class="reminder-actions">
            <button class="reminder-btn" data-action="later">稍后</button>
            <button class="reminder-btn primary" data-action="done">完成</button>
          </div>
        `}
      </div>
    `;
  }

  // Empty state
  function emptyState(icon, title, desc) {
    return `
      <div class="empty-state">
        ${I().icon(icon, 40)}
        <div class="empty-state-title">${title}</div>
        <div class="empty-state-desc">${desc || ''}</div>
      </div>
    `;
  }

  // Loading skeleton
  function skeleton() {
    return `
      <div class="skeleton-block" aria-busy="true" role="status">
        <div class="skeleton"></div>
        <div class="skeleton"></div>
        <div class="skeleton" style="width:60%"></div>
      </div>
    `;
  }

  // AI badge
  function aiBadge(label) {
    return `<span class="ai-badge">${I().icon('ai', 12)} ${label || 'AI 整理'}</span>`;
  }

  global.UI = { toast, catAvatar, catSwitcher, statusRow, reminderHtml, emptyState, skeleton, aiBadge };
})(typeof window !== 'undefined' ? window : global);

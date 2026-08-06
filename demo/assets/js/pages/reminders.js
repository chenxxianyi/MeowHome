/* Reminders page controller */
(function(global) {
  const I = () => global.AppIcons;
  const M = () => global.Mock;
  let currentFilter = 'todo';

  function renderHeader(headerEl) {
    headerEl.innerHTML = `
      <div class="page-header">
        <div class="page-title">提醒中心</div>
      </div>
    `;
  }

  function renderRemindersPage(headerElArg, contentElArg) {
    renderHeader(headerElArg);
    const reminders = M().reminders;
    const todo = reminders.filter(r => r.state === 'todo');
    const done = reminders.filter(r => r.state === 'done');

    contentElArg.innerHTML = `
      <div style="display:flex;gap:8px;margin-bottom:16px;" role="tablist" aria-label="提醒筛选">
        <button class="reminder-btn active" data-filter="todo" role="tab" aria-selected="true">待处理 (${todo.length})</button>
        <button class="reminder-btn" data-filter="done" role="tab" aria-selected="false">已完成 (${done.length})</button>
        <button class="reminder-btn" data-filter="all" role="tab" aria-selected="false">全部 (${reminders.length})</button>
      </div>
      <div class="reminder-list" id="reminder-list">
        ${getRemindersHtml(reminders, currentFilter)}
      </div>
    `;

    // Filter tabs
    contentElArg.querySelectorAll('[data-filter]').forEach(btn => {
      btn.addEventListener('click', () => {
        contentElArg.querySelectorAll('[data-filter]').forEach(b => { b.classList.remove('active'); b.setAttribute('aria-selected', 'false'); });
        btn.classList.add('active');
        btn.setAttribute('aria-selected', 'true');
        currentFilter = btn.dataset.filter;
        contentElArg.querySelector('#reminder-list').innerHTML = getRemindersHtml(reminders, currentFilter);
        bindReminderActions(contentElArg.querySelector('#reminder-list'));
      });
    });

    bindReminderActions(contentElArg.querySelector('#reminder-list'));
  }

  function getRemindersHtml(reminders, filter) {
    let list = reminders;
    if (filter === 'todo') list = reminders.filter(r => r.state === 'todo');
    if (filter === 'done') list = reminders.filter(r => r.state === 'done');
    if (!list.length) return global.UI.emptyState('bell', '暂无提醒', filter === 'todo' ? '当前没有待处理提醒' : '没有已完成的提醒');
    return list.map(item => {
      const done = item.state === 'done';
      return `
        <div class="reminder-item ${done ? 'is-done' : ''}" data-reminder-id="${item.id}">
          <div class="reminder-left">
            <div class="reminder-icon ${item.icon}">${I().icon(item.icon === 'water' ? 'water' : item.icon, 18)}</div>
            <div class="reminder-text">
              <div class="reminder-title">${item.title}</div>
              <div class="reminder-subtitle">${item.subtitle} · ${item.time}</div>
            </div>
          </div>
          ${done ? '<span class="done-check">✓</span>' : `
            <div class="reminder-actions">
              <button class="reminder-btn" data-action="later">稍后</button>
              <button class="reminder-btn primary" data-action="done">完成</button>
            </div>
          `}
        </div>
      `;
    }).join('');
  }

  function bindReminderActions(listEl) {
    listEl.querySelectorAll('[data-action="done"]').forEach(btn => {
      btn.addEventListener('click', () => {
        const item = btn.closest('.reminder-item');
        const id = item.dataset.reminderId;
        const completed = global.AppState.getState().completedReminders;
        completed.push(id);
        global.AppState.setState({ completedReminders: completed });
        item.querySelector('.reminder-actions').innerHTML = '<span class="ai-badge">已完成</span>';
        global.UI.toast('提醒已完成', 'success');
      });
    });
    listEl.querySelectorAll('[data-action="later"]').forEach(btn => {
      btn.addEventListener('click', () => {
        global.UI.toast('已稍后提醒', 'info');
      });
    });
  }

  global.renderRemindersPage = renderRemindersPage;
})(typeof window !== 'undefined' ? window : global);

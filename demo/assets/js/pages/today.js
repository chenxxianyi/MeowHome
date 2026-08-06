/* Today page controller */
(function(global) {
  const M = () => global.Mock;
  const I = () => global.AppIcons;
  const U = () => global.UI;

  const state = { currentCat: 'cat-whit', isLoading: false };

  function renderHeader(headerEl, params) {
    const today = new Date();
    const dateLabel = today.getFullYear() + '年' + (today.getMonth()+1) + '月' + today.getDate() + '日 ' + ['周日','周一','周二','周三','周四','周五','周六'][today.getDay()];
    const hour = today.getHours();
    const greeting = hour < 6 ? '夜深了' : hour < 11 ? '早上好' : hour < 14 ? '中午好' : hour < 18 ? '下午好' : '晚上好';

    headerEl.innerHTML = `
      <div class="page-header">
        <div style="display:flex;align-items:flex-start;justify-content:space-between;width:100%;">
          <div>
            <div style="font-size:var(--font-size-title);font-weight:600;">${greeting}</div>
            <div style="font-size:var(--font-size-assist);color:var(--color-text-tertiary);margin-top:4px;">
              ${M().family.name} · ${dateLabel}
            </div>
          </div>
          <button class="ai-action-btn" data-action="reminders" aria-label="提醒中心">
            ${I().icon('bell', 20)}
          </button>
        </div>
        <div id="cat-switcher"></div>
      </div>
    `;

    // Cat switcher
    const switcherEl = headerEl.querySelector('#cat-switcher');
    switcherEl.innerHTML = U().catSwitcher(global.AppState.getState().currentCat);
    switcherEl.querySelectorAll('button[data-cat]').forEach(btn => {
      btn.addEventListener('click', () => {
        const catId = btn.dataset.cat;
        global.AppState.setState({ currentCat: catId });
        renderHeader(headerEl, params);
        loadContent(contentEl);
      });
    });
  }

  let contentEl;

  function renderTodayStatus(status) {
    const rows = [
      { label: '饮食', value: status.food.label, state: status.food.state, icon: 'food' },
      { label: '饮水', value: status.water.label, state: status.water.state, icon: 'water' },
      { label: '排便', value: status.elimination.label, state: status.elimination.state, icon: 'elimination' },
      { label: '呕吐', value: status.vomit.state === 'none' ? '无呕吐' : status.vomit.label, state: status.vomit.state, icon: 'vomit' },
      { label: '用药', value: status.medication.state === 'none' ? '无需用药' : status.medication.label, state: status.medication.state, icon: 'medication' },
      { label: '精神', value: status.mental.label, state: status.mental.state, icon: 'mental' }
    ];

    return `
      <section class="status-panel" aria-label="今日状态">
        <div class="status-panel-title">今日状态</div>
        ${rows.map(r => `
          <div class="status-row">
            <span class="status-label">${I().icon(r.icon, 18)} ${r.label}</span>
            <span class="status-value ${r.state}">${r.value}</span>
          </div>
        `).join('')}
      </section>
    `;
  }

  function renderFocusItems(items) {
    if (!items.length) return '';
    return `
      <section aria-label="需要关注">
        <div class="section-title">需要关注</div>
        ${items.map(item => `
          <div class="focus-section ${item.severity}">
            <div class="focus-title">
              ${item.severity === 'danger' ? I().icon('alertTriangle', 18) : I().icon('info', 18)}
              ${item.title}
            </div>
            <div class="focus-body">${item.body}</div>
            <div class="focus-actions">
              ${item.actions.map(a => `
                <button class="focus-action-btn ${a.action === 'ai' ? 'primary' : ''}" data-action="${a.action}" data-id="${item.id}">${a.label}</button>
              `).join('')}
            </div>
          </div>
        `).join('')}
      </section>
    `;
  }

  function renderAISummary(summary) {
    const time = summary.generatedAt ? summary.generatedAt.slice(11, 16) : '08:30';
    return `
      <section class="ai-summary" aria-label="AI 每日摘要">
        ${U().aiBadge('AI 每日整理')}
        <div class="ai-summary-title">今日速览</div>
        <div class="ai-summary-body">${summary.body}</div>
        <div class="ai-meta">
          <span>生成于 ${time}</span>
          <span>基于 ${summary.evidenceCount} 条记录</span>
          <span class="ai-evidence-link" data-action="evidence">查看依据</span>
        </div>
        <div class="ai-disclaimer">以上由 AI 整理，不构成医疗诊断。如有异常请咨询兽医。</div>
      </section>
    `;
  }

  function renderReminders() {
    const list = M().reminders.filter(r => r.state === 'todo').slice(0, 3);
    if (!list.length) return '';
    return `
      <section aria-label="今日提醒">
        <div class="section-title">今日提醒</div>
        <div class="reminder-list">
          ${list.map(item => {
            const done = global.AppState.getState().completedReminders.includes(item.id);
            if (done) return '';
            return `
              <div class="reminder-item" data-reminder-id="${item.id}">
                <div class="reminder-left">
                  <div class="reminder-icon ${item.icon}">${I().icon(item.icon === 'water' ? 'water' : item.icon, 18)}</div>
                  <div class="reminder-text">
                    <div class="reminder-title">${item.title}</div>
                    <div class="reminder-subtitle">${item.subtitle} · ${item.time}</div>
                  </div>
                </div>
                <div class="reminder-actions">
                  <button class="reminder-btn" data-action="later">稍后</button>
                  <button class="reminder-btn primary" data-action="done">完成</button>
                </div>
              </div>
            `;
          }).join('')}
        </div>
      </section>
    `;
  }

  function renderQuickRecords() {
    const items = [
      { id: 'feeding', label: '喂食', icon: 'food' },
      { id: 'elimination', label: '排便', icon: 'elimination' },
      { id: 'vomit', label: '呕吐', icon: 'vomit' },
      { id: 'weight', label: '体重', icon: 'weight' },
      { id: 'more', label: '更多', icon: 'more' }
    ];
    return `
      <section class="quick-record-section" aria-label="快捷记录">
        <div class="quick-record-grid">
          ${items.map(item => `
            <button class="quick-record-btn" data-record="${item.id}" aria-label="${item.label}">
              ${I().icon(item.icon, 22)}<span>${item.label}</span>
            </button>
          `).join('')}
        </div>
      </section>
    `;
  }

  function loadContent(container) {
    const catId = global.AppState.getState().currentCat === 'all' ? null : global.AppState.getState().currentCat;
    state.isLoading = true;
    container.innerHTML = U().skeleton();

    Promise.all([
      global.ApiService.getTodayStatus(catId || 'cat-whit'),
      global.ApiService.getFocusItems(catId),
      global.ApiService.getAISummary(catId),
      global.ApiService.getReminders('todo')
    ]).then(([statusRes, focusRes, aiRes, reminderRes]) => {
      // For "all", merge both cats' status
      let statusHtml = '';
      if (global.AppState.getState().currentCat === 'all') {
        statusHtml = renderTodayStatus(M().todayStatus['cat-whit']) + renderTodayStatus(M().todayStatus['cat-oran']);
      } else {
        statusHtml = renderTodayStatus(statusRes.data);
      }

      container.innerHTML = statusHtml + renderFocusItems(focusRes.data) + renderAISummary(aiRes.data) + renderReminders() + renderQuickRecords();
      bindContentEvents(container);
      state.isLoading = false;
    });
  }

  function bindContentEvents(container) {
    // Quick record buttons
    container.querySelectorAll('[data-record]').forEach(btn => {
      btn.addEventListener('click', () => {
        const type = btn.dataset.record;
        if (type === 'more') {
          global.AppState.navigate('records');
        } else {
          global.AppState.navigate('quick-record', { type });
        }
      });
    });

    // Focus actions
    container.querySelectorAll('.focus-action-btn').forEach(btn => {
      btn.addEventListener('click', () => {
        const action = btn.dataset.action;
        if (action === 'ai') {
          global.AppState.navigate('ai-input');
        } else if (action === 'add') {
          global.AppState.navigate('quick-record', { type: 'vomit' });
        } else if (action === 'view') {
          global.AppState.navigate('cat-detail', { catId: 'cat-whit' });
        } else {
          global.UI.toast('已记录', 'success');
        }
      });
    });

    // Reminder actions
    container.querySelectorAll('[data-action="later"], [data-action="done"]').forEach(btn => {
      btn.addEventListener('click', (e) => {
        e.stopPropagation();
        const item = btn.closest('.reminder-item');
        const id = item.dataset.reminderId;
        const action = btn.dataset.action;
        if (action === 'done') {
          const completed = global.AppState.getState().completedReminders;
          completed.push(id);
          global.AppState.setState({ completedReminders: completed });
          item.style.opacity = 0.5;
          item.querySelector('.reminder-actions').innerHTML = '<span class="ai-badge">已完成</span>';
          global.UI.toast('提醒已完成', 'success');
        } else {
          item.style.display = 'none';
          global.UI.toast('已稍后提醒', 'info');
        }
      });
    });

    // Evidence link
    container.querySelectorAll('[data-action="evidence"]').forEach(el => {
      el.addEventListener('click', () => {
        global.UI.toast('查看依据（演示）', 'info');
        // Simulate evidence panel
        const aiSection = el.closest('.ai-summary');
        const existing = aiSection.querySelector('.evidence-panel');
        if (existing) { existing.remove(); return; }
        const panel = document.createElement('div');
        panel.className = 'evidence-panel';
        panel.style.cssText = 'margin-top:12px;padding:12px;background:var(--color-ai-bg);border-radius:8px;font-size:12px;';
        const evidence = global.Mock.aiSummary.evidence.map(ev =>
          `<div style="padding:6px 0;border-bottom:1px solid var(--color-ai-border);">
            <b>${ev.type === 'feeding'?'喂食':ev.type==='vomit'?'呕吐':ev.type==='medication'?'用药':ev.type==='water'?'饮水':ev.type} · ${ev.catId==='cat-whit'?'小白':'小橘'}</b>
            <div style="color:var(--color-text-tertiary);">${ev.time} · ${ev.content}</div>
          </div>`
        ).join('');
        panel.innerHTML = `<div style="font-weight:600;margin-bottom:8px;">${I().icon('ai',12)} AI 分析依据</div>${evidence}`;
        aiSection.appendChild(panel);
      });
    });

    // Reminders bell
    headerEl.querySelector('[data-action="reminders"]').addEventListener('click', () => {
      global.AppState.navigate('reminders');
    });
  }

  let headerEl;

  function renderTodayPage(headerElArg, contentElArg, params) {
    headerEl = headerElArg;
    contentEl = contentElArg;
    renderHeader(headerEl, params);
    loadContent(contentEl);
  }

  global.renderTodayPage = renderTodayPage;
})(typeof window !== 'undefined' ? window : global);

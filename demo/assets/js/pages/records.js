/* Records page controller */
(function(global) {
  const I = () => global.AppIcons;
  const M = () => global.Mock;
  const aiEnabled = true;

  function renderHeader(headerEl) {
    headerEl.innerHTML = `
      <div class="page-header">
        <div class="page-title">记录</div>
      </div>
    `;
  }

  function renderRecordGroups(contentEl) {
    return `
      <section class="record-type-section" aria-label="高频记录">
        <div class="record-type-group-title">高频记录</div>
        <div class="record-type-grid">
          ${M().recordTypes.high.map(t => `
            <button class="record-type-btn" data-type="${t.id}" aria-label="${t.label}">
              ${I().icon(t.icon, 24)}
              <span>${t.label}</span>
            </button>
          `).join('')}
        </div>
      </section>
      <section class="record-type-section" aria-label="健康记录">
        <div class="record-type-group-title">健康记录</div>
        <div class="record-type-grid">
          ${M().recordTypes.health.map(t => `
            <button class="record-type-btn" data-type="${t.id}" aria-label="${t.label}">
              ${I().icon(t.icon, 24)}
              <span>${t.label}</span>
            </button>
          `).join('')}
        </div>
      </section>
      <section class="record-type-section" aria-label="生活记录">
        <div class="record-type-group-title">生活记录</div>
        <div class="record-type-grid">
          ${M().recordTypes.life.map(t => `
            <button class="record-type-btn" data-type="${t.id}" aria-label="${t.label}">
              ${I().icon(t.icon, 24)}
              <span>${t.label}</span>
            </button>
          `).join('')}
        </div>
      </section>
    `;
  }

  function renderContent(contentEl) {
    if (!aiEnabled) {
      contentEl.innerHTML += `
        <div class="offline-banner show" style="position:static;margin-bottom:12px;border-radius:8px;">
          ${I().icon('ai', 14)} AI 管家暂不可用，手动记录仍可使用。
        </div>
      `;
    }
    contentEl.innerHTML += renderRecordGroups(contentEl);

    // Bind record type clicks
    contentEl.querySelectorAll('.record-type-btn').forEach(btn => {
      btn.addEventListener('click', () => {
        const type = btn.dataset.type;
        global.AppState.navigate('quick-record', { type });
      });
    });
  }

  function renderAIInputSection(contentEl) {
    const section = document.createElement('section');
    section.className = 'ai-input-section';
    section.innerHTML = `
      <label class="ai-input-label" for="ai-natural-input">告诉猫宅管家，猫咪今天发生了什么…</label>
      <div class="ai-input-row">
        <textarea class="ai-input-field" id="ai-natural-input" placeholder="例如：小白早上没怎么吃，下午吐了一次黄色的水…" rows="3" aria-label="AI 自然语言输入"></textarea>
      </div>
      <div class="ai-input-actions" style="margin-top:12px;justify-content:flex-end;">
        <button class="ai-action-btn" data-ai-action="scan" aria-label="病历扫描">${I().icon('medical', 20)}</button>
        <button class="ai-action-btn" data-ai-action="photo" aria-label="拍照">${I().icon('camera', 20)}</button>
        <button class="ai-action-btn" data-ai-action="upload" aria-label="上传图片">${I().icon('upload', 20)}</button>
        <button class="btn-primary" id="ai-parse-btn" style="width:auto;min-height:44px;padding:0 20px;font-size:var(--font-size-body);">
          ${I().icon('ai', 16)} 交给管家整理
        </button>
      </div>
    `;

    contentEl.prepend(section);

    section.querySelector('#ai-parse-btn').addEventListener('click', async () => {
      const input = section.querySelector('#ai-natural-input');
      const text = input.value.trim();
      if (!text) { global.UI.toast('请输入描述', 'error'); return; }
      if (!aiEnabled) { global.UI.toast('AI 暂不可用，已转手动记录', 'info'); global.AppState.navigate('quick-record', { type: 'custom' }); return; }
      // Store input for confirm page
      global.AppState.setState({ aiInput: text, aiLoading: true });
      global.UI.toast('AI 正在整理…', 'info');
      try {
        const res = await global.ApiService.parseAI(text);
        if (res.success) {
          global.AppState.setState({ aiInput: text, aiSession: res.data, aiLoading: false });
          global.AppState.navigate('ai-confirm');
        } else {
          global.UI.toast(res.message || 'AI 解析失败', 'error');
          global.AppState.setState({ aiLoading: false });
        }
      } catch (e) {
        global.UI.toast('AI 服务暂时不可用，可用手动记录', 'error');
        global.AppState.setState({ aiLoading: false });
      }
    });

    section.querySelector('[data-ai-action="scan"]').addEventListener('click', () => global.AppState.navigate('medical-upload'));
    section.querySelector('[data-ai-action="photo"]').addEventListener('click', () => global.UI.toast('拍照入口（演示）', 'info'));
    section.querySelector('[data-ai-action="upload"]').addEventListener('click', () => global.UI.toast('上传入口（演示）', 'info'));
  }

  function renderRecordsPage(headerElArg, contentElArg) {
    renderHeader(headerElArg);
    contentElArg.innerHTML = '';
    renderAIInputSection(contentElArg);
    renderContent(contentElArg);
  }

  global.renderRecordsPage = renderRecordsPage;
})(typeof window !== 'undefined' ? window : global);

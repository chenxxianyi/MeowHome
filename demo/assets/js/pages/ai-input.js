/* AI Input page controller (separate route) */
(function(global) {
  const I = () => global.AppIcons;

  function renderHeader(headerEl) {
    headerEl.innerHTML = `
      <div class="page-header">
        <div style="display:flex;align-items:center;gap:12px;">
          <button class="ai-action-btn" data-action="back" aria-label="返回" style="min-height:36px;min-width:36px;">${I().icon('back', 20)}</button>
          <div class="page-title">AI 自然语言输入</div>
        </div>
      </div>
    `;
    headerEl.querySelector('[data-action="back"]').addEventListener('click', () => {
      global.AppState.navigate('records');
    });
  }

  function renderAIInputPage(headerElArg, contentElArg) {
    renderHeader(headerElArg);
    const aiEnabled = global.AppState.getState().isAIEnabled;

    contentElArg.innerHTML = `
      <section class="ai-input-section">
        <label class="ai-input-label" for="ai-natural-input">告诉猫宅管家，猫咪今天发生了什么…</label>
        <div class="ai-input-row">
          <textarea class="ai-input-field" id="ai-natural-input" placeholder="例如：小白早上没怎么吃，下午吐了一次黄色的水… " rows="4" aria-label="AI 自然语言输入"></textarea>
        </div>
        <div style="display:flex;gap:8px;margin-top:12px;flex-wrap:wrap;">
          <button class="ai-action-btn" id="ai-voice-btn" aria-label="语音输入" ${!aiEnabled ? 'disabled' : ''}>
            ${I().icon('record', 20)}
          </button>
          <button class="ai-action-btn" id="ai-photo-btn" aria-label="拍照">
            ${I().icon('camera', 20)}
          </button>
          <button class="ai-action-btn" id="ai-upload-btn" aria-label="上传图片">
            ${I().icon('upload', 20)}
          </button>
          <button class="ai-action-btn" id="ai-medical-btn" aria-label="病历扫描">
            ${I().icon('medical', 20)}
          </button>
        </div>
        <button class="btn-primary" id="ai-submit-btn" style="margin-top:16px;${!aiEnabled ? 'background:var(--color-border);color:var(--color-text-tertiary);' : ''}">
          ${I().icon('ai', 16)} 交给管家整理
        </button>
        ${!aiEnabled ? '<div style="margin-top:8px;font-size:var(--font-size-assist);color:var(--color-warning);">AI 管家暂不可用，请切换到手动记录。</div>' : ''}
      </section>

      <div style="margin-top:24px;">
        <div class="section-title">快捷记录</div>
        <div class="record-type-grid">
          ${global.Mock.recordTypes.high.map(t => `
            <button class="record-type-btn" data-type="${t.id}">
              ${I().icon(t.icon, 24)}<span>${t.label}</span>
            </button>
          `).join('')}
        </div>
      </div>
    `;

    contentElArg.querySelector('#ai-submit-btn').addEventListener('click', async () => {
      const input = contentElArg.querySelector('#ai-natural-input');
      const text = input.value.trim();
      if (!text) { global.UI.toast('请输入描述', 'error'); return; }
      if (!aiEnabled) { global.UI.toast('AI 暂不可用，已转手动记录', 'info'); global.AppState.navigate('quick-record', { type: 'custom' }); return; }
      global.UI.toast('AI 正在整理…', 'info');
      try {
        const res = await global.ApiService.parseAI(text);
        if (res.success) {
          global.AppState.setState({ aiInput: text, aiSession: res.data });
          global.AppState.navigate('ai-confirm');
        } else {
          global.UI.toast(res.message || 'AI 解析失败', 'error');
        }
      } catch (e) {
        global.UI.toast('AI 服务暂时不可用，可用手动记录', 'error');
        global.AppState.navigate('quick-record', { type: 'custom' });
      }
    });

    contentElArg.querySelectorAll('.record-type-btn').forEach(btn => {
      btn.addEventListener('click', () => {
        global.AppState.navigate('quick-record', { type: btn.dataset.type });
      });
    });

    contentElArg.querySelector('#ai-photo-btn').addEventListener('click', () => { global.UI.toast('拍照入口（演示）', 'info'); });
    contentElArg.querySelector('#ai-upload-btn').addEventListener('click', () => { global.UI.toast('上传入口（演示）', 'info'); });
    contentElArg.querySelector('#ai-medical-btn').addEventListener('click', () => { global.AppState.navigate('medical-upload'); });
    contentElArg.querySelector('#ai-voice-btn').addEventListener('click', () => { global.UI.toast('语音输入（演示）', 'info'); });
  }

  global.renderAIInputPage = renderAIInputPage;
})(typeof window !== 'undefined' ? window : global);

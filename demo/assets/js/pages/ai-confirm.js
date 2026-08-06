/* AI Confirm page controller */
(function(global) {
  const I = () => global.AppIcons;
  const M = () => global.Mock;
  const aiModel = 'mock-ai-v1';

  function renderHeader(headerEl) {
    headerEl.innerHTML = `
      <div class="page-header">
        <div style="display:flex;align-items:center;gap:12px;">
          <button class="ai-action-btn" data-action="back" aria-label="返回" style="min-height:36px;min-width:36px;">${I().icon('back', 20)}</button>
          <div class="page-title">AI 解析确认</div>
        </div>
      </div>
    `;
    headerEl.querySelector('[data-action="back"]').addEventListener('click', () => {
      global.AppState.navigate('records');
    });
  }

  function renderAIConfirmPage(headerElArg, contentElArg) {
    const state = global.AppState.getState();
    const session = state.aiSession || M().aiParseSession;
    const originalInput = state.aiInput || session.originalInput;
    const parsedAt = session.parsedAt || '今天 09:15';
    const recordCount = session.records?.length || 0;

    contentElArg.innerHTML = `
      <div class="ai-original-input">
        <div class="ai-original-label">你的原始输入</div>
        <div class="ai-original-text">${originalInput || session.originalInput}</div>
      </div>

      <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:16px;">
        <span style="font-size:var(--font-size-assist);color:var(--color-text-tertiary);">AI 解析为 ${recordCount} 条记录</span>
        <span style="font-size:var(--font-size-caption);color:var(--color-text-tertiary);">模型 ${aiModel} · ${parsedAt}</span>
      </div>

      <div style="font-size:var(--font-size-assist);color:var(--color-warning);background:var(--color-warning-soft);padding:8px 12px;border-radius:8px;margin-bottom:16px;">
        ⚠ 以下数据未确认，不会写入正式档案。请仔细检查每条记录。
      </div>

      <div id="ai-records-container">
        ${session.records.map((rec, idx) => {
          const cat = M().cats.find(c => c.id === rec.catId);
          const hasConcern = rec.fields.some(f => f.confidence === 'low' || f.confidence === 'medium');
          return `
            <div class="ai-record-card ${hasConcern ? 'has-concern' : ''}" data-record-id="${rec.id}">
              <div class="ai-record-header">
                <div class="ai-record-type">
                  ${I().icon(rec.type || 'food', 18)}
                  记录${idx + 1}：${rec.type === 'feeding' ? '喂食' : rec.type === 'vomit' ? '呕吐' : rec.type === 'medication' ? '用药' : '记录'}
                  <span style="font-size:var(--font-size-caption);color:var(--color-text-tertiary);">(${cat ? cat.name : '未知'})</span>
                </div>
                <button class="ai-record-delete" data-delete="${rec.id}" aria-label="删除此记录">删除</button>
              </div>
              <div>
                ${rec.fields.map(f => `
                  <div class="ai-field">
                    <span class="ai-field-label">${f.key}</span>
                    <span class="ai-field-value ${f.confidence === 'high' ? 'confirmed' : 'unconfirmed'}">
                      ${f.value || '—'}
                      ${f.note ? `<span style="font-size:10px;color:var(--color-warning);">(${f.note})</span>` : ''}
                    </span>
                  </div>
                `).join('')}
              </div>
            </div>
          `;
        }).join('')}
      </div>

      <div class="ai-disclaimer" style="margin-top:16px;">
        AI 整理 · 基于 ${recordCount} 条解析记录 · 不构成医疗诊断
      </div>
    `;

    // Delete handlers
    contentElArg.querySelectorAll('[data-delete]').forEach(btn => {
      btn.addEventListener('click', () => {
        btn.closest('.ai-record-card').remove();
        const count = contentElArg.querySelectorAll('.ai-record-card').length;
        contentElArg.querySelector('div[style*="AI 解析"]').innerHTML = `AI 解析为 ${count} 条记录`;
      });
    });

    // Fixed bottom bar
    const bottomBar = document.createElement('div');
    bottomBar.style.cssText = 'position:fixed;bottom:calc(64px + var(--safe-bottom));left:0;right:0;padding:12px 16px;background:var(--color-bg-surface);border-top:1px solid var(--color-divider);z-index:90;display:flex;gap:12px;';
    bottomBar.innerHTML = `
      <button class="btn-secondary" id="ai-back-btn" style="width:auto;flex:1;margin:0;">返回修改</button>
      <button class="btn-primary" id="ai-confirm-btn" style="flex:2;margin:0;">确认并保存 ${recordCount} 条记录</button>
    `;
    document.body.appendChild(bottomBar);

    bottomBar.querySelector('#ai-back-btn').addEventListener('click', () => global.AppState.navigate('records'));
    bottomBar.querySelector('#ai-confirm-btn').addEventListener('click', async () => {
      const cards = contentElArg.querySelectorAll('.ai-record-card');
      const records = [];
      cards.forEach(card => {
        const id = card.dataset.recordId;
        records.push({ id, type: card.classList.contains('has-concern') ? 'confirmed_warn' : 'confirmed' });
      });
      try {
        const res = await global.ApiService.confirmAI(records);
        if (res.success) {
          global.UI.toast(`已确认保存 ${records.length} 条记录`, 'success');
          global.AppState.setState({ aiSession: null, aiInput: null });
          setTimeout(() => global.AppState.navigate('today'), 800);
        }
      } catch (e) {
        global.UI.toast('确认失败，请重试', 'error');
      }
    });
  }

  global.renderAIConfirmPage = renderAIConfirmPage;
})(typeof window !== 'undefined' ? window : global);

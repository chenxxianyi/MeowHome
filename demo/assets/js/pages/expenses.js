/* Expenses page controller */
(function(global) {
  const I = () => global.AppIcons;
  const M = () => global.Mock;

  function renderHeader(headerEl) {
    headerEl.innerHTML = `
      <div class="page-header">
        <div style="display:flex;align-items:center;gap:12px;">
          <button class="ai-action-btn" data-action="back" aria-label="返回" style="min-height:36px;min-width:36px;">${I().icon('back', 20)}</button>
          <div class="page-title">养猫支出</div>
        </div>
      </div>
    `;
    headerEl.querySelector('[data-action="back"]').addEventListener('click', () => {
      global.AppState.navigate('family');
    });
  }

  function renderExpensesPage(headerElArg, contentElArg) {
    renderHeader(headerElArg);
    const total = M().expenses.reduce((sum, e) => sum + e.amount, 0);

    contentElArg.innerHTML = `
      <div class="status-panel">
        <div class="status-panel-title">本月支出</div>
        <div style="font-size:var(--font-size-metric);font-weight:600;color:var(--color-text-primary);">¥${total}</div>
        <div style="font-size:var(--font-size-assist);color:var(--color-text-tertiary);margin-top:4px;">共 ${M().expenses.length} 笔记录</div>
      </div>

      <section>
        <div class="section-title">支出明细</div>
        <div class="reminder-list">
          ${M().expenses.map(e => {
            const cat = e.catId === 'both' ? '两只猫' : (M().cats.find(c => c.id === e.catId)?.name || '');
            return `
              <div class="reminder-item">
                <div class="reminder-left">
                  <div class="reminder-icon wallet">${I().icon('wallet', 18)}</div>
                  <div class="reminder-text">
                    <div class="reminder-title">${e.label}</div>
                    <div class="reminder-subtitle">${e.date} · ${e.category} · ${cat}</div>
                  </div>
                </div>
                <div style="font-weight:600;color:var(--color-text-primary);font-size:var(--font-size-body);">¥${e.amount}</div>
              </div>
            `;
          }).join('')}
        </div>
      </section>
    `;
  }

  global.renderExpensesPage = renderExpensesPage;
})(typeof window !== 'undefined' ? window : global);

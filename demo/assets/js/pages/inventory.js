/* Inventory page controller */
(function(global) {
  const I = () => global.AppIcons;
  const M = () => global.Mock;

  function renderHeader(headerEl) {
    headerEl.innerHTML = `
      <div class="page-header">
        <div style="display:flex;align-items:center;gap:12px;">
          <button class="ai-action-btn" data-action="back" aria-label="返回" style="min-height:36px;min-width:36px;">${I().icon('back', 20)}</button>
          <div class="page-title">库存管理</div>
        </div>
      </div>
    `;
    headerEl.querySelector('[data-action="back"]').addEventListener('click', () => {
      global.AppState.navigate('family');
    });
  }

  function statusLabel(status) {
    const map = {
      low: '低库存',
      ok: '正常',
      expired: '已过期'
    };
    const cls = { low: 'warning', ok: 'normal', expired: 'danger' }[status];
    return `<span class="cat-status ${cls}">${map[status]}</span>`;
  }

  function renderInventoryPage(headerElArg, contentElArg) {
    renderHeader(headerElArg);

    const low = M().inventory.filter(i => i.status === 'low' || i.status === 'expired');
    const ok = M().inventory.filter(i => i.status === 'ok');

    contentElArg.innerHTML = `
      ${low.length ? `
        <section style="margin-bottom:24px;">
          <div class="section-title">需关注</div>
          <div class="reminder-list">
            ${low.map(item => `
              <div class="reminder-item">
                <div class="reminder-left">
                  <div class="reminder-icon ${item.status === 'low' ? 'inventory' : 'inventory'}">${I().icon('alertTriangle', 18)}</div>
                  <div class="reminder-text">
                    <div class="reminder-title">${item.name}</div>
                    <div class="reminder-subtitle">
                      ${item.status === 'low' ? `预计剩余 ${item.estimatedDays} 天` : item.status === 'expired' ? `已过期：${item.expiry}` : ''}
                    </div>
                  </div>
                </div>
                ${statusLabel(item.status)}
              </div>
            `).join('')}
          </div>
        </section>
      ` : ''}

      <section>
        <div class="section-title">全部库存</div>
        <div class="reminder-list">
          ${M().inventory.map(item => `
            <div class="reminder-item">
              <div class="reminder-left">
                <div class="reminder-icon inventory">${I().icon(item.category === '药品' ? 'medication' : item.category === '猫砂' ? 'inbox' : 'food', 18)}</div>
                <div class="reminder-text">
                  <div class="reminder-title">${item.name}</div>
                  <div class="reminder-subtitle">${item.quantity} ${item.unit} · ${item.category}${item.expiry ? ' · 到期 ' + item.expiry : ''}</div>
                </div>
              </div>
            </div>
          `).join('')}
        </div>
      </section>
    `;
  }

  global.renderInventoryPage = renderInventoryPage;
})(typeof window !== 'undefined' ? window : global);

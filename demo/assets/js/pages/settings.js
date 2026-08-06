/* Settings page controller */
(function(global) {
  const I = () => global.AppIcons;

  function renderHeader(headerEl) {
    headerEl.innerHTML = `<div class="page-header"><div class="page-title">设置</div></div>`;
  }

  function renderSettingsPage(headerElArg, contentElArg) {
    renderHeader(headerElArg);
    contentElArg.innerHTML = `
      <div class="family-section">
        <div class="section-title">通用</div>
        <div class="family-list">
          <div class="family-item"><div class="family-item-icon">${I().icon('ai',20)}</div><div class="family-item-content"><div class="family-item-title">AI 管家</div><div class="family-item-desc">启用/禁用 AI 功能</div></div><span class="family-item-arrow">${I().icon('chevronRight',20)}</span></div>
          <div class="family-item"><div class="family-item-icon">${I().icon('eye',20)}</div><div class="family-item-content"><div class="family-item-title">隐私设置</div><div class="family-item-desc">管理数据可见性</div></div><span class="family-item-arrow">${I().icon('chevronRight',20)}</span></div>
          <div class="family-item"><div class="family-item-icon">${I().icon('export',20)}</div><div class="family-item-content"><div class="family-item-title">数据导出</div><div class="family-item-desc">导出 JSON 或 CSV</div></div><span class="family-item-arrow">${I().icon('chevronRight',20)}</span></div>
          <div class="family-item"><div class="family-item-icon">${I().icon('import',20)}</div><div class="family-item-content"><div class="family-item-title">数据导入</div><div class="family-item-desc">从备份恢复</div></div><span class="family-item-arrow">${I().icon('chevronRight',20)}</span></div>
          <div class="family-item"><div class="family-item-icon">${I().icon('refresh',20)}</div><div class="family-item-content"><div class="family-item-title">备份与恢复</div><div class="family-item-desc">本地备份管理</div></div><span class="family-item-arrow">${I().icon('chevronRight',20)}</span></div>
          <div class="family-item"><div class="family-item-icon">${I().icon('info',20)}</div><div class="family-item-content"><div class="family-item-title">关于猫宅</div><div class="family-item-desc">v0.1.0-demo · 开发数据</div></div><span class="family-item-arrow">${I().icon('chevronRight',20)}</span></div>
        </div>
      </div>
      <div style="margin-top:32px;padding:16px;background:var(--color-bg-subtle);border-radius:12px;">
        <div style="font-size:var(--font-size-assist);color:var(--color-text-tertiary);">当前网络状态</div>
        <div style="font-size:var(--font-size-body);color:${navigator.onLine ? 'var(--color-success)' : 'var(--color-danger)'};margin-top:4px;">
          ${navigator.onLine ? '✓ 在线' : '✗ 离线'}
        </div>
      </div>
    `;
  }

  global.renderSettingsPage = renderSettingsPage;
})(typeof window !== 'undefined' ? window : global);

/* Moments timeline page controller */
(function(global) {
  const I = () => global.AppIcons;
  const M = () => global.Mock;

  function renderHeader(headerEl) {
    headerEl.innerHTML = `<div class="page-header"><div class="page-title">时光</div></div>`;
  }

  function eventCard(ev) {
    const iconMap = {
      photo: 'photo',
      interaction: 'users',
      milestone: 'calendar',
      medical: 'medical'
    };
    const typeLabel = {
      photo: '照片',
      interaction: '双猫互动',
      milestone: '成长事件',
      medical: '医疗'
    };
    const catName = ev.catId === 'both' ? '两只猫' : (M().cats.find(c => c.id === ev.catId)?.name || '');
    return `
      <div class="timeline-item">
        <div class="timeline-dot">${I().icon(iconMap[ev.type] || 'spotlight', 18)}</div>
        <div class="timeline-content">
          <div class="timeline-date">${ev.date} · ${typeLabel[ev.type] || ev.type}</div>
          <div class="timeline-title">${ev.title}</div>
          <div class="timeline-body">${ev.body} · ${catName}${ev.images ? ` · ${ev.images}张照片` : ''}</div>
          <div class="timeline-image" style="display:none;"></div>
        </div>
      </div>
    `;
  }

  function renderMomentsPage(headerElArg, contentElArg) {
    renderHeader(headerElArg);

    // AI month review
    contentElArg.innerHTML = `
      <section class="ai-summary" aria-label="月度回顾">
        ${I().icon('ai', 12)} <span class="ai-badge">AI 月度回顾</span>
        <div class="ai-summary-body" style="font-size:var(--font-size-body);">
          两只猫共同记录28天，小白体重保持稳定，小橘本月最喜欢在窗边睡觉，共保存36张照片。
        </div>
        <div class="ai-meta"><span>生成于 07-31 20:00</span></div>
        <div class="ai-disclaimer">AI 整理，不构成医疗诊断。</div>
      </section>

      <div style="margin-top:24px;">
        ${M().moments.map(m => `
          <div class="timeline-month">
            <div class="timeline-month-title">${m.month}</div>
            ${m.events.map(eventCard).join('')}
          </div>
        `).join('')}
      </div>
    `;
  }

  global.renderMomentsPage = renderMomentsPage;
})(typeof window !== 'undefined' ? window : global);

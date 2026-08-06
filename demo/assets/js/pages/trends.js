/* Trends page controller */
(function(global) {
  const I = () => global.AppIcons;
  const M = () => global.Mock;

  function renderHeader(headerEl, params) {
    const catId = params.catId || 'cat-whit';
    const cat = M().cats.find(c => c.id === catId) || M().cats[0];
    headerEl.innerHTML = `
      <div class="page-header">
        <div style="display:flex;align-items:center;gap:12px;">
          <button class="ai-action-btn" data-action="back" aria-label="返回" style="min-height:36px;min-width:36px;">${I().icon('back', 20)}</button>
          <div class="page-title">${cat.name} · 健康趋势</div>
        </div>
      </div>
    `;
    headerEl.querySelector('[data-action="back"]').addEventListener('click', () => {
      global.AppState.navigate('cat-detail', { catId });
    });
  }

  function renderTrendsPage(headerElArg, contentElArg, params) {
    const catId = params.catId || global.AppState.getState().currentCat;
    const cat = M().cats.find(c => c.id === catId) || M().cats[0];
    renderHeader(headerElArg, params);

    contentElArg.innerHTML = `
      <div class="chart-container">
        <div class="chart-header">
          <div class="chart-title">${cat.name} 体重变化</div>
          <div class="chart-range">
            <button class="chart-range-btn" data-range="7">7天</button>
            <button class="chart-range-btn active" data-range="30">30天</button>
            <button class="chart-range-btn" data-range="90">90天</button>
          </div>
        </div>
        <div class="chart-canvas" id="trend-chart" aria-label="体重趋势图表">
          <div class="chart-empty">
            ${I().icon('trend', 40)}
            <span style="font-size:var(--font-size-assist);">加载趋势数据中…</span>
          </div>
        </div>
        <div style="font-size:var(--font-size-caption);color:var(--color-text-tertiary);margin-top:8px;">数据覆盖率：30/30 天</div>
      </div>

      <div class="chart-container">
        <div class="chart-header">
          <div class="chart-title">每日食量与饮水</div>
        </div>
        <div class="chart-canvas" id="diet-chart">
          <div class="chart-empty">
            ${I().icon('trend', 40)}
            <span style="font-size:var(--font-size-assist);">加载数据中…</span>
          </div>
        </div>
      </div>

      <section aria-label="关联事件">
        <div class="section-title">关联事件</div>
        <div class="timeline-month">
          <div class="timeline-month-title">近期事件</div>
          ${M().trends[catId]?.filter(d => d.events?.length).slice(0, 5).map(d => `
            <div class="timeline-item">
              <div class="timeline-dot">${I().icon('calendar', 18)}</div>
              <div class="timeline-content">
                <div class="timeline-date">${d.date}</div>
                <div class="timeline-title">${d.events.map(e => e.label).join(' · ')}</div>
              </div>
            </div>
          `).join('') || '<div class="empty-state" style="padding:24px 0;"><div class="empty-state-desc">暂无关联事件</div></div>'}
        </div>
      </section>
    `;

    // Range buttons
    contentElArg.querySelectorAll('.chart-range-btn').forEach(btn => {
      btn.addEventListener('click', () => {
        contentElArg.querySelectorAll('.chart-range-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        global.UI.toast('已切换至 ' + btn.dataset.range + ' 天视图', 'info');
      });
    });
  }

  global.renderTrendsPage = renderTrendsPage;
})(typeof window !== 'undefined' ? window : global);

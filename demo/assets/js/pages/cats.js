/* Cats list page controller */
(function(global) {
  const I = () => global.AppIcons;
  const M = () => global.Mock;

  function renderHeader(headerEl) {
    headerEl.innerHTML = `<div class="page-header"><div class="page-title">猫咪</div></div>`;
  }

  function catCard(cat) {
    const status = M().todayStatus[cat.id] || {};
    const hasVomit = status.vomit?.state === 'danger';
    const hasMedWarn = status.medication?.state === 'warn';
    const hasIssue = hasVomit || hasMedWarn;
    const statusClass = hasIssue ? 'danger' : 'normal';
    const statusText = hasVomit ? '⚠ 需关注' : hasMedWarn ? 'ℹ 需服药' : '今日状态正常';

    return `
      <a class="cat-list-item" href="#cat-detail?catId=${cat.id}" aria-label="查看 ${cat.name} 详情">
        <img class="cat-avatar" src="${cat.avatar}" alt="${cat.name} 头像">
        <div class="cat-info">
          <div class="cat-name">${cat.name}</div>
          <div class="cat-meta">${cat.age}岁 · ${cat.gender === 'female' ? '母' : '公'} · ${cat.breed}${cat.neutered ? ' · 已绝育' : ''}</div>
          <div class="cat-status ${statusClass}">${statusText}</div>
        </div>
        ${hasIssue ? `<span style="color:var(--color-danger);font-size:20px;">!</span>` : ''}
        <span class="family-item-arrow">${I().icon('chevronRight', 20)}</span>
      </a>
    `;
  }

  function renderContent(contentEl) {
    const cats = M().cats;
    contentEl.innerHTML = `
      <div style="margin-bottom:16px;">
        <div class="section-title">已添加的猫咪</div>
        ${cats.map(cat => catCard(cat)).join('')}
      </div>
      <button class="btn-primary" id="add-cat-btn" style="margin-top:16px;">添加猫咪</button>
    `;
    contentEl.querySelector('#add-cat-btn').addEventListener('click', () => {
      global.AppState.navigate('onboarding');
    });
  }

  function renderCatsPage(headerElArg, contentElArg) {
    renderHeader(headerElArg);
    renderContent(contentElArg);
  }

  global.renderCatsPage = renderCatsPage;
})(typeof window !== 'undefined' ? window : global);

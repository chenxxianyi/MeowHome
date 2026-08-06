/* Family page controller */
(function(global) {
  const I = () => global.AppIcons;
  const M = () => global.Mock;

  function renderHeader(headerEl) {
    headerEl.innerHTML = `
      <div class="page-header">
        <div class="page-title">家庭</div>
        <div class="page-subtitle">${M().family.name}</div>
      </div>
    `;
  }

  function familyItem(icon, title, desc, badge, href) {
    const badgeHtml = badge ? `<span class="inventory-badge ${badge.type}">${badge.text}</span>` : '';
    return `
      <a class="family-item" href="${href || '#'}" ${href ? '' : 'onclick="arguments[0].preventDefault();global.UI.toast(\'功能开发中（演示）\',\'info\');"'} aria-label="${title}">
        <div class="family-item-icon">${I().icon(icon, 20)}</div>
        <div class="family-item-content">
          <div class="family-item-title">${title} ${badgeHtml}</div>
          <div class="family-item-desc">${desc || ''}</div>
        </div>
        <span class="family-item-arrow">${I().icon('chevronRight', 20)}</span>
      </a>
    `;
  }

  function renderFamilyPage(headerElArg, contentElArg) {
    renderHeader(headerElArg);

    const lowInventory = M().inventory.filter(i => i.status === 'low' || i.status === 'expired');
    const lowCount = lowInventory.length;

    contentElArg.innerHTML = `
      <div class="family-section">
        <div class="family-section-title">猫咪管理</div>
        <div class="family-list">
          ${M().cats.map(cat => `
            <a class="family-item" href="#cat-detail?catId=${cat.id}" aria-label="查看 ${cat.name}">
              <div class="family-item-icon"><img src="${cat.avatar}" alt="${cat.name} 头像" style="width:20px;height:20px;border-radius:50%;"></div>
              <div class="family-item-content">
                <div class="family-item-title">${cat.name}</div>
                <div class="family-item-desc">${cat.breed} · ${cat.age}岁</div>
              </div>
              <span class="family-item-arrow">${I().icon('chevronRight', 20)}</span>
            </a>
          `).join('')}
        </div>
      </div>

      <div class="family-section">
        <div class="family-section-title">照顾与库存</div>
        <div class="family-list">
          ${familyItem('inbox', '库存', `${lowCount} 项需关注`, lowCount ? { text: String(lowCount), type: 'warning' } : null, '#inventory')}
          ${familyItem('users', '照顾任务', '查看家庭协作任务', null, '#')}
          ${familyItem('wallet', '养猫支出', '统计本月支出', null, '#expenses')}
          ${familyItem('bell', '提醒设置', '自定义提醒规则', null, '#')}
        </div>
      </div>

      <div class="family-section">
        <div class="family-section-title">设置</div>
        <div class="family-list">
          ${familyItem('ai', 'AI 设置', '管理 AI 管家权限与数据', null, '#')}
          ${familyItem('members', '家庭成员', '管理成员与权限', null, '#')}
          ${familyItem('import', '数据导入导出', '备份与恢复数据', null, '#')}
          ${familyItem('refresh', '备份与恢复', '本地备份管理', null, '#')}
          ${familyItem('eye', '隐私设置', '管理数据可见性', null, '#')}
          ${familyItem('family', '关于猫宅', '版本与说明', null, '#')}
        </div>
      </div>
    `;

    // Handle link to inventory/expenses via hash without page reload
    contentElArg.querySelectorAll('a[href="#inventory"], a[href="#expenses"]').forEach(link => {
      link.addEventListener('click', (e) => {
        e.preventDefault();
        const page = link.getAttribute('href').replace('#', '');
        global.AppState.navigate(page);
      });
    });
  }

  global.renderFamilyPage = renderFamilyPage;
})(typeof window !== 'undefined' ? window : global);

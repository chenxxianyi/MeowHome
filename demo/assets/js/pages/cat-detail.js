/* Cat detail page controller */
(function(global) {
  const I = () => global.AppIcons;
  const M = () => global.Mock;

  function renderHeader(headerEl, params) {
    headerEl.innerHTML = `
      <div class="page-header" style="display:flex;align-items:center;gap:12px;">
        <button class="ai-action-btn" data-action="back" aria-label="返回" style="min-height:36px;min-width:36px;">${I().icon('back', 20)}</button>
        <div class="page-title" id="cat-detail-title">猫咪详情</div>
      </div>
    `;
    headerEl.querySelector('[data-action="back"]').addEventListener('click', () => {
      global.AppState.navigate('cats');
    });
  }

  function catDetailContent(catId) {
    const cat = M().cats.find(c => c.id === catId) || M().cats[0];
    const status = M().todayStatus[catId] || M().todayStatus['cat-whit'];
    const trend = M().trends[catId] || M().trends['cat-whit'];
    const latestWeight = trend[trend.length - 1]?.weight || '—';

    return `
      <div class="cat-detail-header">
        <img class="cat-detail-avatar" src="${cat.avatar}" alt="${cat.name}">
        <div class="cat-detail-name">${cat.name}</div>
        <div class="cat-detail-meta">${cat.age}岁 · ${cat.breed} · ${cat.gender === 'female' ? '母' : '公'}${cat.neutered ? ' · 已绝育' : ''}</div>
        ${cat.diseases?.length ? `<div style="margin-top:8px;font-size:var(--font-size-assist);color:var(--color-danger);">已知疾病：${cat.diseases.join('、')}</div>` : ''}
        ${cat.allergies?.length ? `<div style="font-size:var(--font-size-assist);color:var(--color-warning);">过敏：${cat.allergies.join('、')}</div>` : ''}
      </div>
      <nav class="cat-tabs" role="tablist" aria-label="猫咪详情页签">
        <button class="cat-tab active" role="tab" data-tab="overview" aria-selected="true">概览</button>
        <button class="cat-tab" role="tab" data-tab="records" aria-selected="false">记录</button>
        <button class="cat-tab" role="tab" data-tab="trends" aria-selected="false">趋势</button>
        <button class="cat-tab" role="tab" data-tab="medical" aria-selected="false">医疗</button>
        <button class="cat-tab" role="tab" data-tab="moments" aria-selected="false">时光</button>
      </nav>
      <div id="tab-content">
        ${overviewTab(cat, status, latestWeight)}
      </div>
    `;
  }

  function overviewTab(cat, status, latestWeight) {
    return `
      <section class="status-panel">
        <div class="status-panel-title">健康概览</div>
        <div class="status-row">
          <span class="status-label">${I().icon('weight', 18)} 最近体重</span>
          <span class="status-value normal">${latestWeight} kg</span>
        </div>
        <div class="status-row">
          <span class="status-label">${I().icon('food', 18)} 今日饮食</span>
          <span class="status-value ${status.food?.state || 'normal'}">${status.food?.label || '正常'}</span>
        </div>
        <div class="status-row">
          <span class="status-label">${I().icon('water', 18)} 今日饮水</span>
          <span class="status-value ${status.water?.state || 'normal'}">${status.water?.label || '正常'}</span>
        </div>
        <div class="status-row">
          <span class="status-label">${I().icon('elimination', 18)} 今日排泄</span>
          <span class="status-value ${status.elimination?.state || 'normal'}">${status.elimination?.label || '正常'}</span>
        </div>
        <div class="status-row">
          <span class="status-label">${I().icon('medication', 18)} 当前用药</span>
          <span class="status-value ${status.medication?.state || 'normal'}">${status.medication?.label || '无'}</span>
        </div>
        <div class="status-row">
          <span class="status-label">${I().icon('mental', 18)} 精神状态</span>
          <span class="status-value ${status.mental?.state || 'normal'}">${status.mental?.label || '正常'}</span>
        </div>
      </section>

      <section class="focus-section" style="margin-top:16px;">
        <div class="focus-title">${I().icon('bell', 18)} 下次提醒</div>
        <div style="font-size:var(--font-size-body);color:var(--color-text-secondary);padding:4px 0;">
          ${cat.nextVaccine ? '🛡 疫苗：' + cat.nextVaccine + '<br>' : ''}
          ${cat.nextDeworm ? '🔬 驱虫：' + cat.nextDeworm + '<br>' : ''}
          ${cat.currentMedication ? '💊 当前用药：' + cat.currentMedication : '无'}
        </div>
      </section>

      ${cat.diseases?.length ? `
        <section class="focus-section danger" style="margin-top:16px;">
          <div class="focus-title">${I().icon('alertTriangle', 18)} 已知疾病</div>
          <div class="focus-body">${cat.diseases.join('、')}</div>
        </section>
      ` : ''}

      <div style="margin-top:24px;">
        <button class="btn-secondary" id="tab-trends-btn" style="width:100%;">查看健康趋势</button>
        <button class="btn-secondary" id="tab-medical-btn" style="width:100%;margin-top:8px;">查看病历记录</button>
      </div>
    `;
  }

  function renderCatDetailPage(headerElArg, contentElArg, params) {
    const catId = params.catId || 'cat-whit';
    const cat = M().cats.find(c => c.id === catId) || M().cats[0];
    renderHeader(headerElArg, params);
    document.getElementById('cat-detail-title').textContent = cat.name;
    contentElArg.innerHTML = catDetailContent(catId);

    // Tab switching
    contentElArg.querySelectorAll('.cat-tab').forEach(tab => {
      tab.addEventListener('click', () => {
        contentElArg.querySelectorAll('.cat-tab').forEach(t => {
          t.classList.remove('active');
          t.setAttribute('aria-selected', 'false');
        });
        tab.classList.add('active');
        tab.setAttribute('aria-selected', 'true');
        const t = tab.dataset.tab;
        if (t === 'trends') global.AppState.navigate('trends', { catId });
        else if (t === 'medical') global.AppState.navigate('medical-upload');
        else if (t === 'moments') global.AppState.navigate('moments');
        else if (t === 'records') global.UI.toast('记录列表（演示）', 'info');
        else contentElArg.querySelector('#tab-content').innerHTML = overviewTab(cat, M().todayStatus[catId], M().trends[catId][M().trends[catId].length-1]?.weight || '—');
      });
    });

    contentElArg.querySelector('#tab-trends-btn')?.addEventListener('click', () => global.AppState.navigate('trends', { catId }));
    contentElArg.querySelector('#tab-medical-btn')?.addEventListener('click', () => global.AppState.navigate('medical-upload'));
  }

  global.renderCatDetailPage = renderCatDetailPage;
})(typeof window !== 'undefined' ? window : global);

/* Onboarding page controller */
(function(global) {
  const I = () => global.AppIcons;
  const M = () => global.Mock;
  const state = { step: 1, familyName: '小家的猫宅', cats: [], editingIndex: null };

  const fieldGroups = [
    { key: 'name', label: '名称', type: 'text', placeholder: '如：小白', required: true },
    { key: 'gender', label: '性别', type: 'select', options: ['母', '公'] },
    { key: 'breed', label: '品种', type: 'text', placeholder: '如：中华田园猫' },
    { key: 'birthday', label: '生日', type: 'date' },
    { key: 'neutered', label: '是否绝育', type: 'select', options: ['是', '否'] },
    { key: 'diseases', label: '已知疾病', type: 'text', placeholder: '选填，如：慢性肾病' },
    { key: 'allergies', label: '过敏', type: 'text', placeholder: '选填，如：鸡肉' }
  ];

  function renderHeader(headerEl) {
    const steps = 5;
    headerEl.innerHTML = `
      <div class="page-header">
        <div class="onboarding-steps" aria-label="步骤进度">
          ${Array.from({length: steps}, (_, i) => `
            <div class="step-dot ${i < state.step ? 'active' : ''}"></div>
          `).join('')}
        </div>
      </div>
    `;
  }

  function fieldHtml(field, value) {
    const val = value || '';
    if (field.type === 'select') {
      return `
        <div class="form-group">
          <label class="form-label" for="field-${field.key}">${field.label}${field.required ? ' *' : ''}</label>
          <select class="form-select" id="field-${field.key}" data-field="${field.key}">
            ${field.options.map(o => `<option ${val === o ? 'selected' : ''}>${o}</option>`).join('')}
          </select>
        </div>
      `;
    }
    return `
      <div class="form-group">
        <label class="form-label" for="field-${field.key}">${field.label}${field.required ? ' *' : ''}</label>
        <input class="form-input" type="${field.type}" id="field-${field.key}" data-field="${field.key}" value="${val}" placeholder="${field.placeholder || ''}">
      </div>
    `;
  }

  function addCatForm(cat) {
    cat = cat || {};
    return fieldGroups.map(f => fieldHtml(f, cat[f.key])).join('');
  }

  function renderStep1(contentEl) {
    contentEl.innerHTML = `
      <div class="onboarding-card">
        <div class="onboarding-title">欢迎来到猫宅</div>
        <div class="onboarding-desc">为多猫家庭建立长期、可关联的健康档案。请在独立页面上完成设置。</div>
        <div class="form-group">
          <label class="form-label" for="family-name">家庭名称</label>
          <input class="form-input" id="family-name" value="${state.familyName}" placeholder="如：小家的猫宅" />
        </div>
        <button class="btn-primary" id="step1-next">开始创建家庭</button>
      </div>
    `;
    contentEl.querySelector('#step1-next').addEventListener('click', () => {
      const nameInput = contentEl.querySelector('#family-name');
      state.familyName = nameInput.value.trim() || '小家的猫宅';
      state.step = 2;
      renderStep2(contentEl);
      renderHeader(document.getElementById('page-header'));
    });
  }

  function renderStep2(contentEl) {
    contentEl.innerHTML = `
      <div class="onboarding-card">
        <div class="onboarding-title">添加第一只猫</div>
        <div class="onboarding-desc">先用基本信息建立档案，健康细节可稍后补充。</div>
        <div class="cat-avatar-placeholder" id="avatar-upload" role="button" tabindex="0" aria-label="上传猫咪头像">
          ${I().icon('upload', 36)}
        </div>
        ${addCatForm()}
        <button class="btn-primary" id="step2-next">保存并继续</button>
        <button class="btn-secondary" id="step2-skip" style="margin-top:12px;">跳过（稍后添加）</button>
      </div>
    `;
    contentEl.querySelector('#step2-next').addEventListener('click', () => {
      const cat = readCatForm(contentEl);
      if (!cat.name) { global.UI.toast('请输入猫咪名称', 'error'); return; }
      state.cats.push(cat);
      state.step = 3;
      renderStep3(contentEl);
      renderHeader(document.getElementById('page-header'));
    });
    contentEl.querySelector('#step2-skip').addEventListener('click', () => {
      state.step = 3;
      renderStep3(contentEl);
      renderHeader(document.getElementById('page-header'));
    });
  }

  function readCatForm(container) {
    const cat = { id: 'cat-' + Date.now() };
    fieldGroups.forEach(f => {
      const input = container.querySelector(`[data-field="${f.key}"]`);
      if (input) cat[f.key] = input.value.trim();
    });
    if (cat.name) cat.avatar = 'data:image/svg+xml,%3Csvg xmlns=\'http://www.w3.org/2000/svg\' width=\'80\' height=\'80\'%3E%3Ccircle cx=\'40\' cy=\'40\' r=\'40\' fill=\'%23F4E2D4\'/%3E%3Ctext x=\'40\' y=\'48\' text-anchor=\'middle\' font-size=\'28\' fill=\'%23C58A32\'%3E' + encodeURIComponent(cat.name) + '%3C/text%3E%3C/svg%3E';
    return cat;
  }

  function renderStep3(contentEl) {
    contentEl.innerHTML = `
      <div class="onboarding-card">
        <div class="onboarding-title">是否添加第二只猫？</div>
        <div class="onboarding-desc">多猫家庭可以建立独立档案，避免数据混淆。</div>
        <button class="btn-primary" id="step3-yes">添加第二只猫</button>
        <button class="btn-secondary" id="step3-no" style="margin-top:12px;">只养一只</button>
      </div>
    `;
    contentEl.querySelector('#step3-yes').addEventListener('click', () => {
      state.step = 4;
      renderStep4(contentEl);
      renderHeader(document.getElementById('page-header'));
    });
    contentEl.querySelector('#step3-no').addEventListener('click', () => {
      state.step = 5;
      renderStep5(contentEl);
      renderHeader(document.getElementById('page-header'));
    });
  }

  function renderStep4(contentEl) {
    contentEl.innerHTML = `
      <div class="onboarding-card">
        <div class="onboarding-title">添加第二只猫</div>
        <div class="onboarding-desc">为另一只猫建立独立档案。</div>
        ${addCatForm()}
        <button class="btn-primary" id="step4-next">保存并完成</button>
        <button class="btn-secondary" id="step4-skip" style="margin-top:12px;">跳过</button>
      </div>
    `;
    contentEl.querySelector('#step4-next').addEventListener('click', () => {
      state.cats.push(readCatForm(contentEl));
      state.step = 5;
      renderStep5(contentEl);
      renderHeader(document.getElementById('page-header'));
    });
    contentEl.querySelector('#step4-skip').addEventListener('click', () => {
      state.step = 5;
      renderStep5(contentEl);
      renderHeader(document.getElementById('page-header'));
    });
  }

  function renderStep5(contentEl) {
    contentEl.innerHTML = `
      <div class="onboarding-card" style="text-align:center;">
        <div style="margin:12px 0 16px;">${I().icon('success', 56)}</div>
        <div class="onboarding-title">设置完成 🎉</div>
        <div class="onboarding-desc" style="margin-bottom:8px;">
          家庭「${state.familyName}」已创建<br>
          共 ${state.cats.length} 只猫咪档案
        </div>
        <div style="margin:16px 0;font-size:var(--font-size-assist);color:var(--color-text-tertiary);">
          ${state.cats.map(c => `${c.name}`).join('、') || '（未添加猫咪）'}
        </div>
        <button class="btn-primary" id="finish">进入今日首页</button>
      </div>
    `;
    contentEl.querySelector('#finish').addEventListener('click', () => {
      global.AppState.navigate('today');
    });
  }

  function renderOnboardingPage(headerElArg, contentElArg) {
    renderHeader(headerElArg);
    if (state.step === 1) renderStep1(contentElArg);
    else if (state.step === 2) renderStep2(contentElArg);
    else if (state.step === 3) renderStep3(contentElArg);
    else if (state.step === 4) renderStep4(contentElArg);
    else renderStep5(contentElArg);
  }

  global.renderOnboardingPage = renderOnboardingPage;
})(typeof window !== 'undefined' ? window : global);

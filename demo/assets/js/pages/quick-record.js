/* Quick record form controller */
(function(global) {
  const I = () => global.AppIcons;
  const M = () => global.Mock;

  function renderHeader(headerEl, params) {
    const typeLabel = (M().recordTypes.high.find(t => t.id === params.type) ||
                       M().recordTypes.health.find(t => t.id === params.type) ||
                       M().recordTypes.life.find(t => t.id === params.type) || { label: '记录' }).label;
    headerEl.innerHTML = `
      <div class="page-header">
        <div style="display:flex;align-items:center;gap:12px;">
          <button class="ai-action-btn" data-action="back" aria-label="返回" style="min-height:36px;min-width:36px;">${I().icon('back', 20)}</button>
          <div class="page-title">${typeLabel}记录</div>
        </div>
        <div class="cat-switcher" id="quick-cat-switcher"></div>
      </div>
    `;
    headerEl.querySelector('[data-action="back"]').addEventListener('click', () => {
      global.AppState.navigate('records');
    });
    // Cat switcher
    const switcherEl = headerEl.querySelector('#quick-cat-switcher');
    switcherEl.innerHTML = global.UI.catSwitcher(params.catId || 'cat-whit');
    switcherEl.querySelectorAll('button[data-cat]').forEach(btn => {
      btn.addEventListener('click', () => {
        global.AppState.setState({ currentCat: btn.dataset.cat });
      });
    });
  }

  const forms = {
    feeding: {
      fields: [
        { key: 'time', label: '发生时间', type: 'datetime-local', required: true },
        { key: 'meal', label: '餐次', type: 'select', options: ['早餐', '午餐', '晚餐', '零食'], required: true },
        { key: 'food', label: '食物', type: 'text', placeholder: '如：渴望室内猫粮', required: true },
        { key: 'provided', label: '提供量', type: 'number', placeholder: 'g', required: false },
        { key: 'consumed', label: '实际食量', type: 'number', placeholder: 'g', required: false },
        { key: 'appetite', label: '食欲状态', type: 'select', options: ['正常', '偏低', '很差', '拒食'] },
        { key: 'notes', label: '备注', type: 'textarea', placeholder: '选填' }
      ]
    },
    vomit: {
      fields: [
        { key: 'time', label: '发生时间', type: 'datetime-local', required: true },
        { key: 'count', label: '次数', type: 'number', min: 1, max: 10, required: true },
        { key: 'content', label: '内容物', type: 'select', options: ['黄色液体', '未消化食物', '白色泡沫', '带血', '毛球', '其他'] },
        { key: 'beforeMeal', label: '发生在', type: 'select', options: ['进食前', '进食后', '不确定'] },
        { key: 'mentalState', label: '精神状态', type: 'select', options: ['正常', '偏低', '萎靡'] },
        { key: 'notes', label: '备注', type: 'textarea', placeholder: '选填' }
      ]
    },
    elimination: {
      fields: [
        { key: 'time', label: '发生时间', type: 'datetime-local', required: true },
        { key: 'type', label: '类型', type: 'select', options: ['大便', '小便', '两者'] },
        { key: 'form', label: '粪便形态', type: 'select', options: ['成型', '偏干', '偏软', '稀便', '水样'] },
        { key: 'color', label: '颜色', type: 'select', options: ['棕色', '黄色', '绿色', '黑色', '带血'] },
        { key: 'notes', label: '备注', type: 'textarea', placeholder: '选填' }
      ]
    },
    weight: {
      fields: [
        { key: 'time', label: '称重时间', type: 'datetime-local', required: true },
        { key: 'weight', label: '体重（kg）', type: 'number', step: '0.01', placeholder: '如：4.20', required: true },
        { key: 'notes', label: '备注', type: 'textarea', placeholder: '选填' }
      ]
    },
    medication: {
      fields: [
        { key: 'time', label: '给药时间', type: 'datetime-local', required: true },
        { key: 'medication', label: '药物名称', type: 'text', placeholder: '如：肾上腺素抑制剂', required: true },
        { key: 'dose', label: '剂量', type: 'text', placeholder: '如：1片' },
        { key: 'notes', label: '备注', type: 'textarea', placeholder: '选填' }
      ]
    },
    custom: {
      fields: [
        { key: 'time', label: '时间', type: 'datetime-local', required: true },
        { key: 'content', label: '记录内容', type: 'textarea', placeholder: '请描述猫咪的情况…', required: true },
        { key: 'notes', label: '备注', type: 'textarea', placeholder: '选填' }
      ]
    }
  };

  function buildForm(type, catId) {
    const formDef = forms[type] || forms.custom;
    const cat = (M().cats.find(c => c.id === catId) || M().cats[0]);
    return `
      <div class="record-form" id="quick-record-form" data-type="${type}">
        <div style="margin-bottom:12px;font-size:var(--font-size-assist);color:var(--color-text-tertiary);">
          为 <b style="color:var(--color-text-primary);">${cat.name}</b> 记录
        </div>
        ${formDef.fields.map(f => {
          const req = f.required ? ' <span style="color:var(--color-danger);">*</span>' : '';
          if (f.type === 'textarea') {
            return `
              <div class="form-group">
                <label class="form-label" for="qf-${f.key}">${f.label}${req}</label>
                <textarea class="form-input" id="qf-${f.key}" data-key="${f.key}" rows="3" placeholder="${f.placeholder || ''}" ${f.required ? 'required' : ''}></textarea>
              </div>
            `;
          }
          if (f.type === 'select') {
            return `
              <div class="form-group">
                <label class="form-label" for="qf-${f.key}">${f.label}${req}</label>
                <select class="form-select" id="qf-${f.key}" data-key="${f.key}" ${f.required ? 'required' : ''}>
                  <option value="">请选择</option>
                  ${f.options.map(o => `<option>${o}</option>`).join('')}
                </select>
              </div>
            `;
          }
          return `
            <div class="form-group">
              <label class="form-label" for="qf-${f.key}">${f.label}${req}</label>
              <input class="form-input" type="${f.type}" id="qf-${f.key}" data-key="${f.key}" placeholder="${f.placeholder || ''}" ${f.required ? 'required' : ''} />
            </div>
          `;
        }).join('')}
      </div>
      <div class="record-form collapsed" id="extra-fields">
        <div class="collapse-toggle" data-toggle="extra-fields">
          <span>附加选项（图片、额外备注）</span>
          ${I().icon('chevronDown', 16)}
        </div>
        <div class="record-form-extra">
          <div class="form-group">
            <label class="form-label">图片</label>
            <input type="file" id="qf-image" accept="image/*" capture="environment" aria-label="上传图片">
          </div>
          <div class="form-group">
            <label class="form-label">备注</label>
            <textarea class="form-input" id="qf-extra-notes" rows="2" placeholder="选填"></textarea>
          </div>
        </div>
      </div>
    `;
  }

  function renderQuickRecordPage(headerElArg, contentElArg, params) {
    renderHeader(headerElArg, params);
    const catId = params.catId || global.AppState.getState().currentCat;
    const type = params.type || 'feeding';
    contentElArg.innerHTML = buildForm(type, catId);

    // Collapse toggle
    contentElArg.querySelector('[data-toggle="extra-fields"]').addEventListener('click', function() {
      const form = this.closest('.record-form.collapsed');
      const extra = form.querySelector('.record-form-extra');
      const icon = form.querySelector('[data-toggle] svg');
      if (extra.style.display === 'none') {
        extra.style.display = 'block';
        icon.style.transform = 'rotate(180deg)';
      } else {
        extra.style.display = 'none';
        icon.style.transform = 'rotate(0deg)';
      }
    });

    // Save button (fixed at bottom via CSS)
    const saveBtn = document.createElement('div');
    saveBtn.style.cssText = 'position:fixed;bottom:calc(64px + var(--safe-bottom));left:0;right:0;padding:12px 16px;background:var(--color-bg-surface);border-top:1px solid var(--color-divider);z-index:90;';
    saveBtn.innerHTML = `
      <button class="btn-primary" id="save-record-btn" style="width:100%;margin:0;">保存记录</button>
    `;
    document.body.appendChild(saveBtn);

    saveBtn.querySelector('#save-record-btn').addEventListener('click', async () => {
      const form = contentElArg.querySelector('#quick-record-form');
      const required = form.querySelectorAll('[required]');
      let valid = true;
      required.forEach(input => {
        if (!input.value.trim()) { input.style.borderColor = 'var(--color-danger)'; valid = false; }
        else { input.style.borderColor = ''; }
      });
      if (!valid) { global.UI.toast('请填写必填项', 'error'); return; }

      const record = { type, catId };
      form.querySelectorAll('[data-key]').forEach(input => {
        record[input.dataset.key] = input.value.trim();
      });
      record.extraNotes = (contentElArg.querySelector('#qf-extra-notes') || {}).value || '';
      record.createdAt = new Date().toISOString();
      record.source = 'manual';
      record.status = 'synced';

      try {
        const res = await global.ApiService.saveRecord(record);
        if (res.success) {
          global.UI.toast('记录已保存', 'success');
          setTimeout(() => global.AppState.navigate('today'), 800);
        }
      } catch (e) {
        // Offline fallback: save as draft
        global.AppState.getState().draftRecords.push(record);
        global.UI.toast('已保存为离线草稿', 'info');
        setTimeout(() => global.AppState.navigate('today'), 800);
      }
    });
  }

  global.renderQuickRecordPage = renderQuickRecordPage;
})(typeof window !== 'undefined' ? window : global);

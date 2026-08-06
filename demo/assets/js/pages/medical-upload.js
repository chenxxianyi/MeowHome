/* Medical upload page controller */
(function(global) {
  const I = () => global.AppIcons;
  const M = () => global.Mock;

  function renderHeader(headerEl) {
    headerEl.innerHTML = `
      <div class="page-header">
        <div style="display:flex;align-items:center;gap:12px;">
          <button class="ai-action-btn" data-action="back" aria-label="返回" style="min-height:36px;min-width:36px;">${I().icon('back', 20)}</button>
          <div class="page-title">病历上传</div>
        </div>
      </div>
    `;
    headerEl.querySelector('[data-action="back"]').addEventListener('click', () => {
      global.AppState.navigate('records');
    });
  }

  function renderMedicalUploadPage(headerElArg, contentElArg) {
    renderHeader(headerElArg);

    contentElArg.innerHTML = `
      <div class="upload-zone" id="upload-zone" role="button" tabindex="0" aria-label="上传病历图片">
        ${I().icon('upload', 40)}
        <div class="upload-zone-text">点击或拖拽上传病历图片</div>
        <div class="upload-zone-hint">支持 JPG、PNG，最大 10MB</div>
      </div>
      <div id="ocr-section" style="display:none;">
        <div class="medical-ocr">
          <div class="medical-ocr-title">OCR 识别原文</div>
          <div style="font-size:var(--font-size-body);white-space:pre-wrap;">血常规：白细胞 6.5×10⁹/L（参考 4-10），红细胞 5.2×10¹²/L（参考 5-10），血红蛋白 130g/L（参考 120-160），血小板 250×10⁹/L（参考 100-300）。</div>
        </div>

        <div class="medical-ocr" style="background:var(--color-ai-bg);border-color:var(--color-ai-border);">
          <div class="medical-ocr-title">${I().icon('ai', 12)} AI 整理结果</div>
          <table class="medical-metric-table">
            <thead><tr><th>指标</th><th>结果</th><th>单位</th><th>参考范围</th><th>状态</th></tr></thead>
            <tbody>
              <tr><td>白细胞</td><td>6.5</td><td>×10⁹/L</td><td>4-10</td><td><span class="metric-status-normal">✓ 正常</span></td></tr>
              <tr><td>红细胞</td><td>5.2</td><td>×10¹²/L</td><td>5-10</td><td><span class="metric-status-normal">✓ 正常</span></td></tr>
              <tr><td>血红蛋白</td><td>130</td><td>g/L</td><td>120-160</td><td><span class="metric-status-normal">✓ 正常</span></td></tr>
              <tr><td>血小板</td><td>250</td><td>×10⁹/L</td><td>100-300</td><td><span class="metric-status-normal">✓ 正常</span></td></tr>
            </tbody>
          </table>
        </div>

        <div class="form-group" style="margin-top:16px;">
          <label class="form-label">关联猫咪</label>
          <select class="form-select" id="medical-cat">
            ${M().cats.map(c => `<option value="${c.id}">${c.name}</option>`).join('')}
          </select>
        </div>
        <div class="form-group">
          <label class="form-label">修改字段（如需）</label>
          <textarea class="form-input" rows="3" placeholder="如有修正，请在此说明"></textarea>
        </div>
        <div class="form-group" style="display:flex;align-items:center;gap:8px;">
          <input type="checkbox" id="medical-reminder">
          <label for="medical-reminder">创建复诊提醒</label>
        </div>
        <button class="btn-primary" id="medical-save">保存病历</button>
      </div>
    `;

    // Simulate upload
    const zone = contentElArg.querySelector('#upload-zone');
    zone.addEventListener('click', () => {
      global.UI.toast('正在上传…', 'info');
      setTimeout(() => {
        contentElArg.querySelector('#ocr-section').style.display = 'block';
        zone.style.display = 'none';
        global.UI.toast('识别完成', 'success');
      }, 1500);
    });

    contentElArg.querySelector('#medical-save').addEventListener('click', () => {
      global.UI.toast('病历已保存', 'success');
      setTimeout(() => global.AppState.navigate('today'), 800);
    });
  }

  global.renderMedicalUploadPage = renderMedicalUploadPage;
})(typeof window !== 'undefined' ? window : global);

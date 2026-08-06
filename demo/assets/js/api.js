/* API Service — mock service that mirrors real API structure */
(function(global) {
  const DELAY = 200; // simulate network latency

  function delay(ms) {
    return new Promise(resolve => setTimeout(resolve, ms || DELAY));
  }

  const api = {
    // Family
    async getFamily() {
      await delay();
      return { success: true, data: window.Mock.family };
    },

    // Cats
    async getCats() {
      await delay();
      return { success: true, data: window.Mock.cats };
    },

    async getCat(id) {
      await delay();
      const cat = window.Mock.cats.find(c => c.id === id);
      if (!cat) return { success: false, error: 'CAT_NOT_FOUND', message: '猫咪不存在' };
      return { success: true, data: cat };
    },

    // Trends
    async getTrends(catId, range) {
      await delay();
      const all = window.Mock.trends[catId] || [];
      const days = range === '7' ? 7 : range === '30' ? 30 : range === '90' ? 90 : 30;
      const sliced = all.slice(0, days);
      return { success: true, data: sliced };
    },

    // Today status
    async getTodayStatus(catId) {
      await delay();
      return { success: true, data: window.Mock.todayStatus[catId] || window.Mock.todayStatus['cat-whit'] };
    },

    // Focus items
    async getFocusItems(catId) {
      await delay();
      const items = catId ? window.Mock.focusItems.filter(i => i.catId === catId || i.catId === 'both') : window.Mock.focusItems;
      return { success: true, data: items };
    },

    // AI Summary
    async getAISummary(catId) {
      await delay(500);
      return { success: true, data: { ...window.Mock.aiSummary, catId } };
    },

    // Reminders
    async getReminders(filter) {
      await delay();
      let list = [...window.Mock.reminders];
      if (filter === 'todo') list = list.filter(r => r.state === 'todo');
      if (filter === 'done') list = list.filter(r => r.state === 'done');
      return { success: true, data: list };
    },

    // Moments
    async getMoments() {
      await delay();
      return { success: true, data: window.Mock.moments };
    },

    // Inventory
    async getInventory() {
      await delay();
      return { success: true, data: window.Mock.inventory };
    },

    // Expenses
    async getExpenses() {
      await delay();
      return { success: true, data: window.Mock.expenses };
    },

    // AI Parse (simulate)
    async parseAI(text) {
      await delay(800);
      if (!text || text.trim().length < 3) {
        return { success: false, error: 'INVALID_INPUT', message: '请输入至少3个字符' };
      }
      return { success: true, data: window.Mock.aiParseSession };
    },

    // Confirm AI records (simulate)
    async confirmAI(records) {
      await delay(400);
      return { success: true, data: { saved: records.length, ids: records.map(r => r.id) } };
    },

    // Save record (simulate)
    async saveRecord(record) {
      await delay(300);
      return { success: true, data: { ...record, id: 'rec-' + Date.now(), savedAt: new Date().toISOString() } };
    },

    // Complete reminder
    async completeReminder(id) {
      await delay(200);
      return { success: true, data: { id, state: 'done' } };
    },

    // Save draft (offline)
    async saveDraft(draft) {
      await delay(100);
      return { success: true, data: { ...draft, id: 'draft-' + Date.now(), status: 'local' } };
    },

    // Get drafts
    async getDrafts() {
      await delay(100);
      return { success: true, data: [] };
    }
  };

  global.ApiService = api;
})(typeof window !== 'undefined' ? window : global);

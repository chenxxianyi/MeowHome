/* Mock Data — 猫宅 MeowHome Demo (开发数据) */
(function(global) {
  const utils = {
    todayStr: function() {
      const d = new Date();
      return d.getFullYear() + '-' + String(d.getMonth()+1).padStart(2,'0') + '-' + String(d.getDate()).padStart(2,'0');
    },
    dateStr: function(daysAgo) {
      const d = new Date();
      d.setDate(d.getDate() - daysAgo);
      return d.getFullYear() + '-' + String(d.getMonth()+1).padStart(2,'0') + '-' + String(d.getDate()).padStart(2,'0');
    },
    random: function(min, max) { return Math.round((Math.random() * (max - min) + min) * 10) / 10; }
  };

  const today = utils.todayStr();
  const cats = [
    {
      id: 'cat-whit',
      name: '小白',
      gender: 'female',
      breed: '中华田园猫',
      birthday: '2021-03-15',
      age: 4,
      neutered: true,
      avatar: 'data:image/svg+xml,%3Csvg xmlns=\'http://www.w3.org/2000/svg\' width=\'80\' height=\'80\'%3E%3Ccircle cx=\'40\' cy=\'40\' r=\'40\' fill=\'%23F1ECE5\'/%3E%3Ctext x=\'40\' y=\'48\' text-anchor=\'middle\' font-size=\'28\' fill=\'%239B948C\'%3E%E5%B0%8F%E7%99%BD%3C/text%3E%3C/svg%3E',
      diseases: [],
      allergies: [],
      defaultFeeding: '定时定量',
      currentMedication: null,
      nextVaccine: '2026-09-10',
      nextDeworm: '2026-08-20'
    },
    {
      id: 'cat-oran',
      name: '小橘',
      gender: 'male',
      breed: '中华田园猫',
      birthday: '2020-07-22',
      age: 5,
      neutered: true,
      avatar: 'data:image/svg+xml,%3Csvg xmlns=\'http://www.w3.org/2000/svg\' width=\'80\' height=\'80\'%3E%3Ccircle cx=\'40\' cy=\'40\' r=\'40\' fill=\'%23F4E2D4\'/%3E%3Ctext x=\'40\' y=\'48\' text-anchor=\'middle\' font-size=\'28\' fill=\'%23C58A32\'%3E%E5%B0%8F%E6%A9%98%3C/text%3E%3C/svg%3E',
      diseases: ['慢性肾病'],
      allergies: ['鸡肉'],
      defaultFeeding: '自助',
      currentMedication: '肾上腺素抑制剂',
      nextVaccine: '2026-10-05',
      nextDeworm: '2026-08-15'
    }
  ];

  const family = {
    id: 'fam-001',
    name: '小家的猫宅',
    createdAt: '2024-01-10',
    members: [
      { id: 'mem-1', name: '小明', role: 'owner', avatar: '' },
      { id: 'mem-2', name: '小红', role: 'member', avatar: '' }
    ]
  };

  // 30天趋势数据
  function generateTrends(catId) {
    const data = [];
    const cat = cats.find(c => c.id === catId);
    for (let i = 29; i >= 0; i--) {
      const date = utils.dateStr(i);
      const base = catId === 'cat-whit' ? 4.0 : 4.9;
      const weight = utils.random(base - 0.2, base + 0.2);
      const food = utils.random(60, 90);
      const water = utils.random(80, 150);
      const poop = Math.random() > 0.1 ? 1 : 0;
      const vomit = (catId === 'cat-whit' && i === 0) ? 1 : (Math.random() > 0.95 ? 1 : 0);
      const mental = Math.random() > 0.1 ? 'normal' : 'low';
      data.push({
        date, weight, food, water, poop, vomit, mental,
        events: i === 15 ? [{ type: '换粮', label: '换粮：渴望室内猫粮' }] : (i === 5 ? [{ type: '就诊', label: '血常规检查' }] : [])
      });
    }
    return data;
  }

  const trends = {
    'cat-whit': generateTrends('cat-whit'),
    'cat-oran': generateTrends('cat-oran')
  };

  // 今日状态
  const todayStatus = {
    'cat-whit': {
      food: { amount: 55, expected: 80, unit: 'g', state: 'warn', label: '早餐食量偏低' },
      water: { amount: 120, expected: 150, unit: 'ml', state: 'normal', label: '饮水正常' },
      elimination: { state: 'normal', label: '排便正常' },
      vomit: { count: 1, state: 'danger', label: '下午呕吐1次（黄色液体）' },
      medication: { state: 'none', label: '无需用药' },
      mental: { state: 'normal', label: '精神状态正常' }
    },
    'cat-oran': {
      food: { amount: 75, expected: 80, unit: 'g', state: 'normal', label: '进食正常' },
      water: { amount: 110, expected: 120, unit: 'ml', state: 'normal', label: '饮水正常' },
      elimination: { state: 'normal', label: '排便正常' },
      vomit: { count: 0, state: 'none', label: '无呕吐' },
      medication: { state: 'warn', label: '今晚需服药：肾上腺素抑制剂', time: '20:00' },
      mental: { state: 'normal', label: '精神状态正常' }
    }
  };

  // 关注项
  const focusItems = [
    {
      id: 'f1',
      catId: 'cat-whit',
      type: 'vomit',
      severity: 'warn',
      title: '小白下午呕吐1次',
      body: '黄色液体，无食物残渣。发生在午餐后约2小时。精神正常，建议继续观察。',
      evidence: ['喂食记录 12:30', '呕吐记录 14:45'],
      actions: [
        { label: '查看记录', action: 'view' },
        { label: '继续观察', action: 'observe' },
        { label: '添加记录', action: 'add' },
        { label: '询问管家', action: 'ai' }
      ]
    },
    {
      id: 'f2',
      catId: 'cat-oran',
      type: 'medication',
      severity: 'info',
      title: '小橘今晚需服药',
      body: '肾上腺素抑制剂，20:00 给药。慢性肾病管理用药。',
      evidence: ['用药计划', '病历记录 2025-12-01'],
      actions: [
        { label: '完成服药', action: 'done' },
        { label: '稍后提醒', action: 'later' },
        { label: '查看计划', action: 'plan' }
      ]
    }
  ];

  // AI 每日摘要
  const aiSummary = {
    id: 'ai-daily-' + today,
    generatedAt: today + 'T08:30:00+08:00',
    evidenceCount: 12,
    body: '今日共记录12条数据。小白早餐后食量偏低（55g/80g），下午呕吐1次黄色液体，建议继续观察食欲和排便情况。小橘今日状态稳定，夜间需按时服药。两只猫饮水均正常，排便规律。',
    evidence: [
      { type: 'feeding', catId: 'cat-whit', time: '07:30', content: '早餐食量55g，低于预期' },
      { type: 'vomit', catId: 'cat-whit', time: '14:45', content: '呕吐1次，黄色液体' },
      { type: 'feeding', catId: 'cat-oran', time: '07:00', content: '早餐正常进食75g' },
      { type: 'medication', catId: 'cat-oran', time: '20:00', content: '待服药：肾上腺素抑制剂' },
      { type: 'water', catId: 'cat-whit', time: '全天', content: '饮水120ml，正常' },
      { type: 'water', catId: 'cat-oran', time: '全天', content: '饮水110ml，正常' }
    ]
  };

  // 提醒
  const reminders = [
    { id: 'r1', catId: 'cat-oran', type: 'medication', title: '小橘服药', subtitle: '肾上腺素抑制剂 20:00', time: '20:00', state: 'todo', icon: 'medication' },
    { id: 'r2', catId: 'cat-whit', type: 'checkup', title: '小白复诊', subtitle: '骨科复查 2026-08-15', time: '2026-08-15', state: 'todo', icon: 'checkup' },
    { id: 'r3', catId: 'cat-oran', type: 'deworm', title: '小橘驱虫', subtitle: '下次驱虫 2026-08-20', time: '2026-08-20', state: 'todo', icon: 'deworm' },
    { id: 'r4', catId: 'cat-whit', type: 'vaccine', title: '小白疫苗', subtitle: '三联疫苗 2026-09-10', time: '2026-09-10', state: 'todo', icon: 'vaccine' },
    { id: 'r5', catId: 'both', type: 'inventory', title: '猫砂库存不足', subtitle: '豆腐猫砂预计剩余3天', time: '今日', state: 'todo', icon: 'inventory' },
    { id: 'r6', catId: 'both', type: 'water', title: '饮水机换水', subtitle: '建议今日更换', time: '今日', state: 'todo', icon: 'water' },
    { id: 'r7', catId: 'cat-oran', type: 'medication', title: '小橘服药', subtitle: '肾上腺素抑制剂', time: '昨天 20:00', state: 'done', icon: 'medication' },
    { id: 'r8', catId: 'cat-whit', type: 'weight', title: '小白称重', subtitle: '上周体重4.2kg', time: '2026-08-01', state: 'done', icon: 'weight' }
  ];

  // 时光时间线
  const moments = [
    {
      month: '2026年7月',
      events: [
        { id: 'm1', date: '2026-07-28', type: 'photo', title: '小橘在窗边晒太阳', body: '拍了12张照片', catId: 'cat-oran', images: 12 },
        { id: 'm2', date: '2026-07-25', type: 'interaction', title: '两只猫第一次和平共处', body: '小白和小橘一起睡觉', catId: 'both', images: 5 },
        { id: 'm3', date: '2026-07-20', type: 'milestone', title: '小橘5岁生日', body: '生日快乐！', catId: 'cat-oran', images: 8 },
        { id: 'm4', date: '2026-07-15', type: 'medical', title: '小白体检', body: '血常规正常', catId: 'cat-whit', images: 3 }
      ]
    },
    {
      month: '2026年6月',
      events: [
        { id: 'm5', date: '2026-06-28', type: 'photo', title: '夏日午睡', body: '小橘睡了整整一下午', catId: 'cat-oran', images: 15 },
        { id: 'm6', date: '2026-06-15', type: 'milestone', title: '小白到家一周年', body: '一周年快乐！', catId: 'cat-whit', images: 20 },
        { id: 'm7', date: '2026-06-10', type: 'interaction', title: '双猫互动增加', body: '互相理毛的频率上升', catId: 'both', images: 6 }
      ]
    }
  ];

  // AI 解析会话
  const aiParseSession = {
    id: 'ai-parse-001',
    originalInput: '小白今天早上没怎么吃，只吃了不到一半的猫粮。下午又吐了一次黄色的水。小橘晚上要记得喂药哦。',
    parsedAt: today + 'T09:15:00+08:00',
    model: 'mock-ai-v1',
    records: [
      {
        id: 'rec-1',
        type: 'feeding',
        catId: 'cat-whit',
        fields: [
          { key: 'time', value: '07:30', confidence: 'high' },
          { key: 'food', value: '渴望室内猫粮', confidence: 'medium' },
          { key: 'provided', value: '80g', confidence: 'high' },
          { key: 'consumed', value: '35g', confidence: 'low', note: 'AI推断：不到一半' },
          { key: 'appetite', value: '偏低', confidence: 'medium' },
          { key: 'notes', value: '', confidence: 'none' }
        ]
      },
      {
        id: 'rec-2',
        type: 'vomit',
        catId: 'cat-whit',
        fields: [
          { key: 'time', value: '14:00', confidence: 'medium' },
          { key: 'count', value: '1', confidence: 'high' },
          { key: 'content', value: '黄色液体', confidence: 'high' },
          { key: 'beforeMeal', value: null, confidence: 'none' },
          { key: 'mentalState', value: '正常', confidence: 'low', note: '未明确提及' },
          { key: 'notes', value: '', confidence: 'none' }
        ]
      },
      {
        id: 'rec-3',
        type: 'medication',
        catId: 'cat-oran',
        fields: [
          { key: 'time', value: '20:00', confidence: 'medium', note: '推断为晚间' },
          { key: 'medication', value: '肾上腺素抑制剂', confidence: 'high' },
          { key: 'dose', value: null, confidence: 'none', note: '待确认' },
          { key: 'notes', value: '', confidence: 'none' }
        ]
      }
    ]
  };

  // 记录类型
  const recordTypes = {
    high: [
      { id: 'feeding', label: '喂食', icon: 'food' },
      { id: 'drinking', label: '饮水', icon: 'water' },
      { id: 'elimination', label: '排便', icon: 'elimination' },
      { id: 'vomit', label: '呕吐', icon: 'vomit' },
      { id: 'weight', label: '体重', icon: 'weight' },
      { id: 'medication', label: '用药', icon: 'medication' }
    ],
    health: [
      { id: 'mental', label: '精神状态', icon: 'mental' },
      { id: 'symptom', label: '异常症状', icon: 'alertTriangle' },
      { id: 'visit', label: '就诊', icon: 'checkup' },
      { id: 'vaccine', label: '疫苗', icon: 'vaccine' },
      { id: 'deworm', label: '驱虫', icon: 'deworm' },
      { id: 'food-change', label: '换粮', icon: 'food' }
    ],
    life: [
      { id: 'behavior', label: '行为', icon: 'spotlight' },
      { id: 'interaction', label: '双猫互动', icon: 'users' },
      { id: 'photo', label: '照片', icon: 'photo' },
      { id: 'milestone', label: '成长事件', icon: 'calendar' },
      { id: 'custom', label: '自定义', icon: 'more' }
    ]
  };

  // 库存
  const inventory = [
    { id: 'inv-1', name: '豆腐猫砂', category: '猫砂', quantity: 2, unit: '袋', estimatedDays: 3, status: 'low', expiry: null },
    { id: 'inv-2', name: '渴望室内猫粮', category: '粮食', quantity: 5, unit: 'kg', estimatedDays: 15, status: 'ok', expiry: '2027-03-01' },
    { id: 'inv-3', name: '肾上腺素抑制剂', category: '药品', quantity: 12, unit: '片', estimatedDays: 12, status: 'ok', expiry: '2027-06-01' },
    { id: 'inv-4', name: '驱虫滴剂', category: '药品', quantity: 1, unit: '支', estimatedDays: 30, status: 'ok', expiry: '2026-09-01' },
    { id: 'inv-5', name: '益生菌粉', category: '保健品', quantity: 0, unit: '盒', estimatedDays: 0, status: 'expired', expiry: '2026-07-01' }
  ];

  // 支出
  const expenses = [
    { id: 'exp-1', date: today, amount: 168, category: '粮食', label: '渴望室内猫粮 2kg', catId: 'both' },
    { id: 'exp-2', date: '2026-08-01', amount: 89, category: '猫砂', label: '豆腐猫砂 10L', catId: 'both' },
    { id: 'exp-3', date: '2026-07-20', amount: 320, category: '医疗', label: '小白体检+血常规', catId: 'cat-whit' },
    { id: 'exp-4', date: '2026-07-15', amount: 45, category: '保健品', label: '益生菌粉', catId: 'both' },
    { id: 'exp-5', date: '2026-07-01', amount: 128, category: '药品', label: '驱虫滴剂', catId: 'cat-oran' }
  ];

  global.Mock = {
    utils,
    cats,
    family,
    trends,
    todayStatus,
    focusItems,
    aiSummary,
    reminders,
    moments,
    aiParseSession,
    recordTypes,
    inventory,
    expenses
  };
})(typeof window !== 'undefined' ? window : global);

// Mock 数据 — 提醒、时光、库存、支出、AI、记录类型
import type { AIParseSession, FocusItemData, InventoryItem, Expense, RecordTypeGroup, Reminder, TimelineEvent } from '../types'
import { todayStr } from '../utils/date'

export const mockReminders: Reminder[] = [
  { id: 'r1', catId: 'cat-oran', type: 'medication', title: '小橘服药', subtitle: '肾上腺素抑制剂 20:00', time: '20:00', state: 'todo', icon: 'medication' },
  { id: 'r2', catId: 'cat-whit', type: 'checkup', title: '小白复诊', subtitle: '骨科复查 2026-08-15', time: '2026-08-15', state: 'todo', icon: 'checkup' },
  { id: 'r3', catId: 'cat-oran', type: 'deworm', title: '小橘驱虫', subtitle: '下次驱虫 2026-08-20', time: '2026-08-20', state: 'todo', icon: 'deworm' },
  { id: 'r4', catId: 'cat-whit', type: 'vaccine', title: '小白疫苗', subtitle: '三联疫苗 2026-09-10', time: '2026-09-10', state: 'todo', icon: 'vaccine' },
  { id: 'r5', catId: 'both', type: 'inventory', title: '猫砂库存不足', subtitle: '豆腐猫砂预计剩余3天', time: '今日', state: 'todo', icon: 'inventory' },
  { id: 'r6', catId: 'both', type: 'water', title: '饮水机换水', subtitle: '建议今日更换', time: '今日', state: 'todo', icon: 'water' },
  { id: 'r7', catId: 'cat-oran', type: 'medication', title: '小橘服药', subtitle: '肾上腺素抑制剂', time: '昨天 20:00', state: 'done', icon: 'medication' },
  { id: 'r8', catId: 'cat-whit', type: 'weight', title: '小白称重', subtitle: '上周体重4.2kg', time: '2026-08-01', state: 'done', icon: 'weight' }
]

export const mockMoments: TimelineEvent[] = [
  {
    id: 'm1', date: '2026-07-28', type: 'photo', title: '小橘在窗边晒太阳', body: '拍了12张照片', catId: 'cat-oran', images: 12
  },
  {
    id: 'm2', date: '2026-07-25', type: 'interaction', title: '两只猫第一次和平共处', body: '小白和小橘一起睡觉', catId: 'both', images: 5
  },
  {
    id: 'm3', date: '2026-07-20', type: 'milestone', title: '小橘5岁生日', body: '生日快乐！', catId: 'cat-oran', images: 8
  },
  {
    id: 'm4', date: '2026-07-15', type: 'medical', title: '小白体检', body: '血常规正常', catId: 'cat-whit', images: 3
  },
  {
    id: 'm5', date: '2026-06-28', type: 'photo', title: '夏日午睡', body: '小橘睡了整整一下午', catId: 'cat-oran', images: 15
  },
  {
    id: 'm6', date: '2026-06-15', type: 'milestone', title: '小白到家一周年', body: '一周年快乐！', catId: 'cat-whit', images: 20
  }
]

export const mockInventory: InventoryItem[] = [
  { id: 'inv-1', name: '豆腐猫砂', category: '猫砂', quantity: 2, unit: '袋', estimatedDays: 3, status: 'low', expiry: null },
  { id: 'inv-2', name: '渴望室内猫粮', category: '粮食', quantity: 5, unit: 'kg', estimatedDays: 15, status: 'ok', expiry: '2027-03-01' },
  { id: 'inv-3', name: '肾上腺素抑制剂', category: '药品', quantity: 12, unit: '片', estimatedDays: 12, status: 'ok', expiry: '2027-06-01' },
  { id: 'inv-4', name: '驱虫滴剂', category: '药品', quantity: 1, unit: '支', estimatedDays: 30, status: 'ok', expiry: '2026-09-01' },
  { id: 'inv-5', name: '益生菌粉', category: '保健品', quantity: 0, unit: '盒', estimatedDays: 0, status: 'expired', expiry: '2026-07-01' }
]

export const mockExpenses: Expense[] = [
  { id: 'exp-1', date: todayStr(), amount: 168, category: '粮食', label: '渴望室内猫粮 2kg', catId: 'both' },
  { id: 'exp-2', date: '2026-08-01', amount: 89, category: '猫砂', label: '豆腐猫砂 10L', catId: 'both' },
  { id: 'exp-3', date: '2026-07-20', amount: 320, category: '医疗', label: '小白体检+血常规', catId: 'cat-whit' },
  { id: 'exp-4', date: '2026-07-15', amount: 45, category: '保健品', label: '益生菌粉', catId: 'both' },
  { id: 'exp-5', date: '2026-07-01', amount: 128, category: '药品', label: '驱虫滴剂', catId: 'cat-oran' }
]

export const mockFocusItems: FocusItemData[] = [
  {
    id: 'f1', catId: 'cat-whit', type: 'vomit', severity: 'warn', title: '小白下午呕吐1次',
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
    id: 'f2', catId: 'cat-oran', type: 'medication', severity: 'info', title: '小橘今晚需服药',
    body: '肾上腺素抑制剂，20:00 给药。慢性肾病管理用药。',
    evidence: ['用药计划', '病历记录 2025-12-01'],
    actions: [
      { label: '完成服药', action: 'done' },
      { label: '稍后提醒', action: 'later' },
      { label: '查看计划', action: 'plan' }
    ]
  }
]

export const mockRecordTypes: RecordTypeGroup = {
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
}

export const mockAIParseSession: AIParseSession = {
  id: 'ai-parse-001',
  originalInput: '小白今天早上没怎么吃，只吃了不到一半的猫粮。下午又吐了一次黄色的水。小橘晚上要记得喂药哦。',
  parsedAt: todayStr() + 'T09:15:00+08:00',
  model: 'mock-ai-v1',
  records: [
    {
      id: 'rec-1', type: 'feeding', catId: 'cat-whit',
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
      id: 'rec-2', type: 'vomit', catId: 'cat-whit',
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
      id: 'rec-3', type: 'medication', catId: 'cat-oran',
      fields: [
        { key: 'time', value: '20:00', confidence: 'medium', note: '推断为晚间' },
        { key: 'medication', value: '肾上腺素抑制剂', confidence: 'high' },
        { key: 'dose', value: null, confidence: 'none', note: '待确认' },
        { key: 'notes', value: '', confidence: 'none' }
      ]
    }
  ]
}

import type { RecordTypeGroup } from '../types'

/** 记录类型是产品配置，不是演示数据；独立于 Web 端 mocks 维护。 */
export const recordTypes: RecordTypeGroup = {
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

// CatsView「添加猫咪」行为测试
//
// 独立成文件：这里需要 mock @/services，而 unit.test.ts 中的
// 「services 降级行为」用例依赖真实实现，两者不能共存于同一模块注册表。
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createRouter, createWebHashHistory } from 'vue-router'

import type { Cat } from '@/types'

const mocks = vi.hoisted(() => ({
  getCats: vi.fn(),
  createCat: vi.fn()
}))

vi.mock('@/services', () => ({
  services: { getCats: mocks.getCats, createCat: mocks.createCat },
  recordTypes: {}
}))

const CAT_A: Cat = {
  id: 'cat-a',
  name: '小白',
  gender: 'female',
  breed: '中华田园猫',
  birthday: '',
  age: 3,
  neutered: false,
  avatar: '',
  diseases: [],
  allergies: [],
  currentMedication: null,
  nextVaccine: null,
  nextDeworm: null
}

const CAT_B: Cat = { ...CAT_A, id: 'cat-b', name: '小橘', gender: 'male' }

async function mountCats() {
  const { default: CatsView } = await import('@/views/cats/CatsView.vue')
  const router = createRouter({
    history: createWebHashHistory(),
    routes: [{ path: '/', component: { template: '<div />' } }]
  })
  await router.push('/')
  await router.isReady()

  const wrapper = mount(CatsView, {
    global: {
      plugins: [router],
      // 外壳与断言无关，替换为纯 slot 容器以隔离视图逻辑
      stubs: { AppShell: { template: '<div><slot /></div>' } }
    }
  })
  await flushPromises()
  return wrapper
}

/** 页面上的「添加猫咪」按钮（抽屉常驻 DOM，取文档顺序第一个 .btn-primary）。 */
function addButton(w: Awaited<ReturnType<typeof mountCats>>) {
  return w.findAll('button.btn-primary')[0]
}

function saveButton(w: Awaited<ReturnType<typeof mountCats>>) {
  return w.find('.sheet-actions .btn-primary')
}

/** 抽屉是否可见。复用全局 .sheet-overlay/.sheet-content，用 .show 切换显隐。 */
function isOpen(w: Awaited<ReturnType<typeof mountCats>>) {
  return w.find('.sheet-overlay').classes().includes('show')
}

/** 名字输入框（表单区第一个文本框）。 */
function nameField(w: Awaited<ReturnType<typeof mountCats>>) {
  return w.find('.sheet-body input[type="text"]')
}

describe('CatsView 添加猫咪', () => {
  beforeEach(() => {
    mocks.getCats.mockReset().mockResolvedValue({ success: true, data: [CAT_A] })
    mocks.createCat.mockReset().mockResolvedValue({ success: true, data: CAT_B })
  })

  it('初始抽屉关闭，点击「添加猫咪」后打开', async () => {
    const w = await mountCats()
    expect(isOpen(w)).toBe(false)

    await addButton(w).trigger('click')
    expect(isOpen(w)).toBe(true)
    expect(w.find('#add-cat-title').text()).toBe('添加猫咪')
  })

  it('名字为空时保存按钮禁用', async () => {
    const w = await mountCats()
    await addButton(w).trigger('click')

    expect(saveButton(w).attributes('disabled')).toBeDefined()
  })

  it('填写名字后解除禁用，提交会调用 createCat 并刷新列表', async () => {
    const w = await mountCats()
    await addButton(w).trigger('click')
    await nameField(w).setValue('小橘')

    expect(saveButton(w).attributes('disabled')).toBeUndefined()

    // 提交后重新拉取，服务端返回两只猫
    mocks.getCats.mockResolvedValue({ success: true, data: [CAT_A, CAT_B] })
    await saveButton(w).trigger('click')
    await flushPromises()

    expect(mocks.createCat).toHaveBeenCalledTimes(1)
    expect(mocks.createCat.mock.calls[0][0]).toMatchObject({ name: '小橘', gender: 'unknown' })

    // 抽屉关闭 + 列表刷新（getCats 初次加载一次，提交后再一次）
    expect(isOpen(w)).toBe(false)
    expect(mocks.getCats).toHaveBeenCalledTimes(2)
    expect(w.text()).toContain('小橘')
  })

  it('性别可切换并随提交传出', async () => {
    const w = await mountCats()
    await addButton(w).trigger('click')
    await nameField(w).setValue('小橘')

    const segments = w.findAll('.segment')
    expect(segments.length).toBe(3)
    await segments[1].trigger('click') // 公
    await saveButton(w).trigger('click')
    await flushPromises()

    expect(mocks.createCat.mock.calls[0][0]).toMatchObject({ name: '小橘', gender: 'male' })
  })

  it('后端校验失败时展示错误且抽屉保持打开', async () => {
    mocks.createCat.mockRejectedValueOnce({
      response: { status: 400, data: { code: 'VALIDATION_FAILED', message: 'name is required' } }
    })

    const w = await mountCats()
    await addButton(w).trigger('click')
    await nameField(w).setValue('x')
    await saveButton(w).trigger('click')
    await flushPromises()

    expect(w.find('.form-error').exists()).toBe(true)
    expect(isOpen(w)).toBe(true)
  })

  it('取消按钮关闭抽屉且不提交', async () => {
    const w = await mountCats()
    await addButton(w).trigger('click')
    await nameField(w).setValue('小橘')
    await w.find('.sheet-actions .btn-secondary').trigger('click')

    expect(isOpen(w)).toBe(false)
    expect(mocks.createCat).not.toHaveBeenCalled()
  })

  it('点击遮罩关闭抽屉', async () => {
    const w = await mountCats()
    await addButton(w).trigger('click')
    expect(isOpen(w)).toBe(true)

    await w.find('.sheet-overlay').trigger('click')
    expect(isOpen(w)).toBe(false)
  })

  it('操作区位于滚动区之外，不会被表单内容顶出视野', async () => {
    const w = await mountCats()
    await addButton(w).trigger('click')

    // 结构保证：.sheet-actions 是 .sheet-content 的直接子节点，
    // 滚动只发生在 .sheet-body 内，因此按钮始终可见
    const content = w.find('.sheet-content')
    const actions = w.find('.sheet-actions')
    expect(actions.element.parentElement).toBe(content.element)
    expect(w.find('.sheet-body .sheet-actions').exists()).toBe(false)
  })

  it('无猫咪时展示空状态提示', async () => {
    mocks.getCats.mockResolvedValue({ success: true, data: [] })
    const w = await mountCats()

    expect(w.find('.empty-hint').exists()).toBe(true)
    expect(w.find('.empty-hint').text()).toContain('还没有猫咪')
  })
})

import { describe, it, expect } from 'vitest'

// Test pagination composable logic in isolation.
// Simulates the core logic of usePagination() without Nuxt dependencies.

function normalizePage(page: number): number {
  return page < 1 ? 1 : page
}

function normalizePageSize(pageSize: number): number {
  if (pageSize < 1) return 20
  if (pageSize > 100) return 100
  return pageSize
}

function offset(page: number, pageSize: number): number {
  return (page - 1) * pageSize
}

describe('pagination helpers', () => {
  it('正常页数 1', () => {
    expect(normalizePage(1)).toBe(1)
  })

  it('负数页数归一化为 1', () => {
    expect(normalizePage(-1)).toBe(1)
    expect(normalizePage(0)).toBe(1)
  })

  it('pageSize 在有效范围内', () => {
    expect(normalizePageSize(20)).toBe(20)
    expect(normalizePageSize(1)).toBe(1)
    expect(normalizePageSize(100)).toBe(100)
  })

  it('pageSize 超上限截断', () => {
    expect(normalizePageSize(200)).toBe(100)
    expect(normalizePageSize(1000)).toBe(100)
  })

  it('pageSize 小于 1 回退默认', () => {
    expect(normalizePageSize(0)).toBe(20)
    expect(normalizePageSize(-5)).toBe(20)
  })

  it('offset 计算正确', () => {
    expect(offset(1, 20)).toBe(0)
    expect(offset(2, 20)).toBe(20)
    expect(offset(3, 15)).toBe(30)
  })
})

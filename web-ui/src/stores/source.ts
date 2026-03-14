import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Source } from '@/api/source'
import * as sourceApi from '@/api/source'

export const useSourceStore = defineStore('source', () => {
  const sources = ref<Source[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  const fetchSources = async (): Promise<void> => {
    loading.value = true
    error.value = null
    try {
      sources.value = (await sourceApi.listSources()) ?? []
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch sources'
      throw err
    } finally {
      loading.value = false
    }
  }

  const createSource = async (data: Omit<Source, 'id' | 'created_at' | 'updated_at'>): Promise<Source> => {
    loading.value = true
    error.value = null
    try {
      const source = await sourceApi.createSource(data)
      sources.value.push(source)
      return source
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to create source'
      throw err
    } finally {
      loading.value = false
    }
  }

  const updateSource = async (id: number, data: Omit<Source, 'id' | 'created_at' | 'updated_at'>): Promise<void> => {
    loading.value = true
    error.value = null
    try {
      const updated = await sourceApi.updateSource(id, data)
      const index = sources.value.findIndex(s => s.id === id)
      if (index !== -1) {
        sources.value[index] = updated
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to update source'
      throw err
    } finally {
      loading.value = false
    }
  }

  const deleteSource = async (id: number): Promise<void> => {
    loading.value = true
    error.value = null
    try {
      await sourceApi.deleteSource(id)
      sources.value = sources.value.filter(s => s.id !== id)
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to delete source'
      throw err
    } finally {
      loading.value = false
    }
  }

  const testSource = async (id: number): Promise<any> => {
    try {
      return await sourceApi.testSource(id)
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to test source'
      throw err
    }
  }

  const fetchSource = async (id: number): Promise<any> => {
    loading.value = true
    try {
      return await sourceApi.fetchSource(id)
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch source'
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    sources,
    loading,
    error,
    fetchSources,
    createSource,
    updateSource,
    deleteSource,
    testSource,
    fetchSource
  }
})

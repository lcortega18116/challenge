import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface StockItem {
  ticker: string
  target_from: string
  target_to: string
  company: string
  action: string
  brokerage: string
  rating_from: string
  rating_to: string
  time: string
}

export const useStocksStore = defineStore('stocks', () => {
  const items = ref<StockItem[]>([])
  const loading = ref(false)
  const error = ref('')
  const syncing = ref(false)
  const syncMessage = ref('')

  const API_URL = import.meta.env.VITE_API_URL + '/item' || 'http://localhost:8080/item'
  const SYNC_URL = import.meta.env.VITE_API_URL + '/sync' || 'http://localhost:8080/sync'

  // Cargar datos del localStorage al inicializar
  const loadFromStorage = () => {
    try {
      const stored = localStorage.getItem('stocks_items')
      if (stored) {
        items.value = JSON.parse(stored)
      }
    } catch (err) {
      console.error('Error loading from localStorage:', err)
    }
  }

  // Guardar datos en localStorage
  const saveToStorage = () => {
    try {
      localStorage.setItem('stocks_items', JSON.stringify(items.value))
    } catch (err) {
      console.error('Error saving to localStorage:', err)
    }
  }

  const fetchData = async () => {
    loading.value = true
    error.value = ''

    try {
      const controller = new AbortController()
      const timeout = setTimeout(() => controller.abort(), 8000)

      const response = await fetch(API_URL, {
        method: 'GET',
        headers: {
          'Accept': 'application/json'
        },
        signal: controller.signal
      })

      clearTimeout(timeout)

      if (!response.ok) {
        const errorText = await response.text().catch(() => '')
        throw new Error(`Error HTTP ${response.status}: ${errorText || 'Error al obtener datos'}`)
      }

      const data = await response.json()

      if (!data || !Array.isArray(data.items)) {
        throw new Error('La respuesta de la API no contiene items válidos')
      }

      items.value = data.items
      saveToStorage() // Guardar en localStorage

    } catch (err: any) {
      if (err.name === 'AbortError') {
        error.value = 'La solicitud tardó demasiado y fue cancelada'
      } else {
        error.value = err.message || 'Error desconocido'
      }
    } finally {
      loading.value = false
    }
  }

  const syncData = async () => {
    syncing.value = true
    syncMessage.value = ''
    
    try {
      const response = await fetch(SYNC_URL, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        }
      })
      
      if (!response.ok) {
        throw new Error(`Error al sincronizar: ${response.status}`)
      }
      
      await response.json()
      syncMessage.value = 'Sincronización exitosa'
      
      // Recargar datos después de sincronizar
      setTimeout(() => {
        fetchData()
        syncMessage.value = ''
      }, 1500)
      
    } catch (err: any) {
      syncMessage.value = err instanceof Error ? err.message : 'Error al sincronizar'
    } finally {
      syncing.value = false
    }
  }

  // Cargar datos al inicializar el store
  loadFromStorage()

  return {
    items,
    loading,
    error,
    syncing,
    syncMessage,
    fetchData,
    syncData
  }
})

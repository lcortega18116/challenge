<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useStocksStore, type StockItem } from './stores/stocks'

const stocksStore = useStocksStore()

const searchQuery = ref('')
const sortBy = ref<'ticker' | 'company' | 'date' | 'target'>('date')
const sortOrder = ref<'asc' | 'desc'>('desc')
const selectedItem = ref<StockItem | null>(null)
const currentView = ref<'table' | 'cards'>('table')



const filteredItems = computed(() => {
  let filtered = stocksStore.items.filter(item => {
    const query = searchQuery.value.toLowerCase()
    return (
      item.ticker.toLowerCase().includes(query) ||
      item.company.toLowerCase().includes(query) ||
      item.action.toLowerCase().includes(query)
    )
  })

  // Sort
  filtered.sort((a, b) => {
    let comparison = 0
    
    switch (sortBy.value) {
      case 'ticker':
        comparison = a.ticker.localeCompare(b.ticker)
        break
      case 'company':
        comparison = a.company.localeCompare(b.company)
        break
      case 'date':
        comparison = new Date(a.time).getTime() - new Date(b.time).getTime()

        break
      case 'target':
        const aTarget = parseFloat(a.target_to.replace('$', ''))
        const bTarget = parseFloat(b.target_to.replace('$', ''))
        comparison = aTarget - bTarget
        break
    }
    
    return sortOrder.value === 'asc' ? comparison : -comparison
  })

  return filtered
})

const toggleSort = (field: typeof sortBy.value) => {
  if (sortBy.value === field) {
    sortOrder.value = sortOrder.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortBy.value = field
    sortOrder.value = 'desc'
  }
}

const getTargetChange = (item: StockItem) => {
  const from = parseFloat(item.target_from.replace('$', ''))
  const to = parseFloat(item.target_to.replace('$', ''))
  const change = ((to - from) / from) * 100
  return change.toFixed(2)
}

const openDetails = (item: StockItem) => {
  selectedItem.value = item
}

const closeDetails = () => {
  selectedItem.value = null
}

// Investment Recommendation Algorithm
interface StockScore {
  item: StockItem
  score: number
  reasons: string[]
  risk: 'low' | 'medium' | 'high'
}

const analyzeStocks = computed(() => {
  const scoredStocks: StockScore[] = stocksStore.items.map(item => {
    let score = 0
    const reasons: string[] = []
    
    // 1. Target Price Analysis (40 points max)
    const targetChange = parseFloat(getTargetChange(item))
    if (targetChange > 0) {
      if (targetChange > 20) {
        score += 40
        reasons.push(`Excellent upside potential: ${targetChange.toFixed(1)}% increase`)
      } else if (targetChange > 10) {
        score += 30
        reasons.push(`Strong upside potential: ${targetChange.toFixed(1)}% increase`)
      } else if (targetChange > 5) {
        score += 20
        reasons.push(`Moderate upside potential: ${targetChange.toFixed(1)}% increase`)
      } else {
        score += 10
        reasons.push(`Slight upside potential: ${targetChange.toFixed(1)}% increase`)
      }
    }
    
    // 2. Action Type Analysis (30 points max)
    if (item.action.toLowerCase().includes('raised')) {
      score += 30
      reasons.push('Analysts raised price target')
    } else if (item.action.toLowerCase().includes('reiterated')) {
      score += 15
      reasons.push('Analysts reaffirmed their position')
    }
    
    // 3. Rating Improvement (20 points max)
    const ratingScore: Record<string, number> = {
      'strong buy': 5,
      'buy': 4,
      'outperform': 4,
      'hold': 3,
      'neutral': 3,
      'market perform': 3,
      'underperform': 2,
      'sell': 1
    }
    
    const currentRating = item.rating_to.toLowerCase()
    const previousRating = item.rating_from.toLowerCase()
    
    const currentScore = Object.keys(ratingScore).find(key => currentRating.includes(key))
      ? ratingScore[Object.keys(ratingScore).find(key => currentRating.includes(key))!]
      : 3
    
    const previousScore = Object.keys(ratingScore).find(key => previousRating.includes(key))
      ? ratingScore[Object.keys(ratingScore).find(key => previousRating.includes(key))!]
      : 3
    
    if (currentScore >= 4) {
      score += 20
      reasons.push(`Strong Buy/Outperform rating`)
    } else if (currentScore === 3) {
      score += 10
    }
    
    if (currentScore > previousScore) {
      score += 10
      reasons.push('Rating upgraded')
    }
    
    // 4. Recent Activity Bonus (10 points max)
    const daysSinceUpdate = Math.floor((Date.now() - new Date(item.time).getTime()) / (1000 * 60 * 60 * 24))
    if (daysSinceUpdate <= 7) {
      score += 10
      reasons.push('Recent analyst update (within 7 days)')
    } else if (daysSinceUpdate <= 14) {
      score += 5
      reasons.push('Recent analyst update (within 2 weeks)')
    }
    
    // 5. Target Price Level Analysis
    const targetPrice = parseFloat(item.target_to.replace('$', ''))
    if (targetPrice > 100) {
      reasons.push('High-value stock target')
    } else if (targetPrice < 10) {
      reasons.push('Low-price entry opportunity')
    }
    
    // Calculate Risk Level
    let risk: 'low' | 'medium' | 'high' = 'medium'
    if (targetChange > 30) {
      risk = 'high'
    } else if (targetChange > 15) {
      risk = 'medium'
    } else if (targetChange >= 0) {
      risk = 'low'
    } else {
      risk = 'high'
    }
    
    return {
      item,
      score,
      reasons,
      risk
    }
  })
  
  // Sort by score descending
  return scoredStocks.sort((a, b) => b.score - a.score)
})

const topRecommendations = computed(() => {
  return analyzeStocks.value
    .filter(stock => stock.score >= 50) // Only show stocks with score 50+
    .slice(0, 6) // Top 6 recommendations
})

const getScoreColor = (score: number) => {
  if (score >= 80) return 'text-green-600'
  if (score >= 60) return 'text-blue-600'
  if (score >= 40) return 'text-yellow-600'
  return 'text-gray-600'
}

const getScoreBgColor = (score: number) => {
  if (score >= 80) return 'bg-green-100'
  if (score >= 60) return 'bg-blue-100'
  if (score >= 40) return 'bg-yellow-100'
  return 'bg-gray-100'
}

const getRiskColor = (risk: 'low' | 'medium' | 'high') => {
  if (risk === 'low') return 'bg-green-100 text-green-800'
  if (risk === 'medium') return 'bg-yellow-100 text-yellow-800'
  return 'bg-red-100 text-red-800'
}

onMounted(() => {
  // Si no hay datos en memoria, cargar desde la API
  if (stocksStore.items.length === 0) {
    stocksStore.fetchData()
  }
})
</script>


<template>
  <div class="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100">
    <!-- Header -->
    <header class="bg-white shadow-sm border-b border-gray-200">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <div class="flex items-center justify-between">
          <div>
            <h1 class="text-3xl font-bold text-gray-900">Stock Analyst Ratings</h1>
            <p class="mt-1 text-sm text-gray-500">Real-time stock market analysis and recommendations</p>
          </div>
          <div class="flex items-center gap-3">
            <button 
              @click="stocksStore.syncData"
              :disabled="stocksStore.syncing"
              class="px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors flex items-center gap-2 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <svg v-if="!stocksStore.syncing" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              <svg v-else class="w-4 h-4 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              {{ stocksStore.syncing ? 'Syncing...' : 'Sync' }}
            </button>
            <button 
              @click="stocksStore.fetchData"
              class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors flex items-center gap-2"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              Refresh
            </button>
          </div>
        </div>
        
        <!-- Sync Message -->
        <div v-if="stocksStore.syncMessage" class="mt-4">
          <div :class="[
            'p-3 rounded-lg text-sm',
            stocksStore.syncMessage.includes('Error') || stocksStore.syncMessage.includes('error') 
              ? 'bg-red-100 text-red-700' 
              : 'bg-green-100 text-green-700'
          ]">
            {{ stocksStore.syncMessage }}
          </div>
        </div>
      </div>
    </header>

    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- Investment Recommendations Section -->
      <div v-if="!stocksStore.loading && !stocksStore.error && topRecommendations.length > 0" class="mb-8">
        <div class="bg-gradient-to-r from-blue-600 to-blue-700 rounded-lg shadow-lg p-6 text-white mb-6">
          <div class="flex items-center gap-3 mb-2">
            <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <h2 class="text-2xl font-bold">AI Investment Recommendations</h2>
          </div>
          <p class="text-blue-100">Top stocks based on analyst ratings, price targets, and market momentum</p>
        </div>

        <div class="grid grid-cols-1 lg:grid-cols-2 xl:grid-cols-3 gap-6 mb-8">
          <div
            v-for="(stock, index) in topRecommendations"
            :key="stock.item.ticker + stock.item.time"
            class="bg-white rounded-lg shadow-lg hover:shadow-xl transition-all border-2 border-transparent hover:border-blue-500 cursor-pointer"
            @click="openDetails(stock.item)"
          >
            <div class="p-6">
              <!-- Header -->
              <div class="flex items-start justify-between mb-4">
                <div class="flex items-center gap-3">
                  <div class="flex items-center justify-center w-10 h-10 rounded-full bg-gradient-to-br from-blue-500 to-blue-600 text-white font-bold text-lg">
                    {{ index + 1 }}
                  </div>
                  <div>
                    <h3 class="text-xl font-bold text-gray-900">{{ stock.item.ticker }}</h3>
                    <p class="text-sm text-gray-600">{{ stock.item.company }}</p>
                  </div>
                </div>
                <div class="text-right">
                  <div class="text-xs text-gray-500 mb-1">Investment Score</div>
                  <div class="flex items-center gap-2">
                    <span :class="['text-3xl font-bold', getScoreColor(stock.score)]">
                      {{ stock.score }}
                    </span>
                    <span class="text-gray-400">/100</span>
                  </div>
                </div>
              </div>

              <!-- Score Bar -->
              <div class="mb-4">
                <div class="h-2 bg-gray-200 rounded-full overflow-hidden">
                  <div 
                    :class="['h-full transition-all duration-500', getScoreBgColor(stock.score).replace('100', '500')]"
                    :style="{ width: stock.score + '%' }"
                  ></div>
                </div>
              </div>

              <!-- Metrics -->
              <div class="space-y-3 mb-4">
                <div class="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <span class="text-sm font-medium text-gray-700">Target Price</span>
                  <div class="text-right">
                    <div class="text-xs text-gray-400 line-through">{{ stock.item.target_from }}</div>
                    <div class="text-lg font-bold text-green-600">{{ stock.item.target_to }}</div>
                  </div>
                </div>

                <div class="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <span class="text-sm font-medium text-gray-700">Potential Gain</span>
                  <span class="text-lg font-bold text-green-600">
                    +{{ getTargetChange(stock.item) }}%
                  </span>
                </div>

                <div class="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <span class="text-sm font-medium text-gray-700">Risk Level</span>
                  <span :class="['text-xs px-3 py-1 rounded-full font-semibold', getRiskColor(stock.risk)]">
                    {{ stock.risk.toUpperCase() }}
                  </span>
                </div>

                <div class="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <span class="text-sm font-medium text-gray-700">Rating</span>
                  <span class="text-sm font-semibold text-blue-600">{{ stock.item.rating_to }}</span>
                </div>
              </div>

              <!-- Reasons -->
              <div class="border-t border-gray-200 pt-4">
                <p class="text-xs font-semibold text-gray-700 mb-2 uppercase tracking-wide">Why Invest?</p>
                <ul class="space-y-2">
                  <li v-for="(reason, idx) in stock.reasons.slice(0, 3)" :key="idx" class="flex items-start gap-2">
                    <svg class="w-4 h-4 text-green-500 mt-0.5 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                      <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
                    </svg>
                    <span class="text-xs text-gray-600">{{ reason }}</span>
                  </li>
                </ul>
              </div>

              <!-- Action Button -->
              <button
                @click.stop="openDetails(stock.item)"
                class="mt-4 w-full bg-blue-600 hover:bg-blue-700 text-white font-semibold py-3 px-4 rounded-lg transition-colors flex items-center justify-center gap-2"
              >
                View Full Analysis
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                </svg>
              </button>
            </div>
          </div>
        </div>

        <!-- Disclaimer -->
        <div class="bg-yellow-50 border-l-4 border-yellow-400 p-4 rounded-lg">
          <div class="flex">
            <div class="flex-shrink-0">
              <svg class="h-5 w-5 text-yellow-400" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
              </svg>
            </div>
            <div class="ml-3">
              <p class="text-sm text-yellow-700">
                <strong>Investment Disclaimer:</strong> These recommendations are based on analyst ratings and algorithmic analysis. This is not financial advice. Always conduct your own research and consult with a financial advisor before making investment decisions.
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Loading State -->
      <div v-if="stocksStore.loading" class="text-center py-20">
        <div class="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        <p class="mt-4 text-gray-600">Loading stock data...</p>
      </div>
      
      <!-- Error State -->
      <div v-else-if="stocksStore.error" class="bg-red-50 border-l-4 border-red-500 p-4 rounded-lg">
        <div class="flex">
          <div class="flex-shrink-0">
            <svg class="h-5 w-5 text-red-400" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
            </svg>
          </div>
          <div class="ml-3">
            <p class="text-sm text-red-700">{{ stocksStore.error }}</p>
          </div>
        </div>
      </div>
      
      <!-- Main Content -->
      <div v-else>
        <!-- Controls Bar -->
        <div class="bg-white rounded-lg shadow-sm p-6 mb-6">
          <div class="flex flex-col lg:flex-row gap-4 items-start lg:items-center justify-between">
            <!-- Search -->
            <div class="flex-1 max-w-md">
              <label for="search" class="sr-only">Search</label>
              <div class="relative">
                <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                  <svg class="h-5 w-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                  </svg>
                </div>
                <input
                  id="search"
                  v-model="searchQuery"
                  type="text"
                  class="block w-full pl-10 pr-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition-all"
                  placeholder="Search by ticker, company, or action..."
                />
              </div>
            </div>

            <!-- View Toggle -->
            <div class="flex gap-2 bg-gray-100 p-1 rounded-lg">
              <button
                @click="currentView = 'table'"
                :class="[
                  'px-4 py-2 rounded-md text-sm font-medium transition-colors',
                  currentView === 'table' ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-600 hover:text-gray-900'
                ]"
              >
                Table View
              </button>
              <button
                @click="currentView = 'cards'"
                :class="[
                  'px-4 py-2 rounded-md text-sm font-medium transition-colors',
                  currentView === 'cards' ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-600 hover:text-gray-900'
                ]"
              >
                Card View
              </button>
            </div>

            <!-- Results Count -->
            <div class="text-sm text-gray-600">
              Showing <span class="font-semibold text-gray-900">{{ filteredItems.length }}</span> results
            </div>
          </div>
        </div>

        <!-- Table View -->
        <div v-if="currentView === 'table'" class="bg-white shadow-md rounded-lg overflow-hidden">
          <div class="overflow-x-auto">
            <table class="min-w-full divide-y divide-gray-200">
              <thead class="bg-gray-50">
                <tr>
                  <th 
                    @click="toggleSort('ticker')"
                    class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors"
                  >
                    <div class="flex items-center gap-2">
                      Ticker
                      <svg v-if="sortBy === 'ticker'" class="w-4 h-4" :class="sortOrder === 'asc' ? 'rotate-180' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                      </svg>
                    </div>
                  </th>
                  <th 
                    @click="toggleSort('company')"
                    class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors"
                  >
                    <div class="flex items-center gap-2">
                      Company
                      <svg v-if="sortBy === 'company'" class="w-4 h-4" :class="sortOrder === 'asc' ? 'rotate-180' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                      </svg>
                    </div>
                  </th>
                  <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Action</th>
                  <th 
                    @click="toggleSort('target')"
                    class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors"
                  >
                    <div class="flex items-center gap-2">
                      Target
                      <svg v-if="sortBy === 'target'" class="w-4 h-4" :class="sortOrder === 'asc' ? 'rotate-180' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                      </svg>
                    </div>
                  </th>
                  <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Change</th>
                  <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Rating</th>
                  <th 
                    @click="toggleSort('date')"
                    class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors"
                  >
                    <div class="flex items-center gap-2">
                      Date
                      <svg v-if="sortBy === 'date'" class="w-4 h-4" :class="sortOrder === 'asc' ? 'rotate-180' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                      </svg>
                    </div>
                  </th>
                  <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                </tr>
              </thead>
              <tbody class="bg-white divide-y divide-gray-200">
                <tr v-for="item in filteredItems" :key="item.ticker + item.time" class="hover:bg-gray-50 transition-colors">
                  <td class="px-6 py-4 whitespace-nowrap">
                    <span class="text-sm font-bold text-blue-600">{{ item.ticker }}</span>
                  </td>
                  <td class="px-6 py-4">
                    <span class="text-sm text-gray-900">{{ item.company }}</span>
                  </td>
                  <td class="px-6 py-4">
                    <span class="text-xs px-2 py-1 rounded-full" :class="{
                      'bg-green-100 text-green-800': item.action.includes('raised'),
                      'bg-red-100 text-red-800': item.action.includes('lowered'),
                      'bg-blue-100 text-blue-800': item.action.includes('reiterated')
                    }">
                      {{ item.action }}
                    </span>
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap">
                    <div class="text-sm">
                      <div class="text-gray-500">{{ item.target_from }}</div>
                      <div class="font-medium" :class="parseFloat(item.target_to.replace('$', '')) > parseFloat(item.target_from.replace('$', '')) ? 'text-green-600' : 'text-red-600'">
                        {{ item.target_to }}
                      </div>
                    </div>
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap">
                    <span class="text-sm font-medium" :class="parseFloat(getTargetChange(item)) > 0 ? 'text-green-600' : 'text-red-600'">
                      {{ parseFloat(getTargetChange(item)) > 0 ? '+' : '' }}{{ getTargetChange(item) }}%
                    </span>
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap">
                    <div class="text-xs">
                      <span class="px-2 py-1 bg-gray-100 rounded">{{ item.rating_from }}</span>
                      <span class="mx-1">→</span>
                      <span class="px-2 py-1 bg-blue-100 text-blue-800 rounded">{{ item.rating_to }}</span>
                    </div>
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap">
                    <span class="text-sm text-gray-500">{{ new Date(item.time).toLocaleDateString() }}</span>
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap">
                    <button
                      @click="openDetails(item)"
                      class="text-blue-600 hover:text-blue-800 text-sm font-medium transition-colors"
                    >
                      View Details
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Card View -->
        <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          <div
            v-for="item in filteredItems"
            :key="item.ticker + item.time"
            class="bg-white rounded-lg shadow-md hover:shadow-lg transition-shadow p-6 cursor-pointer"
            @click="openDetails(item)"
          >
            <div class="flex items-start justify-between mb-4">
              <div>
                <h3 class="text-xl font-bold text-blue-600">{{ item.ticker }}</h3>
                <p class="text-sm text-gray-600 mt-1">{{ item.company }}</p>
              </div>
              <span class="text-xs px-3 py-1 rounded-full font-medium" :class="{
                'bg-green-100 text-green-800': item.action.includes('raised'),
                'bg-red-100 text-red-800': item.action.includes('lowered'),
                'bg-blue-100 text-blue-800': item.action.includes('reiterated')
              }">
                {{ item.action.split(' ')[0] }}
              </span>
            </div>

            <div class="space-y-3">
              <div class="flex items-center justify-between">
                <span class="text-sm text-gray-600">Target Price</span>
                <div class="text-right">
                  <div class="text-xs text-gray-400 line-through">{{ item.target_from }}</div>
                  <div class="text-lg font-bold" :class="parseFloat(item.target_to.replace('$', '')) > parseFloat(item.target_from.replace('$', '')) ? 'text-green-600' : 'text-red-600'">
                    {{ item.target_to }}
                  </div>
                </div>
              </div>

              <div class="flex items-center justify-between">
                <span class="text-sm text-gray-600">Change</span>
                <span class="text-sm font-semibold" :class="parseFloat(getTargetChange(item)) > 0 ? 'text-green-600' : 'text-red-600'">
                  {{ parseFloat(getTargetChange(item)) > 0 ? '+' : '' }}{{ getTargetChange(item) }}%
                </span>
              </div>

              <div class="flex items-center justify-between">
                <span class="text-sm text-gray-600">Rating</span>
                <div class="text-xs">
                  <span class="px-2 py-1 bg-gray-100 rounded">{{ item.rating_from }}</span>
                  <span class="mx-1">→</span>
                  <span class="px-2 py-1 bg-blue-100 text-blue-800 rounded">{{ item.rating_to }}</span>
                </div>
              </div>

              <div class="pt-3 border-t border-gray-100">
                <span class="text-xs text-gray-500">{{ new Date(item.time).toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' }) }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- No Results -->
        <div v-if="filteredItems.length === 0" class="text-center py-12 bg-white rounded-lg shadow-sm">
          <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <h3 class="mt-2 text-sm font-medium text-gray-900">No results found</h3>
          <p class="mt-1 text-sm text-gray-500">Try adjusting your search query.</p>
        </div>
      </div>
    </div>

    <!-- Details Modal -->
    <div v-if="selectedItem" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50" @click="closeDetails">
      <div class="bg-white rounded-lg shadow-xl max-w-2xl w-full p-6" @click.stop>
        <div class="flex items-start justify-between mb-6">
          <div>
            <h2 class="text-2xl font-bold text-gray-900">{{ selectedItem.ticker }}</h2>
            <p class="text-gray-600 mt-1">{{ selectedItem.company }}</p>
          </div>
          <button @click="closeDetails" class="text-gray-400 hover:text-gray-600 transition-colors">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div class="space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div class="bg-gray-50 p-4 rounded-lg">
              <p class="text-sm text-gray-600 mb-1">Previous Target</p>
              <p class="text-2xl font-bold text-gray-900">{{ selectedItem.target_from }}</p>
            </div>
            <div class="bg-blue-50 p-4 rounded-lg">
              <p class="text-sm text-gray-600 mb-1">New Target</p>
              <p class="text-2xl font-bold" :class="parseFloat(selectedItem.target_to.replace('$', '')) > parseFloat(selectedItem.target_from.replace('$', '')) ? 'text-green-600' : 'text-red-600'">
                {{ selectedItem.target_to }}
              </p>
            </div>
          </div>

          <div class="bg-gray-50 p-4 rounded-lg">
            <p class="text-sm text-gray-600 mb-1">Target Change</p>
            <p class="text-xl font-semibold" :class="parseFloat(getTargetChange(selectedItem)) > 0 ? 'text-green-600' : 'text-red-600'">
              {{ parseFloat(getTargetChange(selectedItem)) > 0 ? '+' : '' }}{{ getTargetChange(selectedItem) }}%
            </p>
          </div>

          <div class="bg-gray-50 p-4 rounded-lg">
            <p class="text-sm text-gray-600 mb-2">Action</p>
            <span class="inline-flex px-3 py-1 rounded-full text-sm font-medium" :class="{
              'bg-green-100 text-green-800': selectedItem.action.includes('raised'),
              'bg-red-100 text-red-800': selectedItem.action.includes('lowered'),
              'bg-blue-100 text-blue-800': selectedItem.action.includes('reiterated')
            }">
              {{ selectedItem.action }}
            </span>
          </div>

          <div class="grid grid-cols-2 gap-4">
            <div class="bg-gray-50 p-4 rounded-lg">
              <p class="text-sm text-gray-600 mb-1">Previous Rating</p>
              <p class="text-lg font-semibold text-gray-900">{{ selectedItem.rating_from }}</p>
            </div>
            <div class="bg-gray-50 p-4 rounded-lg">
              <p class="text-sm text-gray-600 mb-1">Current Rating</p>
              <p class="text-lg font-semibold text-blue-600">{{ selectedItem.rating_to }}</p>
            </div>
          </div>

          <div class="bg-gray-50 p-4 rounded-lg">
            <p class="text-sm text-gray-600 mb-1">Brokerage</p>
            <p class="text-lg font-semibold text-gray-900">{{ selectedItem.brokerage || 'N/A' }}</p>
          </div>

          <div class="bg-gray-50 p-4 rounded-lg">
            <p class="text-sm text-gray-600 mb-1">Date</p>
            <p class="text-lg font-semibold text-gray-900">
              {{ new Date(selectedItem.time).toLocaleDateString('en-US', { 
                year: 'numeric', 
                month: 'long', 
                day: 'numeric',
                hour: '2-digit',
                minute: '2-digit'
              }) }}
            </p>
          </div>
        </div>

        <div class="mt-6 flex justify-end">
          <button
            @click="closeDetails"
            class="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
<!-- 
<style scoped>
header {
  line-height: 1.5;
}

.logo {
  display: block;
  margin: 0 auto 2rem;
}

@media (min-width: 1024px) {
  header {
    display: flex;
    place-items: center;
    padding-right: calc(var(--section-gap) / 2);
  }

  .logo {
    margin: 0 2rem 0 0;
  }

  header .wrapper {
    display: flex;
    place-items: flex-start;
    flex-wrap: wrap;
  }
}
</style> -->

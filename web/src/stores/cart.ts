import { defineStore } from 'pinia'
import { computed, ref, watch } from 'vue'
import type { Product } from '@/services/catalog'

const STORAGE_KEY = 'marketplace.cart'

export interface CartItem {
  product_id: string
  seller_id: string
  category_id: string
  slug: string
  name: string
  price_cents: number
  currency: 'RUB' | 'USD' | 'EUR'
  stock_quantity: number
  quantity: number
}

function loadFromStorage(): CartItem[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.filter((x) => x && typeof x.product_id === 'string')
  } catch {
    return []
  }
}

export const useCartStore = defineStore('cart', () => {
  const items = ref<CartItem[]>(loadFromStorage())

  watch(
    items,
    (value) => {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(value))
    },
    { deep: true },
  )

  // Sync across tabs.
  if (typeof window !== 'undefined') {
    window.addEventListener('storage', (e) => {
      if (e.key === STORAGE_KEY) {
        items.value = loadFromStorage()
      }
    })
  }

  const count = computed(() => items.value.reduce((sum, i) => sum + i.quantity, 0))
  const subtotalCents = computed(() =>
    items.value.reduce((sum, i) => sum + i.price_cents * i.quantity, 0),
  )
  const isEmpty = computed(() => items.value.length === 0)
  const currency = computed(() => items.value[0]?.currency ?? 'USD')

  // Items grouped by seller for checkout.
  const bySeller = computed(() => {
    const map = new Map<string, CartItem[]>()
    for (const item of items.value) {
      const list = map.get(item.seller_id) ?? []
      list.push(item)
      map.set(item.seller_id, list)
    }
    return map
  })

  function addItem(product: Product, quantity = 1) {
    const existing = items.value.find((i) => i.product_id === product.id)
    const maxQty = Math.max(1, product.stock_quantity)

    if (existing) {
      existing.quantity = Math.min(existing.quantity + quantity, maxQty)
      return
    }

    items.value.push({
      product_id: product.id,
      seller_id: product.seller_id,
      category_id: product.category_id,
      slug: product.slug,
      name: product.name,
      price_cents: product.price_cents,
      currency: product.currency,
      stock_quantity: product.stock_quantity,
      quantity: Math.min(Math.max(1, quantity), maxQty),
    })
  }

  function updateQuantity(productId: string, quantity: number) {
    const item = items.value.find((i) => i.product_id === productId)
    if (!item) return
    if (quantity <= 0) {
      removeItem(productId)
      return
    }
    item.quantity = Math.min(quantity, Math.max(1, item.stock_quantity))
  }

  function removeItem(productId: string) {
    items.value = items.value.filter((i) => i.product_id !== productId)
  }

  function clear() {
    items.value = []
  }

  return {
    items,
    count,
    subtotalCents,
    isEmpty,
    currency,
    bySeller,
    addItem,
    updateQuantity,
    removeItem,
    clear,
  }
})

<template>
  <div class="container products-page">
    <h1>{{ t('products.title') }}</h1>

    <div class="products-layout">
      <CategorySidebar
        :categories="categories"
        :selected-id="selectedCategoryId"
        @select="onSelectCategory"
      />

      <div class="products-main">
        <div class="search-bar">
          <input
            v-model="searchInput"
            type="text"
            class="input"
            :placeholder="t('products.searchPlaceholder')"
            @keyup.enter="onSearch"
          />
          <button class="btn" @click="onSearch">{{ t('common.retry') }}</button>
        </div>

        <div v-if="loading && products.length === 0" class="loading">
          {{ t('products.loading') }}
        </div>

        <div v-else-if="products.length === 0" class="empty">
          {{ t('products.empty') }}
        </div>

        <template v-else>
          <div class="product-grid">
            <ProductCard
              v-for="p in products"
              :key="p.id"
              :product="p"
            />
          </div>

          <Pagination
            :has-more="!!nextCursor"
            :loading="loadingMore"
            :show-end="products.length > 0 && !nextCursor"
            @load-more="onLoadMore"
          />
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import CategorySidebar from '@/components/CategorySidebar.vue'
import ProductCard from '@/components/ProductCard.vue'
import Pagination from '@/components/Pagination.vue'
import { listCategories, listProducts, type Category, type Product } from '@/services/catalog'

const { t } = useI18n()

const categories = ref<Category[]>([])
const products = ref<Product[]>([])
const nextCursor = ref<string | null>(null)
const selectedCategoryId = ref<string | null>(null)
const searchInput = ref('')
const searchQuery = ref('')

const loading = ref(false)
const loadingMore = ref(false)

async function fetchProducts(reset: boolean) {
  if (reset) {
    loading.value = true
    products.value = []
    nextCursor.value = null
  } else {
    loadingMore.value = true
  }

  try {
    const page = await listProducts({
      category_id: selectedCategoryId.value ?? undefined,
      q: searchQuery.value || undefined,
      cursor: reset ? undefined : nextCursor.value ?? undefined,
      limit: 20,
    })

    if (reset) {
      products.value = page.items
    } else {
      products.value = [...products.value, ...page.items]
    }
    nextCursor.value = page.next_cursor ?? null
  } catch (err) {
    console.error('Failed to load products', err)
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

function onSelectCategory(id: string | null) {
  selectedCategoryId.value = id
  fetchProducts(true)
}

function onSearch() {
  searchQuery.value = searchInput.value.trim()
  fetchProducts(true)
}

function onLoadMore() {
  if (nextCursor.value) {
    fetchProducts(false)
  }
}

onMounted(async () => {
  try {
    categories.value = await listCategories()
  } catch (err) {
    console.error('Failed to load categories', err)
  }
  await fetchProducts(true)
})
</script>

<style scoped>
.products-page {
  padding: 16px 0 48px;
}

h1 {
  font-size: 1.75rem;
  margin-bottom: 24px;
}

.products-layout {
  display: grid;
  grid-template-columns: 220px 1fr;
  gap: 32px;
  align-items: start;
}

.products-main {
  min-width: 0;
}

.search-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 24px;
}

.search-bar .input {
  flex: 1;
}

.product-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 20px;
}

.loading,
.empty {
  padding: 48px 0;
  text-align: center;
  color: var(--color-text-muted);
}

@media (max-width: 768px) {
  .products-layout {
    grid-template-columns: 1fr;
    gap: 16px;
  }

  .product-grid {
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: 12px;
  }
}
</style>

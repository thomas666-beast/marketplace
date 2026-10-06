<template>
  <div class="container product-detail-page">
    <router-link to="/products" class="back-link">
      &larr; {{ t('products.backToList') }}
    </router-link>

    <div v-if="loading" class="loading">
      {{ t('common.loading') }}
    </div>

    <div v-else-if="!product" class="not-found">
      <h1>{{ t('products.productNotFound') }}</h1>
      <router-link to="/products" class="btn">
        {{ t('products.backToList') }}
      </router-link>
    </div>

    <div v-else class="product-detail">
      <div class="product-image">
        <span class="placeholder">{{ product.name.charAt(0).toUpperCase() }}</span>
      </div>

      <div class="product-info">
        <h1>{{ product.name }}</h1>
        <p class="price">{{ formattedPrice }}</p>

        <p
          class="stock"
          :class="{ out: product.stock_quantity === 0 }"
        >
          {{ product.stock_quantity > 0 ? t('products.inStock') : t('products.outOfStock') }}
        </p>

        <p v-if="product.description" class="description">
          {{ product.description }}
        </p>

        <div class="actions">
          <button
            class="btn"
            :disabled="product.stock_quantity === 0"
            @click="onAddToCart"
          >
            {{ added ? '✓ ' + t('cart.addedToCart') : t('products.addToCart') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
    import { computed, onMounted, ref, watch } from 'vue'
    import { useRoute } from 'vue-router'
    import { useI18n } from 'vue-i18n'
    import { getProduct, type Product } from '@/services/catalog'
    import { useCartStore } from '@/stores/cart'

    const route = useRoute()
    const { t, locale } = useI18n()
    const cart = useCartStore()

    const product = ref<Product | null>(null)
    const loading = ref(true)
    const added = ref(false)

    const formattedPrice = computed(() => {
    if (!product.value) return ''
    const amount = product.value.price_cents / 100
    try {
        return new Intl.NumberFormat(locale.value, {
        style: 'currency',
        currency: product.value.currency,
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
        }).format(amount)
    } catch {
        return `${amount.toFixed(2)} ${product.value.currency}`
    }
    })

    function onAddToCart() {
    if (!product.value) return
    cart.addItem(product.value, 1)
    added.value = true
    setTimeout(() => {
        added.value = false
    }, 2000)
    }

    async function loadProduct(slug: string) {
    loading.value = true
    product.value = null
    try {
        product.value = await getProduct(slug)
    } catch {
        product.value = null
    } finally {
        loading.value = false
    }
    }

    onMounted(() => loadProduct(route.params.slug as string))
    watch(() => route.params.slug, (slug) => loadProduct(slug as string))
</script>

<style scoped>
.product-detail-page {
  padding: 16px 0 48px;
}

.back-link {
  display: inline-block;
  margin-bottom: 24px;
  color: var(--color-text-muted);
  font-size: 0.95rem;
}

.product-detail {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 48px;
  align-items: start;
}

.product-image {
  aspect-ratio: 1 / 1;
  background: var(--color-bg-alt);
  border-radius: var(--radius);
  display: flex;
  align-items: center;
  justify-content: center;
}

.placeholder {
  font-size: 6rem;
  font-weight: 700;
  color: var(--color-text-muted);
  opacity: 0.4;
}

.product-info h1 {
  font-size: 1.75rem;
  margin-bottom: 16px;
}

.price {
  font-size: 1.75rem;
  font-weight: 700;
  color: var(--color-primary);
  margin-bottom: 12px;
}

.stock {
  color: var(--color-success);
  font-size: 0.95rem;
  margin-bottom: 20px;
}

.stock.out {
  color: var(--color-error);
}

.description {
  color: var(--color-text-muted);
  line-height: 1.6;
  margin-bottom: 24px;
  white-space: pre-wrap;
}

.actions .btn {
  padding: 12px 32px;
  font-size: 1.05rem;
}

.loading,
.not-found {
  padding: 48px 0;
  text-align: center;
  color: var(--color-text-muted);
}

.not-found h1 {
  margin-bottom: 16px;
}

@media (max-width: 768px) {
  .product-detail {
    grid-template-columns: 1fr;
    gap: 24px;
  }
}
</style>

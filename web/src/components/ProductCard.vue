<template>
  <router-link :to="`/products/${product.slug}`" class="product-card">
    <div class="product-image">
      <span class="product-image-placeholder">{{ initial }}</span>
    </div>

    <div class="product-body">
      <h3 class="product-name">{{ product.name }}</h3>
      <p class="product-price">{{ formattedPrice }}</p>
      <p v-if="product.stock_quantity === 0" class="product-stock out">
        {{ t('products.outOfStock') }}
      </p>
      <p v-else class="product-stock in">
        {{ t('products.inStock') }}
      </p>
    </div>
  </router-link>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Product } from '@/services/catalog'

const props = defineProps<{ product: Product }>()
const { t, locale } = useI18n()

const initial = computed(() => props.product.name.charAt(0).toUpperCase())

const formattedPrice = computed(() => {
  const amount = props.product.price_cents / 100
  try {
    return new Intl.NumberFormat(locale.value, {
      style: 'currency',
      currency: props.product.currency,
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(amount)
  } catch {
    return `${amount.toFixed(2)} ${props.product.currency}`
  }
})
</script>

<style scoped>
.product-card {
  display: block;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  overflow: hidden;
  background: var(--color-bg);
  color: var(--color-text);
  transition: box-shadow 0.15s, transform 0.15s;
  text-decoration: none;
}

.product-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  transform: translateY(-2px);
  text-decoration: none;
}

.product-image {
  aspect-ratio: 1 / 1;
  background: var(--color-bg-alt);
  display: flex;
  align-items: center;
  justify-content: center;
}

.product-image-placeholder {
  font-size: 3rem;
  font-weight: 700;
  color: var(--color-text-muted);
  opacity: 0.4;
}

.product-body {
  padding: 12px;
}

.product-name {
  font-size: 1rem;
  font-weight: 500;
  margin-bottom: 8px;
  line-height: 1.3;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.product-price {
  font-size: 1.1rem;
  font-weight: 700;
  color: var(--color-primary);
  margin-bottom: 4px;
}

.product-stock {
  font-size: 0.85rem;
}

.product-stock.in {
  color: var(--color-success);
}

.product-stock.out {
  color: var(--color-error);
}
</style>

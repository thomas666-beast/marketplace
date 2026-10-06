<template>
  <div class="container cart-page">
    <h1>{{ t('cart.title') }}</h1>

    <div v-if="cart.isEmpty" class="empty-state">
      <p class="empty-title">{{ t('cart.empty') }}</p>
      <p class="empty-hint">{{ t('cart.emptyHint') }}</p>
      <router-link to="/products" class="btn">
        {{ t('cart.browseProducts') }}
      </router-link>
    </div>

    <div v-else class="cart-layout">
      <div class="cart-items">
        <div v-for="item in cart.items" :key="item.product_id" class="cart-item">
          <router-link :to="`/products/${item.slug}`" class="item-image">
            <span>{{ item.name.charAt(0).toUpperCase() }}</span>
          </router-link>

          <div class="item-info">
            <router-link :to="`/products/${item.slug}`" class="item-name">
              {{ item.name }}
            </router-link>
            <div class="item-price">{{ formatPrice(item.price_cents, item.currency) }}</div>
            <div v-if="item.stock_quantity <= 5" class="item-stock">
              {{ t('cart.maxStock', { n: item.stock_quantity }) }}
            </div>
          </div>

          <div class="item-quantity">
            <button
              class="qty-btn"
              :disabled="item.quantity <= 1"
              @click="cart.updateQuantity(item.product_id, item.quantity - 1)"
            >
              −
            </button>
            <input
              :value="item.quantity"
              type="number"
              class="qty-input"
              min="1"
              :max="item.stock_quantity"
              @change="onQuantityChange(item.product_id, $event)"
            />
            <button
              class="qty-btn"
              :disabled="item.quantity >= item.stock_quantity"
              @click="cart.updateQuantity(item.product_id, item.quantity + 1)"
            >
              +
            </button>
          </div>

          <div class="item-total">
            {{ formatPrice(item.price_cents * item.quantity, item.currency) }}
          </div>

          <button class="item-remove" @click="cart.removeItem(item.product_id)">
            ✕
          </button>
        </div>

        <button class="clear-cart" @click="onClear">
          {{ t('cart.clearCart') }}
        </button>
      </div>

      <aside class="cart-summary">
        <h2>{{ t('cart.title') }}</h2>

        <div class="summary-row">
          <span>{{ t('cart.subtotal') }}</span>
          <span>{{ formatPrice(cart.subtotalCents, cart.currency) }}</span>
        </div>

        <div class="summary-row muted">
          <span>{{ t('cart.shipping') }}</span>
          <span>{{ t('cart.shippingNote') }}</span>
        </div>

        <div class="summary-row total">
          <span>{{ t('cart.total') }}</span>
          <span>{{ formatPrice(cart.subtotalCents, cart.currency) }}</span>
        </div>

        <router-link to="/checkout" class="btn checkout-btn">
          {{ t('cart.checkout') }}
        </router-link>

        <router-link to="/products" class="continue-link">
          {{ t('cart.continueShopping') }}
        </router-link>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useCartStore } from '@/stores/cart'

const { t, locale } = useI18n()
const cart = useCartStore()

function formatPrice(cents: number, currency: string): string {
  const amount = cents / 100
  try {
    return new Intl.NumberFormat(locale.value, {
      style: 'currency',
      currency,
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    }).format(amount)
  } catch {
    return `${amount.toFixed(2)} ${currency}`
  }
}

function onQuantityChange(productId: string, event: Event) {
  const target = event.target as HTMLInputElement
  const value = parseInt(target.value, 10)
  if (Number.isNaN(value)) return
  cart.updateQuantity(productId, value)
}

function onClear() {
  if (confirm(t('cart.clearCart') + '?')) {
    cart.clear()
  }
}
</script>

<style scoped>
.cart-page {
  padding: 16px 0 48px;
}

h1 {
  font-size: 1.75rem;
  margin-bottom: 24px;
}

.empty-state {
  text-align: center;
  padding: 64px 16px;
}

.empty-title {
  font-size: 1.25rem;
  margin-bottom: 8px;
}

.empty-hint {
  color: var(--color-text-muted);
  margin-bottom: 24px;
}

.cart-layout {
  display: grid;
  grid-template-columns: 1fr 320px;
  gap: 32px;
  align-items: start;
}

.cart-items {
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 8px 16px;
}

.cart-item {
  display: grid;
  grid-template-columns: 64px 1fr auto auto auto;
  gap: 16px;
  align-items: center;
  padding: 16px 0;
  border-bottom: 1px solid var(--color-border);
}

.cart-item:last-child {
  border-bottom: none;
}

.item-image {
  width: 64px;
  height: 64px;
  border-radius: var(--radius);
  background: var(--color-bg-alt);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--color-text-muted);
  opacity: 0.6;
}

.item-info {
  min-width: 0;
}

.item-name {
  display: block;
  font-weight: 500;
  margin-bottom: 4px;
  color: var(--color-text);
}

.item-price {
  color: var(--color-text-muted);
  font-size: 0.9rem;
}

.item-stock {
  color: var(--color-error);
  font-size: 0.85rem;
  margin-top: 2px;
}

.item-quantity {
  display: flex;
  align-items: center;
  gap: 4px;
}

.qty-btn {
  width: 32px;
  height: 32px;
  border: 1px solid var(--color-border);
  background: white;
  border-radius: var(--radius);
  font-size: 1.1rem;
}

.qty-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.qty-input {
  width: 48px;
  height: 32px;
  text-align: center;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-family: inherit;
  font-size: 0.95rem;
}

.item-total {
  font-weight: 700;
  white-space: nowrap;
}

.item-remove {
  border: none;
  background: transparent;
  color: var(--color-text-muted);
  font-size: 1rem;
  padding: 4px;
  border-radius: var(--radius);
}

.item-remove:hover {
  color: var(--color-error);
  background: var(--color-bg-alt);
}

.clear-cart {
  display: block;
  margin: 12px 0;
  border: none;
  background: transparent;
  color: var(--color-text-muted);
  font-size: 0.9rem;
  text-decoration: underline;
}

.cart-summary {
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 20px;
  position: sticky;
  top: 88px;
}

.cart-summary h2 {
  font-size: 1.1rem;
  margin-bottom: 16px;
}

.summary-row {
  display: flex;
  justify-content: space-between;
  padding: 6px 0;
  font-size: 0.95rem;
}

.summary-row.muted {
  color: var(--color-text-muted);
  font-size: 0.85rem;
}

.summary-row.total {
  padding-top: 12px;
  margin-top: 8px;
  border-top: 1px solid var(--color-border);
  font-size: 1.15rem;
  font-weight: 700;
}

.checkout-btn {
  display: block;
  width: 100%;
  text-align: center;
  margin-top: 16px;
  padding: 12px;
}

.continue-link {
  display: block;
  text-align: center;
  margin-top: 12px;
  font-size: 0.9rem;
  color: var(--color-text-muted);
}

@media (max-width: 900px) {
  .cart-layout {
    grid-template-columns: 1fr;
  }

  .cart-summary {
    position: static;
  }

  .cart-item {
    grid-template-columns: 48px 1fr auto;
    grid-template-areas:
      'img info info'
      'qty qty total'
      'qty qty remove';
    gap: 8px;
  }

  .item-image { grid-area: img; width: 48px; height: 48px; }
  .item-info { grid-area: info; }
  .item-quantity { grid-area: qty; }
  .item-total { grid-area: total; }
  .item-remove { grid-area: remove; justify-self: end; }
}
</style>

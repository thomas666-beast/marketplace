<template>
  <router-link to="/cart" class="cart-icon" :aria-label="t('cart.title')">
    <span class="cart-icon-symbol">🛒</span>
    <span v-if="cart.count > 0" class="cart-icon-badge">{{ cart.count }}</span>
  </router-link>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useCartStore } from '@/stores/cart'

const { t } = useI18n()
const cart = useCartStore()
</script>

<style scoped>
.cart-icon {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: var(--radius);
  transition: background 0.15s;
  color: var(--color-text);
  text-decoration: none;
}

.cart-icon:hover {
  background: var(--color-bg-alt);
  text-decoration: none;
}

.cart-icon-symbol {
  font-size: 1.25rem;
}

.cart-icon-badge {
  position: absolute;
  top: 2px;
  right: 2px;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 9px;
  background: var(--color-primary);
  color: white;
  font-size: 0.7rem;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>

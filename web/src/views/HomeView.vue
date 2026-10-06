<template>
  <div class="container home">
    <h1>{{ t('home.title') }}</h1>
    <p class="subtitle">{{ t('home.subtitle') }}</p>
    <router-link to="/products" class="btn">
      {{ t('home.browseProducts') }}
    </router-link>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
</script>

<style scoped>
.home {
  text-align: center;
  padding: 64px 16px;
}

h1 {
  font-size: 2rem;
  margin-bottom: 12px;
}

.subtitle {
  color: var(--color-text-muted);
  margin-bottom: 24px;
  font-size: 1.1rem;
}
</style>

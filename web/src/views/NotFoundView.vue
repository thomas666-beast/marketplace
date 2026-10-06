<template>
  <div class="container not-found">
    <h1>404</h1>
    <p>{{ t('notFound.title') }}</p>
    <router-link to="/" class="btn">{{ t('notFound.back') }}</router-link>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
</script>

<style scoped>
.not-found {
  text-align: center;
  padding: 64px 16px;
}

h1 {
  font-size: 4rem;
  color: var(--color-text-muted);
}
</style>

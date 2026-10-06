<template>
  <div class="pagination">
    <button
      v-if="hasMore"
      class="btn btn-outline"
      :disabled="loading"
      @click="$emit('loadMore')"
    >
      {{ loading ? t('common.loading') : t('products.loadMore') }}
    </button>
    <p v-else-if="showEnd" class="pagination-end">
      {{ t('products.noMore') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  hasMore: boolean
  loading: boolean
  showEnd: boolean
}>()

defineEmits<{
  (e: 'loadMore'): void
}>()

const { t } = useI18n()
</script>

<style scoped>
.pagination {
  display: flex;
  justify-content: center;
  margin: 32px 0;
}

.pagination-end {
  color: var(--color-text-muted);
  font-size: 0.9rem;
}
</style>

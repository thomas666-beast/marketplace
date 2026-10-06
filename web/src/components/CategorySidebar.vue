<template>
  <aside class="category-sidebar">
    <h2 class="sidebar-title">{{ t('products.categoriesTitle') }}</h2>

    <ul class="category-list">
      <li>
        <button
          class="category-item"
          :class="{ active: !selectedId }"
          @click="$emit('select', null)"
        >
          {{ t('products.allCategories') }}
        </button>
      </li>

      <li v-for="cat in categories" :key="cat.id">
        <button
          class="category-item"
          :class="{ active: selectedId === cat.id }"
          @click="$emit('select', cat.id)"
        >
          {{ cat.name }}
        </button>

        <ul v-if="cat.children?.length" class="category-children">
          <li v-for="child in cat.children" :key="child.id">
            <button
              class="category-item child"
              :class="{ active: selectedId === child.id }"
              @click="$emit('select', child.id)"
            >
              {{ child.name }}
            </button>
          </li>
        </ul>
      </li>
    </ul>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { Category } from '@/services/catalog'

defineProps<{
  categories: Category[]
  selectedId: string | null
}>()

defineEmits<{
  (e: 'select', id: string | null): void
}>()

const { t } = useI18n()
</script>

<style scoped>
.category-sidebar {
  padding: 16px 0;
}

.sidebar-title {
  font-size: 1rem;
  font-weight: 700;
  margin-bottom: 12px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-muted);
}

.category-list {
  list-style: none;
}

.category-children {
  list-style: none;
  padding-left: 12px;
  margin-top: 2px;
}

.category-item {
  display: block;
  width: 100%;
  text-align: left;
  padding: 6px 10px;
  border: none;
  background: transparent;
  border-radius: var(--radius);
  color: var(--color-text);
  font-size: 0.95rem;
  transition: background 0.1s;
}

.category-item:hover {
  background: var(--color-bg-alt);
}

.category-item.active {
  background: var(--color-primary);
  color: white;
  font-weight: 500;
}

.category-item.child {
  font-size: 0.9rem;
  color: var(--color-text-muted);
}

.category-item.child.active {
  color: white;
}
</style>

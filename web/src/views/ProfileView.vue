<template>
  <div class="container profile">
    <h1>{{ t('auth.profileTitle') }}</h1>

    <div v-if="!auth.user" class="loading">{{ t('common.loading') }}</div>

    <div v-else class="profile-card">
      <div class="row">
        <span class="label">{{ t('auth.displayName') }}</span>
        <span class="value">{{ auth.user.display_name }}</span>
      </div>
      <div class="row">
        <span class="label">{{ t('auth.email') }}</span>
        <span class="value">{{ auth.user.email }}</span>
      </div>
      <div class="row">
        <span class="label">{{ t('auth.role') }}</span>
        <span class="value">
          {{ auth.user.role === 'seller' ? t('auth.roleSeller') : t('auth.roleBuyer') }}
        </span>
      </div>
      <div class="row">
        <span class="label">{{ t('auth.preferredLocale') }}</span>
        <span class="value">{{ auth.user.preferred_locale.toUpperCase() }}</span>
      </div>
      <div class="row">
        <span class="label">{{ t('auth.createdAt') }}</span>
        <span class="value">{{ formatDate(auth.user.created_at) }}</span>
      </div>
      <div class="row">
        <span class="label">{{ t('auth.emailVerified') }}</span>
        <span class="value">{{ auth.user.email_verified ? t('auth.yes') : t('auth.no') }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const auth = useAuthStore()

function formatDate(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleDateString(undefined, { year: 'numeric', month: 'long', day: 'numeric' })
}

onMounted(async () => {
  if (!auth.user) {
    await auth.loadCurrentUser()
  }
})
</script>

<style scoped>
.profile {
  max-width: 600px;
  padding: 32px 16px;
}

h1 {
  font-size: 1.5rem;
  margin-bottom: 24px;
}

.profile-card {
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 24px;
  background: var(--color-bg);
}

.row {
  display: flex;
  padding: 12px 0;
  border-bottom: 1px solid var(--color-border);
}

.row:last-child {
  border-bottom: none;
}

.label {
  flex: 0 0 160px;
  color: var(--color-text-muted);
  font-weight: 500;
}

.value {
  flex: 1;
}

.loading {
  color: var(--color-text-muted);
}
</style>

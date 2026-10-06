<template>
  <div class="container auth-page">
    <div class="auth-card">
      <h1>{{ t('auth.loginTitle') }}</h1>

      <form @submit.prevent="onSubmit">
        <div class="form-group">
          <label for="email">{{ t('auth.email') }}</label>
          <input
            id="email"
            v-model="form.email"
            type="email"
            class="input"
            autocomplete="email"
            @blur="touched.email = true"
          />
          <div v-if="touched.email && errors.email" class="error-text">
            {{ errors.email }}
          </div>
        </div>

        <div class="form-group">
          <label for="password">{{ t('auth.password') }}</label>
          <input
            id="password"
            v-model="form.password"
            type="password"
            class="input"
            autocomplete="current-password"
            @blur="touched.password = true"
          />
          <div v-if="touched.password && errors.password" class="error-text">
            {{ errors.password }}
          </div>
        </div>

        <div v-if="serverError" class="error-text server-error">
          {{ serverError }}
        </div>

        <button type="submit" class="btn full-width" :disabled="auth.loading || !isValid">
          {{ auth.loading ? t('common.loading') : t('auth.signIn') }}
        </button>
      </form>

      <p class="auth-switch">
        {{ t('auth.noAccount') }}
        <router-link to="/register">{{ t('auth.signUp') }}</router-link>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import type { ApiError } from '@/services/api'

const { t } = useI18n()
const auth = useAuthStore()
const router = useRouter()
const route = useRoute()

const form = reactive({ email: '', password: '' })
const touched = reactive({ email: false, password: false })
const serverError = ref<string | null>(null)

const errors = computed(() => {
  const e: { email?: string; password?: string } = {}
  if (!form.email) e.email = t('auth.validation.emailRequired')
  else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) e.email = t('auth.validation.emailInvalid')
  if (!form.password) e.password = t('auth.validation.passwordRequired')
  return e
})

const isValid = computed(() => Object.keys(errors.value).length === 0)

async function onSubmit() {
  touched.email = true
  touched.password = true
  if (!isValid.value) return

  serverError.value = null
  try {
    await auth.login({ email: form.email, password: form.password })
    const redirect = (route.query.redirect as string) || '/'
    router.push(redirect)
  } catch (err) {
    const apiErr = err as ApiError
    serverError.value = apiErr.message || t('common.error')
  }
}
</script>

<style scoped>
.auth-page {
  display: flex;
  justify-content: center;
  padding: 48px 16px;
}

.auth-card {
  width: 100%;
  max-width: 400px;
  padding: 32px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-bg);
}

h1 {
  font-size: 1.5rem;
  margin-bottom: 24px;
}

.full-width {
  width: 100%;
  margin-top: 8px;
}

.server-error {
  margin-bottom: 12px;
  padding: 8px 12px;
  background: #ffebee;
  border-radius: var(--radius);
}

.auth-switch {
  margin-top: 16px;
  text-align: center;
  color: var(--color-text-muted);
  font-size: 0.9rem;
}
</style>

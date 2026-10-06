<template>
  <div class="container auth-page">
    <div class="auth-card">
      <h1>{{ t('auth.registerTitle') }}</h1>

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
          <div v-if="touched.email && errors.email" class="error-text">{{ errors.email }}</div>
        </div>

        <div class="form-group">
          <label for="displayName">{{ t('auth.displayName') }}</label>
          <input
            id="displayName"
            v-model="form.displayName"
            type="text"
            class="input"
            autocomplete="name"
            @blur="touched.displayName = true"
          />
          <div v-if="touched.displayName && errors.displayName" class="error-text">
            {{ errors.displayName }}
          </div>
        </div>

        <div class="form-group">
          <label for="password">{{ t('auth.password') }}</label>
          <input
            id="password"
            v-model="form.password"
            type="password"
            class="input"
            autocomplete="new-password"
            @blur="touched.password = true"
          />
          <div v-if="touched.password && errors.password" class="error-text">{{ errors.password }}</div>
        </div>

        <div class="form-group">
          <label>{{ t('auth.role') }}</label>
          <div class="radio-row">
            <label class="radio">
              <input v-model="form.role" type="radio" value="buyer" />
              <span>{{ t('auth.roleBuyer') }}</span>
            </label>
            <label class="radio">
              <input v-model="form.role" type="radio" value="seller" />
              <span>{{ t('auth.roleSeller') }}</span>
            </label>
          </div>
        </div>

        <div class="form-group">
          <label>{{ t('auth.preferredLocale') }}</label>
          <select v-model="form.preferredLocale" class="input">
            <option value="en">English</option>
            <option value="ru">Русский</option>
            <option value="es">Español</option>
          </select>
        </div>

        <div v-if="serverError" class="error-text server-error">{{ serverError }}</div>

        <button type="submit" class="btn full-width" :disabled="auth.loading || !isValid">
          {{ auth.loading ? t('common.loading') : t('auth.signUp') }}
        </button>
      </form>

      <p class="auth-switch">
        {{ t('auth.alreadyHaveAccount') }}
        <router-link to="/login">{{ t('auth.signIn') }}</router-link>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import type { ApiError } from '@/services/api'

const { t, locale } = useI18n()
const auth = useAuthStore()
const router = useRouter()

const form = reactive({
  email: '',
  displayName: '',
  password: '',
  role: 'buyer' as 'buyer' | 'seller',
  preferredLocale: locale.value as 'en' | 'ru' | 'es',
})

const touched = reactive({ email: false, displayName: false, password: false })
const serverError = ref<string | null>(null)

const errors = computed(() => {
  const e: { email?: string; displayName?: string; password?: string } = {}
  if (!form.email) e.email = t('auth.validation.emailRequired')
  else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) e.email = t('auth.validation.emailInvalid')

  if (!form.displayName) e.displayName = t('auth.validation.displayNameRequired')
  else if (form.displayName.trim().length < 2) e.displayName = t('auth.validation.displayNameTooShort')

  if (!form.password) e.password = t('auth.validation.passwordRequired')
  else if (form.password.length < 8) e.password = t('auth.validation.passwordTooShort')

  return e
})

const isValid = computed(() => Object.keys(errors.value).length === 0)

async function onSubmit() {
  touched.email = true
  touched.displayName = true
  touched.password = true
  if (!isValid.value) return

  serverError.value = null
  try {
    await auth.register({
      email: form.email,
      password: form.password,
      display_name: form.displayName,
      role: form.role,
      preferred_locale: form.preferredLocale,
    })
    router.push('/')
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
  max-width: 440px;
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

.radio-row {
  display: flex;
  gap: 24px;
  padding-top: 4px;
}

.radio {
  display: flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  font-weight: normal;
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

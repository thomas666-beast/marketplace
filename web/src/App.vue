<template>
  <div class="app">
    <header class="app-header">
      <div class="container app-header-inner">
        <router-link to="/" class="app-logo">
          {{ t('app.name') }}
        </router-link>

        <nav class="app-nav">
          <router-link to="/">{{ t('nav.home') }}</router-link>
          <router-link to="/products">{{ t('nav.products') }}</router-link>
          <router-link v-if="auth.isAuthenticated" to="/profile">
            {{ t('nav.profile') }}
          </router-link>
        </nav>

        <div class="app-actions">
          <select v-model="currentLocale" @change="onLocaleChange" class="locale-select">
            <option value="en">English</option>
            <option value="ru">Русский</option>
            <option value="es">Español</option>
          </select>

          <CartIcon />

          <template v-if="auth.isAuthenticated">
            <span class="user-name">{{ auth.user?.display_name }}</span>
            <button class="btn-outline btn" @click="onLogout">
              {{ t('nav.logout') }}
            </button>
          </template>
          <template v-else>
            <router-link to="/login" class="btn-outline btn">
              {{ t('nav.login') }}
            </router-link>
            <router-link to="/register" class="btn">
              {{ t('nav.register') }}
            </router-link>
          </template>
        </div>
      </div>
    </header>

    <main class="app-main">
      <router-view />
    </main>

    <footer class="app-footer">
      <div class="container">
        {{ t('app.name') }} &copy; 2026
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
    import { onMounted, ref } from 'vue'
    import { useI18n } from 'vue-i18n'
    import { useRouter } from 'vue-router'
    import { useAuthStore } from '@/stores/auth'
    import CartIcon from '@/components/CartIcon.vue'

    const { t, locale } = useI18n()
    const auth = useAuthStore()
    const router = useRouter()

    const currentLocale = ref(locale.value)

    function onLocaleChange() {
    locale.value = currentLocale.value
    localStorage.setItem('marketplace.locale', currentLocale.value)
    }

    function onLogout() {
    auth.logout()
    router.push('/')
    }

    onMounted(async () => {
    if (auth.accessToken && !auth.user) {
        await auth.loadCurrentUser()
    }
    })
</script>

<style scoped>
.app {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.app-header {
  background: var(--color-bg);
  border-bottom: 1px solid var(--color-border);
  position: sticky;
  top: 0;
  z-index: 10;
}

.app-header-inner {
  display: flex;
  align-items: center;
  gap: 24px;
  height: 64px;
}

.app-logo {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--color-text);
}

.app-logo:hover {
  text-decoration: none;
}

.app-nav {
  display: flex;
  gap: 16px;
  flex: 1;
}

.app-nav a {
  color: var(--color-text);
  font-weight: 500;
}

.app-nav a.router-link-active {
  color: var(--color-primary);
}

.app-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.locale-select {
  padding: 6px 10px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-bg);
  font-family: inherit;
}

.user-name {
  color: var(--color-text-muted);
  font-size: 0.9rem;
}

.app-main {
  flex: 1;
  padding: 24px 0;
}

.app-footer {
  background: var(--color-bg-alt);
  padding: 16px 0;
  color: var(--color-text-muted);
  font-size: 0.9rem;
  text-align: center;
}
</style>

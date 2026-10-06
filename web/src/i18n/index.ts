import { createI18n } from 'vue-i18n'
import en from './stores/en.json'
import ru from './stores/ru.json'
import es from './stores/es.json'

export type SupportedLocale = 'en' | 'ru' | 'es'

const SUPPORTED: SupportedLocale[] = ['en', 'ru', 'es']

function detectLocale(): SupportedLocale {
  const saved = localStorage.getItem('marketplace.locale')
  if (saved && SUPPORTED.includes(saved as SupportedLocale)) {
    return saved as SupportedLocale
  }
  const browser = navigator.language.split('-')[0]
  if (SUPPORTED.includes(browser as SupportedLocale)) {
    return browser as SupportedLocale
  }
  return 'en'
}

const i18n = createI18n({
  legacy: false,
  globalInjection: true,
  locale: detectLocale(),
  fallbackLocale: 'en',
  messages: { en, ru, es },
})

export default i18n

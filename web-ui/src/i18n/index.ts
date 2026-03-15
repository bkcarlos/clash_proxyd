import { createI18n } from 'vue-i18n'
import zh from './locales/zh'
import en from './locales/en'

const savedLocale = localStorage.getItem('locale') || 'zh'

export const i18n = createI18n({
  legacy: false,
  locale: savedLocale,
  fallbackLocale: 'en',
  messages: { zh, en },
})

export const setLocale = (locale: 'zh' | 'en') => {
  i18n.global.locale.value = locale
  localStorage.setItem('locale', locale)
}

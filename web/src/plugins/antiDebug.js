import DisableDevtool from 'disable-devtool'

const DEFAULT_REDIRECT_URL = 'about:blank'

const isEnabledByEnv = () => {
  if (!import.meta.env.PROD) return false
  return String(import.meta.env.VITE_ANTI_DEBUG || 'true') !== 'false'
}

const getBypassMd5 = (options = {}) => {
  return options.md5 || options.bypassMd5 || import.meta.env.VITE_ANTI_DEBUG_BYPASS_MD5 || ''
}

const getRedirectUrl = (options = {}) => {
  return options.redirectUrl || import.meta.env.VITE_ANTI_DEBUG_REDIRECT_URL || DEFAULT_REDIRECT_URL
}

const startAntiDebug = (options = {}) => {
  if (typeof window === 'undefined' || typeof document === 'undefined') return

  const enabled = typeof options.enabled === 'boolean' ? options.enabled : isEnabledByEnv()
  if (!enabled || window.__ANTI_DEBUG_STARTED__) return
  window.__ANTI_DEBUG_STARTED__ = true

  const bypassMd5 = getBypassMd5(options)
  const redirectUrl = getRedirectUrl(options)

  DisableDevtool({
    ...(bypassMd5 ? { md5: bypassMd5 } : {}),
    tkName: options.tkName || import.meta.env.VITE_ANTI_DEBUG_TK_NAME || 'ddtk',
    url: redirectUrl,
    timeOutUrl: redirectUrl,
    interval: options.interval || 500,
    disableMenu: options.disableMenu ?? true,
    clearLog: options.clearLog ?? true,
    disableSelect: options.disableSelect ?? false,
    disableCopy: options.disableCopy ?? false,
    disableCut: options.disableCut ?? false,
    disablePaste: options.disablePaste ?? false,
    clearIntervalWhenDevOpenTrigger: options.clearIntervalWhenDevOpenTrigger ?? true,
    ignore: options.ignore || null
  })
}

export default {
  install(_app, options = {}) {
    startAntiDebug(options)
  }
}

import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { OverlayScrollbarsComponent } from 'overlayscrollbars-react'
import { Sidebar } from './components/Sidebar/Sidebar'
import { WikiEditor } from './components/Editor/WikiEditor'
import { ToastViewport } from './components/Toast/ToastViewport'
import { useWikiStore } from './stores/wikiStore'
import { api, clearAuthToken, getAuthToken } from './utils/api'
import { useToastStore } from './stores/toastStore'
import './App.css'

const scrollOptions = {
  scrollbars: { autoHide: 'scroll' as const, autoHideDelay: 800 },
}

function getPageIdFromUrl(): string | null {
  const params = new URLSearchParams(window.location.search)
  return params.get('page')
}

function setPageIdInUrl(pageId: string | null) {
  const url = new URL(window.location.href)
  if (pageId) {
    url.searchParams.set('page', pageId)
  } else {
    url.searchParams.delete('page')
  }
  window.history.pushState({}, '', url.toString())
}

type AuthMode = 'login' | 'register'
type AuthStatus = 'checking' | 'authenticated' | 'unauthenticated'

function AuthScreen({
  mode,
  loading,
  error,
  onModeChange,
  onSubmit,
}: {
  mode: AuthMode
  loading: boolean
  error: string | null
  onModeChange: (mode: AuthMode) => void
  onSubmit: (username: string, password: string) => Promise<void>
}) {
  const { t } = useTranslation()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    await onSubmit(username.trim(), password)
  }

  return (
    <div className="auth-screen">
      <div className="auth-card">
        <div className="auth-brand">
          <span className="auth-brand-icon">W</span>
          <div>
            <div className="auth-title">{t(mode === 'login' ? 'auth.loginTitle' : 'auth.registerTitle')}</div>
            <div className="auth-subtitle">{t('auth.subtitle')}</div>
          </div>
        </div>

        <div className="auth-tabs">
          <button
            type="button"
            className={`auth-tab ${mode === 'login' ? 'is-active' : ''}`}
            onClick={() => onModeChange('login')}
          >
            {t('auth.login')}
          </button>
          <button
            type="button"
            className={`auth-tab ${mode === 'register' ? 'is-active' : ''}`}
            onClick={() => onModeChange('register')}
          >
            {t('auth.register')}
          </button>
        </div>

        <form className="auth-form" onSubmit={handleSubmit}>
          <input
            className="auth-input"
            value={username}
            onChange={(event) => setUsername(event.target.value)}
            placeholder={t('auth.username')}
            autoComplete="username"
          />
          <input
            className="auth-input"
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            placeholder={t('auth.password')}
            autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
          />
          {error && <div className="auth-error">{error}</div>}
          <button
            className="btn-primary auth-submit"
            type="submit"
            disabled={loading || username.trim().length < 2 || password.length < 4}
          >
            {loading ? t('auth.loading') : t(mode === 'login' ? 'auth.login' : 'auth.register')}
          </button>
        </form>
      </div>
    </div>
  )
}

function App() {
  const { t } = useTranslation()
  const loadPages = useWikiStore((s) => s.loadPages)
  const refreshPages = useWikiStore((s) => s.refreshPages)
  const loaded = useWikiStore((s) => s.loaded)
  const currentPageId = useWikiStore((s) => s.currentPageId)
  const setCurrentPage = useWikiStore((s) => s.setCurrentPage)
  const createPage = useWikiStore((s) => s.createPage)
  const pushToast = useToastStore((s) => s.pushToast)
  const [authStatus, setAuthStatus] = useState<AuthStatus>('checking')
  const [authMode, setAuthMode] = useState<AuthMode>('login')
  const [authLoading, setAuthLoading] = useState(false)
  const [authError, setAuthError] = useState<string | null>(null)

  useEffect(() => {
    let active = true

    const checkAuth = async () => {
      if (!getAuthToken()) {
        if (active) setAuthStatus('unauthenticated')
        return
      }

      const result = await api.fetchMe()
      if (!active) return
      setAuthStatus(result.data ? 'authenticated' : 'unauthenticated')
      setAuthError(result.error?.status === 401 ? null : result.error?.message ?? null)
    }

    void checkAuth()
    return () => {
      active = false
    }
  }, [])

  useEffect(() => {
    if (authStatus !== 'authenticated') return
    loadPages()
  }, [authStatus, loadPages])

  useEffect(() => {
    if (loaded && authStatus === 'authenticated') setPageIdInUrl(currentPageId)
  }, [currentPageId, loaded, authStatus])

  useEffect(() => {
    if (authStatus !== 'authenticated') return

    if (!loaded) return

    const syncPages = () => {
      void refreshPages()
    }

    const intervalId = window.setInterval(syncPages, 3000)
    const handleFocus = () => syncPages()
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        syncPages()
      }
    }

    window.addEventListener('focus', handleFocus)
    document.addEventListener('visibilitychange', handleVisibilityChange)

    return () => {
      window.clearInterval(intervalId)
      window.removeEventListener('focus', handleFocus)
      document.removeEventListener('visibilitychange', handleVisibilityChange)
    }
  }, [loaded, refreshPages, authStatus])

  useEffect(() => {
    if (authStatus !== 'authenticated') return

    const handler = () => {
      const urlPageId = getPageIdFromUrl()
      setCurrentPage(urlPageId)
    }
    window.addEventListener('popstate', handler)
    return () => window.removeEventListener('popstate', handler)
  }, [setCurrentPage, authStatus])

  const handleAuthSubmit = async (username: string, password: string) => {
    if (authLoading) return
    setAuthLoading(true)
    setAuthError(null)

    const result = authMode === 'login'
      ? await api.login(username, password)
      : await api.register(username, password)

    if (result.data) {
      setAuthStatus('authenticated')
      setAuthLoading(false)
      return
    }

    clearAuthToken()
    const message = result.error?.message ?? t('auth.error')
    setAuthError(message)
    pushToast(message, 'error')
    setAuthLoading(false)
  }

  if (authStatus === 'checking') {
    return (
      <>
        <div className="app-loading">
          <div className="spinner" />
          {t('auth.checking')}
        </div>
        <ToastViewport />
      </>
    )
  }

  if (authStatus === 'unauthenticated') {
    return (
      <>
        <AuthScreen
          mode={authMode}
          loading={authLoading}
          error={authError}
          onModeChange={setAuthMode}
          onSubmit={handleAuthSubmit}
        />
        <ToastViewport />
      </>
    )
  }

  if (!loaded) {
    return (
      <>
        <div className="app-loading">
          <div className="spinner" />
          {t('app.loading')}
        </div>
        <ToastViewport />
      </>
    )
  }

  return (
    <>
      <div className="app">
        <Sidebar />
        <OverlayScrollbarsComponent element="main" className="main-content" options={scrollOptions} defer>
          {currentPageId ? (
            <WikiEditor key={currentPageId} pageId={currentPageId} />
          ) : (
            <div className="main-empty">
              <div className="main-empty-content">
                <div className="main-empty-icon">
                  <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
                    <polyline points="14 2 14 8 20 8"/>
                    <line x1="16" y1="13" x2="8" y2="13"/>
                    <line x1="16" y1="17" x2="8" y2="17"/>
                    <polyline points="10 9 9 9 8 9"/>
                  </svg>
                </div>
                <h2>{t('app.title')}</h2>
                <p>{t('app.selectPage')}</p>
                <button className="btn-primary" onClick={() => createPage()}>
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
                  {t('app.newPage')}
                </button>
              </div>
            </div>
          )}
        </OverlayScrollbarsComponent>
      </div>
      <ToastViewport />
    </>
  )
}

export default App

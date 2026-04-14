import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { OverlayScrollbarsComponent } from 'overlayscrollbars-react'
import { useWikiStore } from '../../stores/wikiStore'
import type { WikiPage, WikiSearchResult } from '../../types'
import { api } from '../../utils/api'
import './Sidebar.css'

const LANGS = [
  { code: 'ru', label: 'RU' },
  { code: 'en', label: 'EN' },
]

const scrollOptions = {
  scrollbars: { autoHide: 'scroll' as const, autoHideDelay: 800 },
}

type PageTreeMap = Map<string | null, WikiPage[]>
type PageByIdMap = Map<string, WikiPage>

function SidebarPageTree({
  pageId,
  currentPageId,
  level,
  pagesByParent,
  pagesById,
  onOpen,
  onDelete,
  untitledLabel,
  deleteLabel,
}: {
  pageId: string
  currentPageId: string | null
  level: number
  pagesByParent: PageTreeMap
  pagesById: PageByIdMap
  onOpen: (id: string) => void
  onDelete: (id: string) => void
  untitledLabel: string
  deleteLabel: string
}) {
  const page = pagesById.get(pageId)
  if (!page) return null
  const children = pagesByParent.get(pageId) ?? []

  return (
    <>
      <div
        className={`sidebar-page ${page.id === currentPageId ? 'is-active' : ''}`}
        onClick={() => onOpen(page.id)}
        style={{ paddingLeft: `${8 + level * 16}px` }}
      >
        <span className="sidebar-page-icon">{page.icon || '📄'}</span>
        <span className="sidebar-page-title">{page.title || untitledLabel}</span>
        <button
          className="sidebar-page-delete"
          onClick={(event) => {
            event.stopPropagation()
            onDelete(page.id)
          }}
          title={deleteLabel}
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
        </button>
      </div>
      {children.map((child) => (
        <SidebarPageTree
          key={child.id}
          pageId={child.id}
          currentPageId={currentPageId}
          level={level + 1}
          pagesByParent={pagesByParent}
          pagesById={pagesById}
          onOpen={onOpen}
          onDelete={onDelete}
          untitledLabel={untitledLabel}
          deleteLabel={deleteLabel}
        />
      ))}
    </>
  )
}

export function Sidebar() {
  const { t, i18n } = useTranslation()
  const pages = useWikiStore((s) => s.pages)
  const currentPageId = useWikiStore((s) => s.currentPageId)
  const createPage = useWikiStore((s) => s.createPage)
  const setCurrentPage = useWikiStore((s) => s.setCurrentPage)
  const deletePage = useWikiStore((s) => s.deletePage)
  const [search, setSearch] = useState('')
  const [searchResults, setSearchResults] = useState<WikiSearchResult[] | null>(null)
  const [searchLoading, setSearchLoading] = useState(false)

  useEffect(() => {
    const query = search.trim()
    if (query.length < 2) {
      setSearchResults(null)
      setSearchLoading(false)
      return
    }

    let active = true
    setSearchLoading(true)
    const timer = setTimeout(() => {
      api.searchPages(query, 8).then((results) => {
        if (!active) return
        setSearchResults(results)
        setSearchLoading(false)
      })
    }, 220)

    return () => {
      active = false
      clearTimeout(timer)
    }
  }, [search])

  const switchLang = (code: string) => {
    i18n.changeLanguage(code)
    localStorage.setItem('mws-wiki-lang', code)
  }

  const filtered = search
    ? pages.filter((p) => p.title.toLowerCase().includes(search.toLowerCase()))
    : pages
  const remoteMode = search.trim().length >= 2 && searchResults !== null
  const pagesByParent = pages.reduce<PageTreeMap>((map, page) => {
    const key = page.parentId ?? null
    const current = map.get(key) ?? []
    current.push(page)
    map.set(key, current)
    return map
  }, new Map())
  const pagesById = new Map(pages.map((page) => [page.id, page]))
  const rootPageIds = (pagesByParent.get(null) ?? [])
    .concat(pages.filter((page) => page.parentId && !pagesById.has(page.parentId)))
    .filter((page, index, list) => list.findIndex((item) => item.id === page.id) === index)
    .map((page) => page.id)

  return (
    <aside className="sidebar">
      <div className="sidebar-header">
        <div className="sidebar-logo">
          <span className="sidebar-logo-icon">W</span>
          <span>{t('sidebar.logo')}</span>
        </div>
        <div className="lang-switcher">
          {LANGS.map((l) => (
            <button
              key={l.code}
              className={`lang-btn ${i18n.language === l.code ? 'is-active' : ''}`}
              onClick={() => switchLang(l.code)}
            >
              {l.label}
            </button>
          ))}
        </div>
      </div>

      <div className="sidebar-search">
        <svg className="sidebar-search-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
          <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
        </svg>
        <input
          className="sidebar-search-input"
          placeholder={t('sidebar.search', 'Search...')}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      <OverlayScrollbarsComponent className="sidebar-section" options={scrollOptions} defer>
        <div className="sidebar-section-header">
          <span>{t('sidebar.pages')}</span>
          <button className="sidebar-add" onClick={() => createPage()} title={t('sidebar.newPage')}>
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/></svg>
          </button>
        </div>
        <div className="sidebar-pages">
          {searchLoading && <div className="sidebar-empty">{t('sidebar.searching')}</div>}
          {!searchLoading && remoteMode && searchResults?.map((result) => (
            <button
              key={result.pageId}
              type="button"
              className={`sidebar-search-result ${result.pageId === currentPageId ? 'is-active' : ''}`}
              onClick={() => setCurrentPage(result.pageId)}
            >
              <span className="sidebar-search-result-title">{result.title || t('editor.untitled')}</span>
              <span className="sidebar-search-result-snippet">{result.snippet}</span>
            </button>
          ))}
          {!searchLoading && remoteMode && searchResults?.length === 0 && (
            <div className="sidebar-empty">{t('sidebar.noResults')}</div>
          )}
          {!remoteMode && !search && rootPageIds.map((rootId) => (
            <SidebarPageTree
              key={rootId}
              pageId={rootId}
              currentPageId={currentPageId}
              level={0}
              pagesByParent={pagesByParent}
              pagesById={pagesById}
              onOpen={(id) => setCurrentPage(id)}
              onDelete={(id) => deletePage(id)}
              untitledLabel={t('editor.untitled')}
              deleteLabel={t('sidebar.deletePage')}
            />
          ))}
          {!remoteMode && Boolean(search) && filtered.map((page) => (
            <div
              key={page.id}
              className={`sidebar-page ${page.id === currentPageId ? 'is-active' : ''}`}
              onClick={() => setCurrentPage(page.id)}
            >
              <span className="sidebar-page-icon">{page.icon || '📄'}</span>
              <span className="sidebar-page-title">{page.title || t('editor.untitled')}</span>
              <button
                className="sidebar-page-delete"
                onClick={(e) => {
                  e.stopPropagation()
                  deletePage(page.id)
                }}
                title={t('sidebar.deletePage')}
              >
                <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>
              </button>
            </div>
          ))}
          {!remoteMode && !filtered.length && (
            <div className="sidebar-empty">
              {search ? t('sidebar.noResults', 'Nothing found') : t('sidebar.empty')}
            </div>
          )}
        </div>
      </OverlayScrollbarsComponent>
    </aside>
  )
}

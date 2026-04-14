import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { api, type ApiPageSummary } from '../../utils/api'
import { useWikiStore } from '../../stores/wikiStore'
import './ChildPages.css'

interface ChildPagesProps {
  pageId: string
  compact?: boolean
}

export function ChildPages({ pageId, compact = false }: ChildPagesProps) {
  const { t } = useTranslation()
  const pages = useWikiStore((state) => state.pages)
  const createPage = useWikiStore((state) => state.createPage)
  const setCurrentPage = useWikiStore((state) => state.setCurrentPage)
  const [remoteChildren, setRemoteChildren] = useState<ApiPageSummary[] | null>(null)
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)

  const localChildren = useMemo(
    () => pages.filter((page) => page.parentId === pageId),
    [pageId, pages]
  )

  useEffect(() => {
    let active = true
    setLoading(true)
    api.fetchChildren(pageId).then((data) => {
      if (!active) return
      setRemoteChildren(data)
      setLoading(false)
    })
    return () => {
      active = false
    }
  }, [pageId, pages.length])

  const localItems = localChildren.map((page) => ({
    id: page.id,
    title: page.title,
    icon: page.icon,
    parent_id: page.parentId ?? null,
    version: page.version ?? 0,
    updatedAt: new Date(page.updatedAt).toISOString(),
  }))

  const items = [...(remoteChildren ?? []), ...localItems].reduce<ApiPageSummary[]>((acc, item) => {
    if (acc.some((existing) => existing.id === item.id)) return acc
    acc.push(item)
    return acc
  }, [])

  const handleCreate = async () => {
    if (creating) return
    setCreating(true)
    await createPage(t('children.newPageTitle'), pageId)
    setCreating(false)
  }

  return (
    <section className={`child-pages ${compact ? 'is-compact' : ''}`}>
      <div className="child-pages-header">
        <div>
          <div className="child-pages-title">{t('children.title')}</div>
          <div className="child-pages-subtitle">{t('children.subtitle')}</div>
        </div>
        <button className="child-pages-add" type="button" onClick={handleCreate} disabled={creating}>
          {creating ? t('children.creating') : t('children.add')}
        </button>
      </div>

      <div className="child-pages-list">
        {loading && <div className="child-pages-empty">{t('children.loading')}</div>}
        {!loading && items.length === 0 && <div className="child-pages-empty">{t('children.empty')}</div>}
        {items.map((child) => (
          <button
            key={child.id}
            type="button"
            className="child-page-card"
            onClick={() => setCurrentPage(child.id)}
          >
            <span className="child-page-icon">{child.icon || '📄'}</span>
            <span className="child-page-copy">
              <span className="child-page-name">{child.title || t('editor.untitled')}</span>
              <span className="child-page-meta">
                {new Date(child.updatedAt).toLocaleString()}
              </span>
            </span>
          </button>
        ))}
      </div>
    </section>
  )
}

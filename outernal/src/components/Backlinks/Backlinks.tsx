import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useWikiStore } from '../../stores/wikiStore'
import type { Backlink } from '../../types'
import './Backlinks.css'

interface BacklinksProps {
  pageId: string
}

export function Backlinks({ pageId }: BacklinksProps) {
  const { t } = useTranslation()
  const getBacklinks = useWikiStore((s) => s.getBacklinks)
  const fetchBacklinks = useWikiStore((s) => s.fetchBacklinks)
  const setCurrentPage = useWikiStore((s) => s.setCurrentPage)
  const [backlinks, setBacklinks] = useState<Backlink[]>(() => getBacklinks(pageId))

  useEffect(() => {
    setBacklinks(getBacklinks(pageId))
    fetchBacklinks(pageId).then(setBacklinks)
  }, [pageId, getBacklinks, fetchBacklinks])

  if (!backlinks.length) return null

  return (
    <div className="backlinks">
      <div className="backlinks-header">{t('backlinks.title')}</div>
      <div className="backlinks-list">
        {backlinks.map((bl) => (
          <button
            key={bl.sourcePageId}
            className="backlink-item"
            onClick={() => setCurrentPage(bl.sourcePageId)}
          >
            <span className="backlink-icon">↩</span>
            {bl.sourcePageTitle}
          </button>
        ))}
      </div>
    </div>
  )
}

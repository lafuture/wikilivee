import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useWikiStore } from '../../stores/wikiStore'
import './TablePicker.css'

interface PageLinkPickerProps {
  currentPageId: string
  onSelect: (pageId: string, pageTitle: string) => void
  onClose: () => void
}

export function PageLinkPicker({ currentPageId, onSelect, onClose }: PageLinkPickerProps) {
  const { t } = useTranslation()
  const pages = useWikiStore((s) => s.pages)
  const [search, setSearch] = useState('')

  const filtered = pages
    .filter((p) => p.id !== currentPageId)
    .filter((p) => p.title.toLowerCase().includes(search.toLowerCase()))

  return (
    <div className="table-picker-overlay" onClick={onClose}>
      <div className="table-picker" onClick={(e) => e.stopPropagation()}>
        <div className="table-picker-header">
          <h3>{t('pageLinkPicker.title')}</h3>
          <button className="table-picker-close" onClick={onClose}>
            &times;
          </button>
        </div>
        <input
          className="table-picker-search"
          type="text"
          placeholder={t('pageLinkPicker.search')}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          autoFocus
        />
        <div className="table-picker-list">
          {filtered.map((page) => (
            <button
              key={page.id}
              className="table-picker-item"
              onClick={() => onSelect(page.id, page.title)}
            >
              <span className="table-picker-icon">📄</span>
              <span>{page.title}</span>
            </button>
          ))}
          {!filtered.length && (
            <div className="table-picker-empty">{t('pageLinkPicker.empty')}</div>
          )}
        </div>
      </div>
    </div>
  )
}

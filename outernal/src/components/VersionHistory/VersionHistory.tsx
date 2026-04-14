import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { JSONContent } from '@tiptap/react'
import type { WikiPage, WikiPageVersion, WikiPageVersionSummary } from '../../types'
import { api } from '../../utils/api'
import './VersionHistory.css'

interface VersionHistoryProps {
  pageId: string
  currentVersion?: number
  onRestore: (page: WikiPage) => Promise<void> | void
  embedded?: boolean
}

function extractPreviewText(node: JSONContent | undefined): string {
  if (!node) return ''
  if (typeof node.text === 'string') return node.text
  if (!node.content) return ''
  return node.content.map((child) => extractPreviewText(child)).join(' ').trim()
}

function versionSnippet(version: WikiPageVersion | null): string {
  if (!version) return ''
  return extractPreviewText(version.content).replace(/\s+/g, ' ').trim()
}

export function VersionHistory({ pageId, currentVersion, onRestore, embedded = false }: VersionHistoryProps) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(embedded)
  const [versions, setVersions] = useState<WikiPageVersionSummary[]>([])
  const [selectedVersion, setSelectedVersion] = useState<number | null>(null)
  const [selectedData, setSelectedData] = useState<WikiPageVersion | null>(null)
  const [loadingList, setLoadingList] = useState(false)
  const [loadingVersion, setLoadingVersion] = useState(false)
  const [restoring, setRestoring] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    setExpanded(embedded)
  }, [embedded, pageId])

  useEffect(() => {
    setVersions([])
    setSelectedVersion(null)
    setSelectedData(null)
    setError(null)
  }, [pageId])

  useEffect(() => {
    if (!expanded) return

    let active = true
    setLoadingList(true)
    setError(null)

    api.fetchVersions(pageId).then((data) => {
      if (!active) return
      const sorted = [...(data ?? [])].sort((a, b) => b.version - a.version)
      setVersions(sorted)

      const nextVersion = sorted[0]?.version ?? null
      setSelectedVersion((current) => current ?? nextVersion)
      if (data === null) {
        setError(t('versions.loadError'))
      }
      setLoadingList(false)
    })

    return () => {
      active = false
    }
  }, [expanded, pageId, t])

  useEffect(() => {
    if (!expanded || selectedVersion === null) return

    let active = true
    setLoadingVersion(true)
    setError(null)

    api.fetchVersion(pageId, selectedVersion).then((data) => {
      if (!active) return
      setSelectedData(data)
      if (data === null) {
        setError(t('versions.previewError'))
      }
      setLoadingVersion(false)
    })

    return () => {
      active = false
    }
  }, [expanded, pageId, selectedVersion, t])

  const handleRestore = async () => {
    if (selectedVersion === null || restoring) return

    setRestoring(true)
    setError(null)

    const restored = await api.restoreVersion(pageId, selectedVersion)
    if (!restored) {
      setError(t('versions.restoreError'))
      setRestoring(false)
      return
    }

    const page = await api.fetchPage(pageId)
    if (!page) {
      setError(t('versions.refreshError'))
      setRestoring(false)
      return
    }

    await onRestore(page)
    setRestoring(false)
  }

  return (
    <section className={`version-history ${embedded ? 'is-embedded' : ''}`}>
      {embedded ? (
        <div className="version-history-static-head">
          <div>
            <div className="version-history-title">{t('versions.title')}</div>
            <div className="version-history-subtitle">
              {t('versions.current', { version: currentVersion ?? 0 })}
            </div>
          </div>
        </div>
      ) : (
        <button className="version-history-toggle" onClick={() => setExpanded((value) => !value)}>
          <div>
            <div className="version-history-title">{t('versions.title')}</div>
            <div className="version-history-subtitle">
              {t('versions.current', { version: currentVersion ?? 0 })}
            </div>
          </div>
          <span className={`version-history-caret ${expanded ? 'is-open' : ''}`}>▾</span>
        </button>
      )}

      {expanded && (
        <div className="version-history-body">
          {error && <div className="version-history-error">{error}</div>}

          <div className="version-history-grid">
            <div className="version-history-list">
              {loadingList && (
                <div className="version-history-empty">{t('versions.loading')}</div>
              )}

              {!loadingList && versions.length === 0 && (
                <div className="version-history-empty">{t('versions.empty')}</div>
              )}

              {versions.map((version) => (
                <button
                  key={version.version}
                  className={`version-item ${selectedVersion === version.version ? 'is-active' : ''}`}
                  onClick={() => setSelectedVersion(version.version)}
                >
                  <span className="version-item-number">
                    {t('versions.versionLabel', { version: version.version })}
                  </span>
                  <span className="version-item-date">
                    {new Date(version.savedAt).toLocaleString()}
                  </span>
                </button>
              ))}
            </div>

            <div className="version-preview">
              {loadingVersion && (
                <div className="version-history-empty">{t('versions.previewLoading')}</div>
              )}

              {!loadingVersion && selectedData && (
                <>
                  <div className="version-preview-header">
                    <div>
                      <div className="version-preview-title">{selectedData.title || t('editor.untitled')}</div>
                      <div className="version-preview-date">
                        {new Date(selectedData.savedAt).toLocaleString()}
                      </div>
                    </div>
                    <button
                      className="version-restore-btn"
                      onClick={handleRestore}
                      disabled={selectedData.version === currentVersion || restoring}
                    >
                      {restoring ? t('versions.restoring') : t('versions.restore')}
                    </button>
                  </div>

                  <div className="version-preview-badge">
                    {selectedData.version === currentVersion
                      ? t('versions.currentBadge')
                      : t('versions.versionLabel', { version: selectedData.version })}
                  </div>

                  <div className="version-preview-text">
                    {versionSnippet(selectedData) || t('versions.noPreview')}
                  </div>
                </>
              )}
            </div>
          </div>
        </div>
      )}
    </section>
  )
}

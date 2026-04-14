import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { WikiGraph } from '../../types'
import { api } from '../../utils/api'
import { useWikiStore } from '../../stores/wikiStore'
import './GraphInsights.css'

interface GraphInsightsProps {
  pageId: string
  compact?: boolean
}

export function usePageGraph(pageId: string) {
  const pages = useWikiStore((state) => state.pages)
  const [graph, setGraph] = useState<WikiGraph | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let active = true
    setLoading(true)
    api.fetchGraph().then((data) => {
      if (!active) return
      setGraph(data)
      setLoading(false)
    })
    return () => {
      active = false
    }
  }, [pageId, pages])

  const nodeTitles = useMemo(() => {
    const map = new Map<string, string>()
    pages.forEach((page) => map.set(page.id, page.title))
    graph?.nodes.forEach((node) => map.set(node.id, node.title))
    return map
  }, [graph, pages])

  const outgoing = useMemo(
    () => (graph?.edges ?? []).filter((edge) => edge.source === pageId),
    [graph, pageId]
  )
  const incoming = useMemo(
    () => (graph?.edges ?? []).filter((edge) => edge.target === pageId),
    [graph, pageId]
  )

  return {
    graph,
    loading,
    incoming,
    outgoing,
    nodeTitles,
  }
}

export function GraphInsights({ pageId, compact = false }: GraphInsightsProps) {
  const { t } = useTranslation()
  const setCurrentPage = useWikiStore((state) => state.setCurrentPage)
  const { graph, loading, incoming, outgoing, nodeTitles } = usePageGraph(pageId)

  return (
    <section className={`graph-insights ${compact ? 'is-compact' : ''}`}>
      <div className="graph-insights-header">
        <div>
          <div className="graph-insights-title">{t('graph.title')}</div>
          {compact && <div className="graph-insights-subtitle">{t('graph.subtitle')}</div>}
        </div>
        <div className="graph-insights-stats">
          <span>{t('graph.incoming', { count: incoming.length })}</span>
          <span>{t('graph.outgoing', { count: outgoing.length })}</span>
        </div>
      </div>

      {loading && <div className="graph-insights-empty">{t('graph.loading')}</div>}
      {!loading && !graph && <div className="graph-insights-empty">{t('graph.empty')}</div>}

      {!loading && graph && (
        <div className="graph-insights-grid">
          <div>
            <div className="graph-insights-label">{t('graph.references')}</div>
            <div className="graph-insights-list">
              {outgoing.length === 0 && <div className="graph-insights-empty">{t('graph.none')}</div>}
              {outgoing.map((edge) => (
                <button key={`${edge.source}-${edge.target}`} type="button" onClick={() => setCurrentPage(edge.target)}>
                  {nodeTitles.get(edge.target) || edge.target}
                </button>
              ))}
            </div>
          </div>
          <div>
            <div className="graph-insights-label">{t('graph.referencedBy')}</div>
            <div className="graph-insights-list">
              {incoming.length === 0 && <div className="graph-insights-empty">{t('graph.none')}</div>}
              {incoming.map((edge) => (
                <button key={`${edge.source}-${edge.target}`} type="button" onClick={() => setCurrentPage(edge.source)}>
                  {nodeTitles.get(edge.source) || edge.source}
                </button>
              ))}
            </div>
          </div>
        </div>
      )}
    </section>
  )
}

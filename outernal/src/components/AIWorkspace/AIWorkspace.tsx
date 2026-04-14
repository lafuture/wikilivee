import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { Editor } from '@tiptap/react'
import type { WikiPage } from '../../types'
import { api } from '../../utils/api'
import './AIWorkspace.css'

interface AIWorkspaceProps {
  page: WikiPage
  editor: Editor | null
  compact?: boolean
}

function extractPlainText(value: unknown): string {
  if (!value || typeof value !== 'object') return ''
  const node = value as { text?: string; content?: unknown[] }
  if (typeof node.text === 'string') return node.text
  if (!Array.isArray(node.content)) return ''
  return node.content.map((item) => extractPlainText(item)).join(' ').trim()
}

export function AIWorkspace({ page, editor, compact = false }: AIWorkspaceProps) {
  const { t } = useTranslation()
  const [prompt, setPrompt] = useState('')
  const [output, setOutput] = useState('')
  const [loadingAction, setLoadingAction] = useState<'complete' | 'summarize' | 'suggest' | null>(null)

  const pageContext = useMemo(() => extractPlainText(page.content).replace(/\s+/g, ' ').trim(), [page.content])

  const runAction = async (action: 'complete' | 'summarize' | 'suggest') => {
    if (loadingAction) return
    setLoadingAction(action)

    let result: string | null = null
    if (action === 'complete') {
      result = await api.completeText(prompt.trim(), pageContext || null)
    } else if (action === 'summarize') {
      result = await api.summarizePage(page.id)
    } else {
      result = await api.suggestNextBlock(pageContext)
    }

    setLoadingAction(null)
    if (result) {
      setOutput(result)
    } else {
      setOutput(t('ai.error'))
    }
  }

  const insertOutput = () => {
    if (!editor || !output || output === t('ai.error')) return
    editor.chain().focus().insertContent([
      { type: 'paragraph', content: [{ type: 'text', text: output }] },
    ]).run()
  }

  return (
    <section className={`ai-workspace ${compact ? 'is-compact' : ''}`}>
      <div className="ai-workspace-header">
        <div>
          <div className="ai-workspace-title">{t('ai.title')}</div>
          <div className="ai-workspace-subtitle">{t('ai.subtitle')}</div>
        </div>
        <div className="ai-workspace-actions">
          <button type="button" onClick={() => runAction('summarize')} disabled={Boolean(loadingAction)}>
            {loadingAction === 'summarize' ? t('ai.loading') : t('ai.summarize')}
          </button>
          <button type="button" onClick={() => runAction('suggest')} disabled={Boolean(loadingAction)}>
            {loadingAction === 'suggest' ? t('ai.loading') : t('ai.suggest')}
          </button>
        </div>
      </div>

      <label className="ai-workspace-prompt">
        <span>{t('ai.prompt')}</span>
        <textarea
          rows={3}
          value={prompt}
          onChange={(event) => setPrompt(event.target.value)}
          placeholder={t('ai.promptPlaceholder')}
        />
      </label>

      <div className="ai-workspace-footer">
        <button
          type="button"
          className="ai-workspace-generate"
          onClick={() => runAction('complete')}
          disabled={!prompt.trim() || Boolean(loadingAction)}
        >
          {loadingAction === 'complete' ? t('ai.loading') : t('ai.complete')}
        </button>
        <button
          type="button"
          className="ai-workspace-insert"
          onClick={insertOutput}
          disabled={!output || output === t('ai.error')}
        >
          {t('ai.insert')}
        </button>
      </div>

      <div className="ai-workspace-output">
        {output || t('ai.empty')}
      </div>
    </section>
  )
}

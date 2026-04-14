import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { Editor } from '@tiptap/react'
import type { WikiComment } from '../../types'
import { api } from '../../utils/api'
import './Comments.css'

interface CommentsProps {
  pageId: string
  editor: Editor | null
}

const AUTHOR_STORAGE_KEY = 'mws-wiki-comment-author'

function readSelectionAnchor(editor: Editor | null) {
  if (!editor) return null

  const { from, to, empty } = editor.state.selection
  if (empty || from === to) return null

  const anchorText = editor.state.doc.textBetween(from, to, ' ').trim()
  return {
    anchorFrom: from,
    anchorTo: to,
    anchorText,
  }
}

export function Comments({ pageId, editor }: CommentsProps) {
  const { t } = useTranslation()
  const [comments, setComments] = useState<WikiComment[]>([])
  const [author, setAuthor] = useState(() => localStorage.getItem(AUTHOR_STORAGE_KEY) || '')
  const [text, setText] = useState('')
  const [selectionAnchor, setSelectionAnchor] = useState(() => readSelectionAnchor(editor))
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let active = true

    setLoading(true)
    setError(null)

    api.fetchComments(pageId).then((data) => {
      if (!active) return
      setComments(data ?? [])
      if (data === null) {
        setError(t('comments.loadError'))
      }
      setLoading(false)
    })

    return () => {
      active = false
    }
  }, [pageId, t])

  useEffect(() => {
    if (!editor) {
      setSelectionAnchor(null)
      return
    }

    const syncSelection = () => {
      setSelectionAnchor(readSelectionAnchor(editor))
    }

    syncSelection()
    editor.on('selectionUpdate', syncSelection)
    editor.on('update', syncSelection)

    return () => {
      editor.off('selectionUpdate', syncSelection)
      editor.off('update', syncSelection)
    }
  }, [editor, pageId])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const trimmedAuthor = author.trim()
    const trimmedText = text.trim()

    if (!trimmedAuthor || !trimmedText || submitting) return

    setSubmitting(true)
    setError(null)
    localStorage.setItem(AUTHOR_STORAGE_KEY, trimmedAuthor)

    const created = await api.createComment(pageId, trimmedAuthor, trimmedText, selectionAnchor)
    if (!created) {
      setError(t('comments.submitError'))
      setSubmitting(false)
      return
    }

    setComments((current) => [...current, created])
    setText('')
    setSubmitting(false)
  }

  const handleJumpToAnchor = (comment: WikiComment) => {
    if (!editor || comment.anchorFrom <= 0 && comment.anchorTo <= 0) return

    const maxPosition = editor.state.doc.content.size
    const from = Math.max(0, Math.min(comment.anchorFrom, maxPosition))
    const to = Math.max(0, Math.min(comment.anchorTo, maxPosition))
    editor.chain().focus().setTextSelection({ from, to }).run()
  }

  const handleDelete = async (commentId: string) => {
    const ok = await api.deleteComment(pageId, commentId)
    if (!ok) {
      setError(t('comments.deleteError'))
      return
    }

    setComments((current) => current.filter((comment) => comment.id !== commentId))
  }

  return (
    <section className="comments">
      <div className="comments-header">{t('comments.title')}</div>

      <form className="comments-form" onSubmit={handleSubmit}>
        <div className={`comments-anchor ${selectionAnchor ? 'is-active' : ''}`}>
          <div className="comments-anchor-label">
            {selectionAnchor ? t('comments.selectionAttached') : t('comments.selectionEmpty')}
          </div>
          {selectionAnchor && (
            <button
              type="button"
              className="comments-anchor-preview"
              onClick={() => {
                if (!editor) return
                editor.chain().focus().setTextSelection({
                  from: selectionAnchor.anchorFrom,
                  to: selectionAnchor.anchorTo,
                }).run()
              }}
            >
              {selectionAnchor.anchorText || t('comments.selectionFallback')}
            </button>
          )}
        </div>
        <input
          className="comments-author"
          value={author}
          onChange={(e) => setAuthor(e.target.value)}
          placeholder={t('comments.authorPlaceholder')}
        />
        <textarea
          className="comments-text"
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder={t('comments.textPlaceholder')}
          rows={3}
        />
        <button
          className="comments-submit"
          type="submit"
          disabled={!author.trim() || !text.trim() || submitting}
        >
          {submitting ? t('comments.submitting') : t('comments.add')}
        </button>
      </form>

      {error && <div className="comments-error">{error}</div>}

      <div className="comments-list">
        {!loading && comments.length === 0 && (
          <div className="comments-empty">{t('comments.empty')}</div>
        )}

        {comments.map((comment) => (
          <article key={comment.id} className="comment-item">
            <div className="comment-meta">
              <div className="comment-author">{comment.author}</div>
              <div className="comment-date">
                {new Date(comment.createdAt).toLocaleString()}
              </div>
            </div>
            {(comment.anchorText || comment.anchorFrom || comment.anchorTo) && (
              <button
                type="button"
                className="comment-anchor"
                onClick={() => handleJumpToAnchor(comment)}
              >
                {comment.anchorText || t('comments.selectionFallback')}
              </button>
            )}
            <div className="comment-text">{comment.text}</div>
            <button
              className="comment-delete"
              type="button"
              onClick={() => handleDelete(comment.id)}
            >
              {t('comments.delete')}
            </button>
          </article>
        ))}
      </div>
    </section>
  )
}

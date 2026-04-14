import { useEffect, useRef, useState, useCallback, type CSSProperties } from 'react'
import { useTranslation } from 'react-i18next'
import { useEditor, EditorContent, type Editor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import Link from '@tiptap/extension-link'
import { WikiLink } from '../../extensions/WikiLink'
import { TableEmbed } from '../../extensions/TableEmbed'
import { SlashCommands } from '../../extensions/SlashCommands'
import { RemotePresence, remotePresencePluginKey, type RemotePresenceCursor } from '../../extensions/RemotePresence'
import { slashSuggestion } from '../../utils/slashSuggestion'
import { createAutosave } from '../../utils/autosave'
import { useCollaboration } from '../../utils/useCollaboration'
import { useWikiStore } from '../../stores/wikiStore'
import { apiBlocksToTiptap } from '../../utils/api'
import { BubbleToolbar } from './BubbleToolbar'
import { IconPicker } from './IconPicker'
import { PageCover } from './PageCover'
import { Backlinks } from '../Backlinks/Backlinks'
import { Comments } from '../Comments/Comments'
import { VersionHistory } from '../VersionHistory/VersionHistory'
import { TablePicker } from '../TableEmbed/TablePicker'
import { PageLinkPicker } from '../TableEmbed/PageLinkPicker'
import { ChildPages } from '../ChildPages/ChildPages'
import { AIWorkspace } from '../AIWorkspace/AIWorkspace'
import { GraphInsights, usePageGraph } from '../GraphInsights/GraphInsights'
import './WikiEditor.css'

interface WikiEditorProps {
  pageId: string
}

type UtilityPanel = 'graph' | 'children' | 'ai' | 'versions' | null
type UtilityPanelKey = Exclude<UtilityPanel, null>

function getUserInitials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? '')
    .join('')
}

export function WikiEditor({ pageId }: WikiEditorProps) {
  const { t } = useTranslation()
  const pages = useWikiStore((s) => s.pages)
  const updatePage = useWikiStore((s) => s.updatePage)
  const updatePageMeta = useWikiStore((s) => s.updatePageMeta)
  const hydrateRemotePage = useWikiStore((s) => s.hydrateRemotePage)
  const setCurrentPage = useWikiStore((s) => s.setCurrentPage)

  const page = pages.find((p) => p.id === pageId)
  const [showTablePicker, setShowTablePicker] = useState(false)
  const [showPageLinkPicker, setShowPageLinkPicker] = useState(false)
  const [showIconPicker, setShowIconPicker] = useState(false)
  const [activeUtilityPanel, setActiveUtilityPanel] = useState<UtilityPanel>(null)
  const [selectedUtilityPanel, setSelectedUtilityPanel] = useState<UtilityPanelKey>('graph')
  const [title, setTitle] = useState(page?.title || '')
  const utilityDockRef = useRef<HTMLDivElement | null>(null)
  const titleRef = useRef(title)
  titleRef.current = title
  const [saveStatus, setSaveStatus] = useState<'saved' | 'saving' | 'idle'>('idle')
  const dirtyRef = useRef(false)
  const versionRef = useRef(page?.version ?? 0)
  const hideTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const showSavedTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const lastSelectionSentRef = useRef('')
  versionRef.current = page?.version ?? versionRef.current

  const autosaveRef = useRef(
    createAutosave((content) => {
      updatePage(pageId, content, titleRef.current)
      showSavedTimerRef.current = setTimeout(() => {
        setSaveStatus('saved')
        if (hideTimerRef.current) clearTimeout(hideTimerRef.current)
        hideTimerRef.current = setTimeout(() => {
          setSaveStatus('idle')
          dirtyRef.current = false
        }, 4000)
      }, 500)
    }, 500)
  )

  const isUpdatingFromRemote = useRef(false)
  const { connected: collabConnected, self, peers, remoteSelections, sendUpdate, sendCursor } = useCollaboration({
    pageId,
    onRemoteUpdate: (event) => {
      if (!event.payload || !editor) return
      isUpdatingFromRemote.current = true
      versionRef.current = event.payload.version
      const newContent = apiBlocksToTiptap(event.payload.content)
      const nextTitle = event.payload.title

      titleRef.current = nextTitle
      editor.commands.setContent(newContent)
      if (nextTitle !== title) {
        setTitle(nextTitle)
      }
      void hydrateRemotePage({
        id: pageId,
        title: nextTitle,
        parentId: page?.parentId ?? null,
        icon: page?.icon ?? '📄',
        cover: page?.cover ?? null,
        content: newContent,
        createdAt: page?.createdAt ?? Date.now(),
        updatedAt: Date.now(),
        version: event.payload.version,
      })
      isUpdatingFromRemote.current = false
    },
  })

  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        link: false,
      }),
      Placeholder.configure({ placeholder: t('editor.placeholder') }),
      Link.configure({ openOnClick: false }),
      WikiLink,
      TableEmbed,
      RemotePresence,
      SlashCommands.configure({
        suggestion: {
          ...slashSuggestion(),
          char: '/',
          items: ({ query }: { query: string }) => {
            const allItems = [
              { title: t('slash.text'), description: t('slash.textDesc'), icon: 'T', command: ({ editor, range }: any) => { editor.chain().focus().deleteRange(range).setParagraph().run() } },
              { title: t('slash.heading1'), description: t('slash.heading1Desc'), icon: 'H1', command: ({ editor, range }: any) => { editor.chain().focus().deleteRange(range).setHeading({ level: 1 }).run() } },
              { title: t('slash.heading2'), description: t('slash.heading2Desc'), icon: 'H2', command: ({ editor, range }: any) => { editor.chain().focus().deleteRange(range).setHeading({ level: 2 }).run() } },
              { title: t('slash.heading3'), description: t('slash.heading3Desc'), icon: 'H3', command: ({ editor, range }: any) => { editor.chain().focus().deleteRange(range).setHeading({ level: 3 }).run() } },
              { title: t('slash.bulletList'), description: t('slash.bulletListDesc'), icon: '•', command: ({ editor, range }: any) => { editor.chain().focus().deleteRange(range).toggleBulletList().run() } },
              { title: t('slash.numberedList'), description: t('slash.numberedListDesc'), icon: '1.', command: ({ editor, range }: any) => { editor.chain().focus().deleteRange(range).toggleOrderedList().run() } },
              { title: t('slash.table'), description: t('slash.tableDesc'), icon: '📊', command: ({ editor, range }: any) => { editor.chain().focus().deleteRange(range).run(); setShowTablePicker(true) } },
              { title: t('slash.pageLink'), description: t('slash.pageLinkDesc'), icon: '🔗', command: ({ editor, range }: any) => { editor.chain().focus().deleteRange(range).run(); setShowPageLinkPicker(true) } },
            ]
            return allItems.filter((item) => item.title.toLowerCase().includes(query.toLowerCase()))
          },
          command: ({ editor, range, props }: any) => {
            props.command({ editor, range })
          },
        } as any,
      }),
    ],
    content: page?.content || { type: 'doc', content: [{ type: 'paragraph' }] },
    onUpdate: ({ editor }) => {
      if (isUpdatingFromRemote.current) return
      dirtyRef.current = true
      if (showSavedTimerRef.current) clearTimeout(showSavedTimerRef.current)
      if (hideTimerRef.current) clearTimeout(hideTimerRef.current)
      setSaveStatus('saving')
      sendUpdate(titleRef.current, editor.getJSON(), versionRef.current)
      autosaveRef.current.save(editor.getJSON())
    },
    onSelectionUpdate: ({ editor }) => {
      const { anchor, head } = editor.state.selection
      const nextKey = `${anchor}:${head}`
      if (nextKey === lastSelectionSentRef.current) return
      lastSelectionSentRef.current = nextKey
      sendCursor(anchor, head)
    },
    onCreate: ({ editor }) => {
      const { anchor, head } = editor.state.selection
      lastSelectionSentRef.current = `${anchor}:${head}`
      sendCursor(anchor, head)
    },
  }, [pageId, sendUpdate])

  useEffect(() => {
    if (page) setTitle(page.title)
  }, [pageId, page?.title])

  useEffect(() => {
    lastSelectionSentRef.current = ''
  }, [pageId])

  useEffect(() => {
    setActiveUtilityPanel(null)
    setSelectedUtilityPanel('graph')
  }, [pageId])

  const handleTitleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setTitle(e.target.value)
      if (editor) {
        dirtyRef.current = true
        if (showSavedTimerRef.current) clearTimeout(showSavedTimerRef.current)
        if (hideTimerRef.current) clearTimeout(hideTimerRef.current)
        setSaveStatus('saving')
        sendUpdate(e.target.value, editor.getJSON(), versionRef.current)
        autosaveRef.current.save(editor.getJSON())
      }
    },
    [editor, sendUpdate]
  )

  const handleTitleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Enter') {
        e.preventDefault()
        editor?.commands.focus('start')
      }
    },
    [editor]
  )

  useEffect(() => {
    const handler = (e: Event) => {
      const detail = (e as CustomEvent).detail
      if (detail?.pageId) setCurrentPage(detail.pageId)
    }
    window.addEventListener('wiki-navigate', handler)
    return () => window.removeEventListener('wiki-navigate', handler)
  }, [setCurrentPage])

  useEffect(() => {
    return () => {
      if (showSavedTimerRef.current) clearTimeout(showSavedTimerRef.current)
      if (hideTimerRef.current) clearTimeout(hideTimerRef.current)
      if (editor && dirtyRef.current) {
        updatePage(pageId, editor.getJSON(), titleRef.current)
      }
      autosaveRef.current.cancel()
    }
  }, [editor, pageId])

  useEffect(() => {
    if (!editor) return

    const cursors: RemotePresenceCursor[] = remoteSelections.map(({ updatedAt: _updatedAt, ...cursor }) => cursor)
    editor.view.dispatch(editor.state.tr.setMeta(remotePresencePluginKey, { cursors }))
  }, [editor, remoteSelections])

  useEffect(() => {
    if (!activeUtilityPanel) return

    const handlePointerDown = (event: MouseEvent) => {
      if (utilityDockRef.current && !utilityDockRef.current.contains(event.target as Node)) {
        setActiveUtilityPanel(null)
      }
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setActiveUtilityPanel(null)
      }
    }

    document.addEventListener('mousedown', handlePointerDown)
    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('mousedown', handlePointerDown)
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [activeUtilityPanel])

  const handleTableSelect = useCallback(
    (tableId: string, tableName: string) => {
      editor?.chain().focus().insertContent({ type: 'tableEmbed', attrs: { tableId, tableName } }).run()
      setShowTablePicker(false)
    },
    [editor]
  )

  const handlePageLinkSelect = useCallback(
    (linkedPageId: string, pageTitle: string) => {
      editor?.chain().focus().insertContent({ type: 'wikiLink', attrs: { pageId: linkedPageId, pageTitle } }).run()
      setShowPageLinkPicker(false)
    },
    [editor]
  )

  const handleVersionRestore = useCallback(async (restoredPage: typeof page) => {
    if (!restoredPage || !editor) return

    await hydrateRemotePage(restoredPage)

    isUpdatingFromRemote.current = true
    versionRef.current = restoredPage.version ?? versionRef.current
    titleRef.current = restoredPage.title
    setTitle(restoredPage.title)
    editor.commands.setContent(restoredPage.content)
    isUpdatingFromRemote.current = false
    dirtyRef.current = false
    setSaveStatus('saved')
  }, [editor, hydrateRemotePage])

  const collabUsers = self ? [self, ...peers] : peers
  const childPageCount = pages.filter((candidate) => candidate.parentId === pageId).length
  const { incoming, outgoing } = usePageGraph(pageId)
  const graphLinkCount = incoming.length + outgoing.length

  const toggleUtilityPanel = useCallback((panel: UtilityPanelKey) => {
    setSelectedUtilityPanel(panel)
    setActiveUtilityPanel(panel)
  }, [])

  const currentUtilityPanel = activeUtilityPanel ?? selectedUtilityPanel
  const utilityPanelTitles = {
    graph: t('graph.title'),
    children: t('children.title'),
    ai: t('ai.title'),
    versions: t('versions.title'),
  } as const

  if (!page) return (
    <div className="editor-empty">
      <div className="spinner" />
    </div>
  )

  return (
    <div className="wiki-editor-wrapper">
      <PageCover
        cover={page.cover}
        onSelect={(cover) => updatePageMeta(pageId, { cover })}
      />
      <div className="wiki-editor">
        <div className="editor-layout">
          <aside className={`editor-utility-dock ${activeUtilityPanel ? 'is-open' : ''}`} ref={utilityDockRef}>
            <button
              type="button"
              className="editor-utility-trigger"
              aria-expanded={Boolean(activeUtilityPanel)}
              aria-label={activeUtilityPanel ? t('editor.closeUtilityMenu') : t('editor.openUtilityMenu')}
              onClick={() => setActiveUtilityPanel((current) => current ? null : selectedUtilityPanel)}
            >
              <span className="editor-utility-trigger-icon">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M4 6h16"/>
                  <path d="M4 12h10"/>
                  <path d="M4 18h16"/>
                </svg>
              </span>
              <span className="editor-utility-trigger-copy">
                <span className="editor-utility-trigger-label">{t('editor.pageTools')}</span>
                <span className="editor-utility-trigger-meta">
                  {t('graph.outgoing', { count: outgoing.length })} · {t('graph.incoming', { count: incoming.length })}
                </span>
              </span>
              <span className="editor-utility-trigger-caret">▾</span>
            </button>

            {activeUtilityPanel && (
              <div className="editor-utility-panel">
                <div className="editor-utility-panel-head">
                  <div className="editor-utility-page">
                    <span className="editor-utility-page-icon">{page.icon || '📄'}</span>
                    <div className="editor-utility-page-copy">
                      <span className="editor-utility-panel-eyebrow">{t('editor.pageTools')}</span>
                      <span className="editor-utility-panel-title">{page.title || t('editor.untitled')}</span>
                      <span className="editor-utility-panel-subtitle">{utilityPanelTitles[currentUtilityPanel]}</span>
                    </div>
                  </div>
                  <button
                    type="button"
                    className="editor-utility-close"
                    aria-label={t('editor.closeUtilityMenu')}
                    onClick={() => setActiveUtilityPanel(null)}
                  >
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/>
                    </svg>
                  </button>
                </div>

                <div className="editor-utility-overview">
                  <div className="editor-utility-stat">
                    <span className="editor-utility-stat-label">{t('graph.outgoingShort')}</span>
                    <strong>{outgoing.length}</strong>
                  </div>
                  <div className="editor-utility-stat">
                    <span className="editor-utility-stat-label">{t('graph.incomingShort')}</span>
                    <strong>{incoming.length}</strong>
                  </div>
                  <div className="editor-utility-stat">
                    <span className="editor-utility-stat-label">{t('children.short')}</span>
                    <strong>{childPageCount}</strong>
                  </div>
                  <div className="editor-utility-stat">
                    <span className="editor-utility-stat-label">{t('versions.short')}</span>
                    <strong>v{page.version ?? 0}</strong>
                  </div>
                </div>

                <div className="editor-utility-tabs" role="tablist" aria-label={t('editor.pageTools')}>
                  <button
                    type="button"
                    role="tab"
                    aria-selected={currentUtilityPanel === 'graph'}
                    className={`editor-utility-tab ${currentUtilityPanel === 'graph' ? 'is-active' : ''}`}
                    onClick={() => toggleUtilityPanel('graph')}
                  >
                    <span>{t('graph.title')}</span>
                    {graphLinkCount > 0 && <span className="editor-utility-tab-badge">{graphLinkCount}</span>}
                  </button>
                  <button
                    type="button"
                    role="tab"
                    aria-selected={currentUtilityPanel === 'children'}
                    className={`editor-utility-tab ${currentUtilityPanel === 'children' ? 'is-active' : ''}`}
                    onClick={() => toggleUtilityPanel('children')}
                  >
                    <span>{t('children.title')}</span>
                    {childPageCount > 0 && <span className="editor-utility-tab-badge">{childPageCount}</span>}
                  </button>
                  <button
                    type="button"
                    role="tab"
                    aria-selected={currentUtilityPanel === 'ai'}
                    className={`editor-utility-tab ${currentUtilityPanel === 'ai' ? 'is-active is-ai' : ''}`}
                    onClick={() => toggleUtilityPanel('ai')}
                  >
                    <span>{t('ai.title')}</span>
                  </button>
                  <button
                    type="button"
                    role="tab"
                    aria-selected={currentUtilityPanel === 'versions'}
                    className={`editor-utility-tab ${currentUtilityPanel === 'versions' ? 'is-active' : ''}`}
                    onClick={() => toggleUtilityPanel('versions')}
                  >
                    <span>{t('versions.title')}</span>
                    <span className="editor-utility-tab-badge">v{page.version ?? 0}</span>
                  </button>
                </div>

                <div className="editor-utility-panel-body">
                  {activeUtilityPanel === 'graph' && <GraphInsights pageId={pageId} compact />}
                  {activeUtilityPanel === 'children' && <ChildPages pageId={pageId} compact />}
                  {activeUtilityPanel === 'ai' && <AIWorkspace page={page} editor={editor} compact />}
                  {activeUtilityPanel === 'versions' && (
                    <VersionHistory
                      pageId={pageId}
                      currentVersion={page.version}
                      onRestore={handleVersionRestore}
                      embedded
                    />
                  )}
                </div>
              </div>
            )}
          </aside>

          <div className="editor-main-column">
            <div className={`page-icon-area ${page.cover ? 'has-cover' : ''}`}>
              <button className="page-icon-btn" onClick={() => setShowIconPicker(true)}>
                {page.icon || '📄'}
              </button>
            </div>

            <div className="editor-header">
              <input
                className="editor-title"
                value={title}
                onChange={handleTitleChange}
                onKeyDown={handleTitleKeyDown}
                placeholder={t('editor.untitled')}
              />
              <div className="editor-meta">
                <span className="editor-meta-date">
                  {t('editor.edited')} {new Date(page.updatedAt).toLocaleString(
                    undefined, { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }
                  )}
                </span>
                {saveStatus !== 'idle' && (
                  <>
                    <span className="editor-meta-sep" />
                    <span className={`save-indicator save-indicator--${saveStatus}`}>
                      <span className="save-dot" />
                      {saveStatus === 'saving' && t('editor.saving')}
                      {saveStatus === 'saved' && t('editor.saved')}
                    </span>
                  </>
                )}
                {(self || collabConnected || peers.length > 0) && (
                  <>
                    <span className="editor-meta-sep" />
                    <span className={`collab-indicator ${collabConnected ? '' : 'is-offline'}`}>
                      <span className="collab-dot" />
                      {collabConnected ? t('editor.live') : t('editor.offline')}
                    </span>
                    <div className="collab-users">
                      {collabUsers.map((user) => (
                        <div
                          key={user.userId}
                          className={`collab-user ${user.userId === self?.userId ? 'is-self' : ''}`}
                          style={{ '--user-color': user.color } as CSSProperties}
                          title={user.name}
                        >
                          {getUserInitials(user.name)}
                        </div>
                      ))}
                    </div>
                  </>
                )}
              </div>
            </div>

            {editor && <BubbleToolbar editor={editor} />}
            <EditorContent editor={editor} className="editor-content" />
            <Backlinks pageId={pageId} />
            <Comments pageId={pageId} editor={editor as Editor | null} />
          </div>
        </div>
      </div>

      {showIconPicker && (
        <IconPicker
          current={page.icon || '📄'}
          onSelect={(icon) => updatePageMeta(pageId, { icon })}
          onClose={() => setShowIconPicker(false)}
        />
      )}
      {showTablePicker && (
        <TablePicker onSelect={handleTableSelect} onClose={() => setShowTablePicker(false)} />
      )}
      {showPageLinkPicker && (
        <PageLinkPicker currentPageId={pageId} onSelect={handlePageLinkSelect} onClose={() => setShowPageLinkPicker(false)} />
      )}
    </div>
  )
}

import { create } from 'zustand'
import type { JSONContent } from '@tiptap/react'
import type { WikiPage, Backlink } from '../types'
import { set as idbSet, get as idbGet } from 'idb-keyval'
import { api } from '../utils/api'
import i18n from '../i18n'
import { useToastStore } from './toastStore'

const STORAGE_KEY = 'mws-wiki-pages'
let lastSaveErrorAt = 0

function pushErrorToast(message: string) {
  useToastStore.getState().pushToast(message, 'error')
}

interface WikiStore {
  pages: WikiPage[]
  currentPageId: string | null
  loaded: boolean
  apiMode: boolean

  loadPages: () => Promise<void>
  refreshPages: () => Promise<void>
  persistPages: (pages: WikiPage[]) => Promise<void>
  hydrateRemotePage: (page: WikiPage) => Promise<void>
  createPage: (title?: string, parentId?: string | null) => Promise<WikiPage | null>
  updatePage: (id: string, content: JSONContent, title?: string) => void
  updatePageMeta: (id: string, meta: Partial<Pick<WikiPage, 'icon' | 'cover'>>) => void
  deletePage: (id: string) => Promise<boolean>
  setCurrentPage: (id: string | null) => Promise<void>
  getBacklinks: (pageId: string) => Backlink[]
  fetchBacklinks: (pageId: string) => Promise<Backlink[]>
}

function generateId(): string {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 8)
}

const PAGE_ICONS = ['📄', '📝', '📋', '📒', '📓', '📔', '📕', '📗', '📘', '📙', '🗒️', '💡', '🎯', '🚀', '⭐', '🔖']
function randomIcon(): string {
  return PAGE_ICONS[Math.floor(Math.random() * PAGE_ICONS.length)]
}

function createEmptyDoc(): JSONContent {
  return { type: 'doc', content: [{ type: 'paragraph' }] }
}

function extractPageLinks(content: JSONContent): string[] {
  const links: string[] = []
  function walk(node: JSONContent) {
    if (node.type === 'wikiLink' && node.attrs?.pageId) {
      links.push(node.attrs.pageId)
    }
    if (node.content) {
      node.content.forEach(walk)
    }
  }
  walk(content)
  return links
}

async function loadLocal(): Promise<WikiPage[]> {
  try {
    const stored = await idbGet(STORAGE_KEY)
    if (stored) return stored
  } catch {}
  const raw = localStorage.getItem(STORAGE_KEY)
  if (raw) return JSON.parse(raw)
  return []
}

async function persistLocal(pages: WikiPage[]) {
  try {
    await idbSet(STORAGE_KEY, pages)
  } catch {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(pages))
  }
}

function mergeRemotePages(remoteSummaries: NonNullable<Awaited<ReturnType<typeof api.fetchPages>>>, currentPages: WikiPage[]): WikiPage[] {
  const localById = new Map(currentPages.map((page) => [page.id, page]))

  return remoteSummaries.map((summary) => {
    const localPage = localById.get(summary.id)
    return {
      id: summary.id,
      title: summary.title,
      parentId: summary.parent_id || localPage?.parentId || null,
      icon: summary.icon ?? localPage?.icon ?? '📄',
      cover: localPage?.cover ?? null,
      content: localPage?.content ?? createEmptyDoc(),
      createdAt: localPage?.createdAt ?? new Date(summary.updatedAt).getTime(),
      updatedAt: new Date(summary.updatedAt).getTime(),
      version: summary.version ?? localPage?.version,
    } satisfies WikiPage
  })
}

export const useWikiStore = create<WikiStore>((set, get) => ({
  pages: [],
  currentPageId: null,
  loaded: false,
  apiMode: false,

  loadPages: async () => {
    const localPages = await loadLocal()
    const urlPageId = new URLSearchParams(window.location.search).get('page')
    let pages = localPages
    let apiUp = false

    const remoteSummaries = await api.fetchPages()
    if (remoteSummaries !== null) {
      apiUp = true
      pages = mergeRemotePages(remoteSummaries, localPages)
    }

    if (urlPageId) {
      const remotePage = await api.fetchPage(urlPageId)
      if (remotePage) {
        const existingPage = pages.find((page) => page.id === urlPageId)
        const hydratedPage = {
          ...remotePage,
          parentId: remotePage.parentId ?? existingPage?.parentId ?? null,
          icon: existingPage?.icon ?? remotePage.icon,
          cover: existingPage?.cover ?? remotePage.cover,
        }

        if (existingPage) {
          pages = pages.map((page) => page.id === urlPageId ? hydratedPage : page)
        } else {
          pages = [...pages, hydratedPage]
        }
        apiUp = true
      }
    }

    set({
      pages,
      currentPageId: urlPageId || null,
      loaded: true,
      apiMode: apiUp,
    })
    await persistLocal(pages)
  },

  refreshPages: async () => {
    const remoteSummaries = await api.fetchPages()
    if (remoteSummaries === null) return

    const currentPages = get().pages
    const currentPageId = get().currentPageId
    const nextPages = mergeRemotePages(remoteSummaries, currentPages)
    const remoteIds = new Set(nextPages.map((page) => page.id))
    const nextCurrentPageId = currentPageId && remoteIds.has(currentPageId)
      ? currentPageId
      : null

    set({
      pages: nextPages,
      currentPageId: nextCurrentPageId,
      apiMode: true,
    })
    await persistLocal(nextPages)

    if (nextCurrentPageId) {
      const remotePage = await api.fetchPage(nextCurrentPageId)
      if (remotePage) {
        await get().hydrateRemotePage(remotePage)
      }
    }
  },

  persistPages: async (pages: WikiPage[]) => {
    await persistLocal(pages)
  },

  hydrateRemotePage: async (page) => {
    const existing = get().pages.find((item) => item.id === page.id)
    const hydratedPage = {
      ...page,
      parentId: page.parentId ?? existing?.parentId ?? null,
      icon: existing?.icon ?? page.icon,
      cover: existing?.cover ?? page.cover,
    }

    const newPages = existing
      ? get().pages.map((item) => item.id === page.id ? hydratedPage : item)
      : [...get().pages, hydratedPage]

    set({ pages: newPages, apiMode: true })
    await persistLocal(newPages)
  },

  createPage: async (title?: string, parentId?: string | null) => {
    const pageTitle = title || 'Untitled'
    const pageIcon = randomIcon()
    const apiMode = get().apiMode

    const remoteResult = await api.createPageResult(pageTitle, parentId, pageIcon)
    const remote = remoteResult.data
    if (remote) {
      remote.parentId = remote.parentId || parentId || null
      remote.icon = remote.icon || pageIcon
      const newPages = [...get().pages, remote]
      set({ pages: newPages, currentPageId: remote.id, apiMode: true })
      await persistLocal(newPages)
      await get().refreshPages()
      return remote
    }

    if (apiMode) {
      pushErrorToast(remoteResult.error?.message || i18n.t('toast.createPageFailed'))
      await get().refreshPages()
      return null
    }

    const page: WikiPage = {
      id: generateId(),
      title: pageTitle,
      parentId: parentId ?? null,
      icon: pageIcon,
      cover: null,
      content: createEmptyDoc(),
      createdAt: Date.now(),
      updatedAt: Date.now(),
      version: 0,
    }
    const newPages = [...get().pages, page]
    set({ pages: newPages, currentPageId: page.id })
    await persistLocal(newPages)
    return page
  },

  updatePage: (id, content, title) => {
    const existing = get().pages.find((p) => p.id === id)
    if (!existing) return
    const newVersion = (existing?.version ?? 0) + 1
    const newTitle = title !== undefined ? title : existing?.title ?? ''

    const newPages = get().pages.map((p) =>
      p.id === id
        ? { ...p, content, title: newTitle, updatedAt: Date.now(), version: newVersion }
        : p
    )
    set({ pages: newPages })
    persistLocal(newPages)

    api.updatePageResult(id, newTitle, content, existing?.version ?? 0).then(async (result) => {
      if (result.data) {
        const nextVersion = result.data.version
        const updated = get().pages.map((p) =>
          p.id === id ? { ...p, version: nextVersion } : p
        )
        set({ pages: updated, apiMode: true })
        persistLocal(updated)
      } else {
        const now = Date.now()
        if (get().apiMode && now - lastSaveErrorAt > 5000) {
          lastSaveErrorAt = now
          pushErrorToast(result.error?.message || i18n.t('toast.savePageFailed'))
        }
        if (get().apiMode) {
          const remotePage = await api.fetchPage(id)
          if (remotePage) {
            await get().hydrateRemotePage(remotePage)
          } else {
            await get().refreshPages()
          }
        }
      }
    })
  },

  updatePageMeta: (id, meta) => {
    const newPages = get().pages.map((p) =>
      p.id === id ? { ...p, ...meta, updatedAt: Date.now() } : p
    )
    set({ pages: newPages })
    persistLocal(newPages)
  },

  deletePage: async (id) => {
    if (get().apiMode) {
      const result = await api.deletePageResult(id)
      await get().refreshPages()
      if (result.error) {
        pushErrorToast(result.error.message || i18n.t('toast.deletePageFailed'))
        return false
      }
      return true
    }

    const newPages = get().pages.filter((p) => p.id !== id)
    set({
      pages: newPages,
      currentPageId: get().currentPageId === id ? null : get().currentPageId,
    })
    persistLocal(newPages)

    void api.deletePage(id)
    return true
  },

  setCurrentPage: async (id) => {
    set({ currentPageId: id })
    if (!id) return

    const remote = await api.fetchPage(id)
    if (remote) {
      await get().hydrateRemotePage(remote)
    }
  },

  getBacklinks: (pageId) => {
    const { pages } = get()
    const backlinks: Backlink[] = []
    for (const page of pages) {
      if (page.id === pageId) continue
      const links = extractPageLinks(page.content)
      if (links.includes(pageId)) {
        backlinks.push({ sourcePageId: page.id, sourcePageTitle: page.title })
      }
    }
    return backlinks
  },

  fetchBacklinks: async (pageId) => {
    const remote = await api.fetchBacklinks(pageId)
    if (remote) return remote
    return get().getBacklinks(pageId)
  },
}))

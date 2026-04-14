import type { JSONContent } from '@tiptap/react'
import type {
  WikiPage,
  MWSTable,
  MWSTableSummary,
  Backlink,
  WikiPageVersionSummary,
  WikiPageVersion,
  WikiComment,
  WikiSearchResult,
  WikiGraph,
} from '../types'

const AUTH_TOKEN_KEY = 'mws-auth-token'

const API_BASE_URL = '/api'

const WS_BASE_URL = `${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}/ws`

function apiFetch(url: string, init?: RequestInit): Promise<Response> {
  const headers = new Headers(init?.headers)
  const token = getAuthToken()
  if (token && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  return fetch(url, {
    ...init,
    headers,
  })
}

async function getErrorMessage(res: Response): Promise<string> {
  try {
    const data = await res.clone().json() as { message?: string; error?: string }
    return data.message || data.error || `HTTP ${res.status}`
  } catch {
    try {
      const text = await res.clone().text()
      return text || `HTTP ${res.status}`
    } catch {
      return `HTTP ${res.status}`
    }
  }
}

async function fetchApiResult<T>(url: string, init?: RequestInit): Promise<ApiResult<T>> {
  try {
    const res = await apiFetch(url, init)
    if (!res.ok) {
      return {
        data: null,
        error: {
          status: res.status,
          message: await getErrorMessage(res),
        },
      }
    }

    return {
      data: await res.json() as T,
      error: null,
    }
  } catch (error) {
    return {
      data: null,
      error: {
        status: null,
        message: error instanceof Error ? error.message : 'Network error',
      },
    }
  }
}

function unavailableResult<T>(): ApiResult<T> {
  return {
    data: null,
    error: { status: null, message: 'Base API is unavailable.' },
  }
}

let _apiAvailable: boolean | null = null

async function isApiAvailable(): Promise<boolean> {
  if (_apiAvailable !== null) return _apiAvailable
  try {
    const res = await apiFetch(`${API_BASE_URL}/pages`, {
      method: 'GET',
      signal: AbortSignal.timeout(2000),
    })
    _apiAvailable = res.ok || res.status === 401 || res.status === 403
  } catch {
    _apiAvailable = false
  }
  setTimeout(() => { _apiAvailable = null }, 30_000)
  return _apiAvailable
}

export interface AuthSession {
  token: string
  userId: string
  username: string
}

export interface AuthUser {
  userId: string
  username: string
}

interface AuthRequest {
  username: string
  password: string
}

function readStoredAuthToken(): string | null {
  try {
    return localStorage.getItem(AUTH_TOKEN_KEY)
  } catch {
    return null
  }
}

export function getAuthToken(): string | null {
  return readStoredAuthToken()
}

export function setAuthToken(token: string) {
  try {
    localStorage.setItem(AUTH_TOKEN_KEY, token)
  } catch {}
}

export function clearAuthToken() {
  try {
    localStorage.removeItem(AUTH_TOKEN_KEY)
  } catch {}
}


interface ApiBlock {
  id: string
  type: 'paragraph' | 'heading' | 'list' | 'table_embed' | 'page_link'
  content: string | null
  props: Record<string, unknown>
}

interface ApiPage {
  id: string
  title: string
  parent_id?: string | null
  icon?: string | null
  content: ApiBlock[]
  version: number
  updatedAt: string
}

export interface ApiPageSummary {
  id: string
  title: string
  parent_id?: string | null
  icon?: string | null
  version: number
  updatedAt: string
}

interface ApiUpdatePageResponse {
  id: string
  version: number
}

interface ApiPageVersion {
  pageId: string
  version: number
  title: string
  content: ApiBlock[]
  savedAt: string
}

interface AiResponse {
  result: string
}

export interface ApiErrorInfo {
  status: number | null
  message: string
}

export interface ApiResult<T> {
  data: T | null
  error: ApiErrorInfo | null
}

export interface CursorPayload {
  anchor: number
  head: number
}

export interface CommentAnchorPayload {
  anchorFrom: number
  anchorTo: number
  anchorText: string
}

export interface PresenceUser {
  userId: string
  name: string
  color: string
}

export interface WsUpdateEvent {
  type: 'update'
  userId: string
  payload: {
    title: string
    content: ApiBlock[]
    version: number
  }
}

export interface WsCursorEvent {
  type: 'cursor'
  userId: string
  cursor: CursorPayload
}

export interface WsPresenceEvent {
  type: 'presence'
  users: PresenceUser[]
}

export interface LegacyJoinEvent {
  type: 'join'
  userId: string
}

export interface LegacyLeaveEvent {
  type: 'leave'
  userId: string
}

export type PageEvent =
  | WsUpdateEvent
  | WsCursorEvent
  | WsPresenceEvent
  | LegacyJoinEvent
  | LegacyLeaveEvent

export interface CollaborationProfile {
  userId: string
  name: string
  color: string
}

const COLLAB_PROFILE_KEY = 'mws-wiki-collaboration-profile'
const COLLAB_ADJECTIVES = ['Swift', 'Quiet', 'Curious', 'Bright', 'Bold', 'Calm', 'Keen', 'Silver']
const COLLAB_NOUNS = ['Fox', 'Otter', 'Falcon', 'Comet', 'Panda', 'Raven', 'Lynx', 'Orca']
const COLLAB_COLORS = ['#0f7b6c', '#2f6fed', '#e57a44', '#b454d4', '#d9485f', '#1f9d55']

function generateClientId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return Math.random().toString(36).slice(2, 10)
}

function hashString(value: string): number {
  let hash = 0
  for (let index = 0; index < value.length; index += 1) {
    hash = ((hash << 5) - hash + value.charCodeAt(index)) | 0
  }
  return Math.abs(hash)
}

export function getDerivedPresenceUser(userId: string): PresenceUser {
  const hash = hashString(userId)
  return {
    userId,
    name: `${COLLAB_ADJECTIVES[hash % COLLAB_ADJECTIVES.length]} ${COLLAB_NOUNS[Math.floor(hash / COLLAB_ADJECTIVES.length) % COLLAB_NOUNS.length]}`,
    color: COLLAB_COLORS[hash % COLLAB_COLORS.length],
  }
}

function loadCollaborationProfile(): CollaborationProfile {
  try {
    const raw = localStorage.getItem(COLLAB_PROFILE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw) as Partial<CollaborationProfile>
      if (parsed.userId) {
        const derived = getDerivedPresenceUser(parsed.userId)
        const profile: CollaborationProfile = {
          userId: parsed.userId,
          name: derived.name,
          color: derived.color,
        }
        try {
          localStorage.setItem(COLLAB_PROFILE_KEY, JSON.stringify(profile))
        } catch {}
        return profile
      }
    }
  } catch {}

  const userId = generateClientId()
  const derived = getDerivedPresenceUser(userId)
  const profile: CollaborationProfile = {
    userId,
    name: derived.name,
    color: derived.color,
  }

  try {
    localStorage.setItem(COLLAB_PROFILE_KEY, JSON.stringify(profile))
  } catch {}

  return profile
}

function generateBlockId(): string {
  return Math.random().toString(36).slice(2, 10)
}

function cloneJsonContent(content: JSONContent[] | undefined): JSONContent[] | undefined {
  if (!content) return undefined
  return JSON.parse(JSON.stringify(content)) as JSONContent[]
}

function getStoredTiptapContent(props: Record<string, unknown>): JSONContent[] | null {
  const value = props.tiptapContent
  return Array.isArray(value) ? value as JSONContent[] : null
}

function mapApiPageSummary(page: ApiPageSummary): ApiPageSummary {
  return {
    ...page,
    parent_id: page.parent_id || null,
    icon: page.icon ?? '📄',
  }
}

function mapApiComment(comment: WikiComment): WikiComment {
  return {
    ...comment,
    anchorFrom: comment.anchorFrom ?? 0,
    anchorTo: comment.anchorTo ?? 0,
    anchorText: comment.anchorText ?? '',
  }
}

interface ApiTableSummary {
  id: string
  name: string
  created_at?: string | null
  updated_at?: string | null
}

interface ApiTableColumn {
  id: string
  table_id?: string
  name: string
  type: 'text' | 'number' | 'date' | 'select'
  position?: number
}

interface ApiTableRow {
  id: string
  table_id?: string
  created_at?: string | null
  values: Record<string, unknown>
}

interface ApiTable {
  id: string
  name: string
  created_at?: string | null
  updated_at?: string | null
  columns: ApiTableColumn[]
  rows: ApiTableRow[]
}

export interface CreateTableColumnInput {
  name: string
  type: 'text' | 'number' | 'date' | 'select'
}

function apiTableSummaryToSummary(table: ApiTableSummary): MWSTableSummary {
  return {
    id: table.id,
    name: table.name,
    createdAt: table.created_at ?? null,
    updatedAt: table.updated_at ?? null,
  }
}

function apiTableToWikiTable(table: ApiTable): MWSTable {
  return {
    id: table.id,
    name: table.name,
    createdAt: table.created_at ?? null,
    updatedAt: table.updated_at ?? null,
    columns: (table.columns ?? []).map((column) => ({
      id: column.id,
      tableId: column.table_id,
      name: column.name,
      type: column.type,
      position: column.position,
    })),
    rows: (table.rows ?? []).map((row) => ({
      id: row.id,
      tableId: row.table_id,
      createdAt: row.created_at ?? null,
      values: row.values ?? {},
    })),
  }
}

function tiptapNodeToApiBlock(node: JSONContent): ApiBlock | null {
  switch (node.type) {
    case 'paragraph':
      return {
        id: generateBlockId(),
        type: 'paragraph',
        content: extractText(node),
        props: {
          tiptapContent: cloneJsonContent(node.content),
        },
      }
    case 'heading':
      return {
        id: generateBlockId(),
        type: 'heading',
        content: extractText(node),
        props: {
          level: node.attrs?.level ?? 1,
          tiptapContent: cloneJsonContent(node.content),
        },
      }
    case 'bulletList':
    case 'orderedList':
      return {
        id: generateBlockId(),
        type: 'list',
        content: extractListText(node),
        props: {
          ordered: node.type === 'orderedList',
          tiptapContent: cloneJsonContent(node.content),
        },
      }
    case 'tableEmbed':
      return {
        id: generateBlockId(),
        type: 'table_embed',
        content: null,
        props: { tableId: node.attrs?.tableId, tableName: node.attrs?.tableName },
      }
    case 'wikiLink':
      return {
        id: generateBlockId(),
        type: 'page_link',
        content: null,
        props: {
          targetId: node.attrs?.pageId,
          targetTitle: node.attrs?.pageTitle,
          pageId: node.attrs?.pageId,
          pageTitle: node.attrs?.pageTitle,
        },
      }
    default:
      return null
  }
}

function extractText(node: JSONContent): string {
  if (typeof node.text === 'string') return node.text
  if (!node.content) return ''
  return node.content.map((child) => extractText(child)).join('')
}

function extractListText(node: JSONContent): string {
  if (!node.content) return ''
  return node.content
    .map((li) => {
      if (!li.content) return ''
      return li.content.map((p) => extractText(p)).join('')
    })
    .join('\n')
}

export function tiptapToApiBlocks(doc: JSONContent): ApiBlock[] {
  if (!doc.content) return []
  const blocks: ApiBlock[] = []
  for (const node of doc.content) {
    const block = tiptapNodeToApiBlock(node)
    if (block) blocks.push(block)
  }
  return blocks
}

function apiBlockToTiptapNode(block: ApiBlock): JSONContent | null {
  const tiptapContent = getStoredTiptapContent(block.props)

  switch (block.type) {
    case 'paragraph':
      return {
        type: 'paragraph',
        content: tiptapContent ?? (block.content ? [{ type: 'text', text: block.content }] : []),
      }
    case 'heading':
      return {
        type: 'heading',
        attrs: { level: (block.props?.level as number) ?? 1 },
        content: tiptapContent ?? (block.content ? [{ type: 'text', text: block.content }] : []),
      }
    case 'list': {
      const ordered = block.props?.ordered === true
      const items = tiptapContent ?? (block.content || '').split('\n').map((text) => ({
        type: 'listItem' as const,
        content: [{ type: 'paragraph' as const, content: text ? [{ type: 'text' as const, text }] : [] }],
      }))
      return {
        type: ordered ? 'orderedList' : 'bulletList',
        content: items.length ? items : [{ type: 'listItem', content: [{ type: 'paragraph' }] }],
      }
    }
    case 'table_embed':
      return {
        type: 'tableEmbed',
        attrs: { tableId: block.props?.tableId, tableName: block.props?.tableName },
      }
    case 'page_link':
      return {
        type: 'wikiLink',
        attrs: {
          pageId: block.props?.targetId ?? block.props?.pageId,
          pageTitle: block.props?.targetTitle ?? block.props?.pageTitle,
        },
      }
    default:
      return null
  }
}

export function apiBlocksToTiptap(blocks: ApiBlock[]): JSONContent {
  const content: JSONContent[] = []
  for (const block of blocks) {
    const node = apiBlockToTiptapNode(block)
    if (node) content.push(node)
  }
  return { type: 'doc', content: content.length ? content : [{ type: 'paragraph' }] }
}

function apiPageToWikiPage(apiPage: ApiPage): WikiPage {
  return {
    id: apiPage.id,
    title: apiPage.title,
    parentId: apiPage.parent_id || null,
    icon: apiPage.icon || '📄',
    cover: null,
    content: apiBlocksToTiptap(apiPage.content || []),
    createdAt: new Date(apiPage.updatedAt).getTime(),
    updatedAt: new Date(apiPage.updatedAt).getTime(),
    version: apiPage.version,
  }
}

function apiVersionToWikiPageVersion(version: ApiPageVersion): WikiPageVersion {
  return {
    pageId: version.pageId,
    version: version.version,
    title: version.title,
    content: apiBlocksToTiptap(version.content || []),
    savedAt: version.savedAt,
  }
}

export const api = {
  async register(username: string, password: string): Promise<ApiResult<AuthSession>> {
    const payload: AuthRequest = { username, password }
    const result = await fetchApiResult<AuthSession>(`${API_BASE_URL}/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })

    if (result.data?.token) {
      setAuthToken(result.data.token)
    }

    return result
  },

  async login(username: string, password: string): Promise<ApiResult<AuthSession>> {
    const payload: AuthRequest = { username, password }
    const result = await fetchApiResult<AuthSession>(`${API_BASE_URL}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })

    if (result.data?.token) {
      setAuthToken(result.data.token)
    }

    return result
  },

  async fetchMe(): Promise<ApiResult<AuthUser>> {
    if (!getAuthToken()) {
      return {
        data: null,
        error: {
          status: 401,
          message: 'Authentication required',
        },
      }
    }

    const result = await fetchApiResult<AuthUser>(`${API_BASE_URL}/auth/me`)
    if (result.error?.status === 401) {
      clearAuthToken()
    }
    return result
  },

  async createPageResult(title: string, parentId?: string | null, icon?: string): Promise<ApiResult<WikiPage>> {
    if (!(await isApiAvailable())) return unavailableResult<WikiPage>()
    const result = await fetchApiResult<ApiPage>(`${API_BASE_URL}/pages`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        title,
        ...(parentId !== undefined ? { parentId } : {}),
        ...(icon !== undefined ? { icon } : {}),
      }),
    })

    return {
      data: result.data ? apiPageToWikiPage(result.data) : null,
      error: result.error,
    }
  },

  async fetchPages(parentId?: string | null): Promise<ApiPageSummary[] | null> {
    if (!(await isApiAvailable())) return null
    try {
      const url = new URL(`${API_BASE_URL}/pages`, window.location.origin)
      if (parentId !== undefined) {
        url.searchParams.set('parentId', parentId ?? 'null')
      }
      const res = await apiFetch(url.pathname + url.search)
      if (!res.ok) return null
      const data: ApiPageSummary[] = await res.json()
      return data.map(mapApiPageSummary)
    } catch {
      return null
    }
  },

  async fetchPagesResult(parentId?: string | null): Promise<ApiResult<ApiPageSummary[]>> {
    if (!(await isApiAvailable())) return unavailableResult<ApiPageSummary[]>()
    const url = new URL(`${API_BASE_URL}/pages`, window.location.origin)
    if (parentId !== undefined) {
      url.searchParams.set('parentId', parentId ?? 'null')
    }
    const result = await fetchApiResult<ApiPageSummary[]>(url.pathname + url.search)
    return {
      data: result.data?.map(mapApiPageSummary) ?? null,
      error: result.error,
    }
  },

  async fetchPage(id: string): Promise<WikiPage | null> {
    if (!(await isApiAvailable())) return null
    try {
      const res = await apiFetch(`${API_BASE_URL}/pages/${id}`)
      if (!res.ok) return null
      const data: ApiPage = await res.json()
      return apiPageToWikiPage(data)
    } catch {
      return null
    }
  },

  async createPage(title: string, parentId?: string | null, icon?: string): Promise<WikiPage | null> {
    const result = await this.createPageResult(title, parentId, icon)
    return result.data
  },

  async updatePageResult(
    id: string,
    title: string,
    content: JSONContent,
    version: number
  ): Promise<ApiResult<ApiUpdatePageResponse>> {
    if (!(await isApiAvailable())) return unavailableResult<ApiUpdatePageResponse>()
    return fetchApiResult<ApiUpdatePageResponse>(`${API_BASE_URL}/pages/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        title,
        content: tiptapToApiBlocks(content),
        version,
      }),
    })
  },

  async updatePage(
    id: string,
    title: string,
    content: JSONContent,
    version: number
  ): Promise<ApiUpdatePageResponse | null> {
    const result = await this.updatePageResult(id, title, content, version)
    return result.data
  },

  async deletePageResult(id: string): Promise<ApiResult<null>> {
    if (!(await isApiAvailable())) return unavailableResult<null>()
    try {
      const res = await apiFetch(`${API_BASE_URL}/pages/${id}`, { method: 'DELETE' })
      if (res.status === 204) {
        return { data: null, error: null }
      }
      return {
        data: null,
        error: {
          status: res.status,
          message: await getErrorMessage(res),
        },
      }
    } catch (error) {
      return {
        data: null,
        error: {
          status: null,
          message: error instanceof Error ? error.message : 'Network error',
        },
      }
    }
  },

  async deletePage(id: string): Promise<boolean> {
    const result = await this.deletePageResult(id)
    return !result.error
  },

  async fetchBacklinks(pageId: string): Promise<Backlink[] | null> {
    if (!(await isApiAvailable())) return null
    try {
      const res = await apiFetch(`${API_BASE_URL}/pages/${pageId}/backlinks`)
      if (!res.ok) return null
      const data: ApiPageSummary[] = await res.json()
      return data.map((p) => ({ sourcePageId: p.id, sourcePageTitle: p.title }))
    } catch {
      return null
    }
  },

  async fetchChildren(pageId: string): Promise<ApiPageSummary[] | null> {
    if (!(await isApiAvailable())) return null
    try {
      const res = await apiFetch(`${API_BASE_URL}/pages/${pageId}/children`)
      if (!res.ok) return null
      const data: ApiPageSummary[] = await res.json()
      return data.map(mapApiPageSummary)
    } catch {
      return null
    }
  },

  async fetchChildrenResult(pageId: string): Promise<ApiResult<ApiPageSummary[]>> {
    if (!(await isApiAvailable())) return unavailableResult<ApiPageSummary[]>()
    const result = await fetchApiResult<ApiPageSummary[]>(`${API_BASE_URL}/pages/${pageId}/children`)
    return {
      data: result.data?.map(mapApiPageSummary) ?? null,
      error: result.error,
    }
  },

  async fetchVersions(pageId: string): Promise<WikiPageVersionSummary[] | null> {
    if (!(await isApiAvailable())) return null
    try {
      const res = await apiFetch(`${API_BASE_URL}/pages/${pageId}/versions`)
      if (!res.ok) return null
      return await res.json()
    } catch {
      return null
    }
  },

  async fetchVersion(pageId: string, version: number): Promise<WikiPageVersion | null> {
    if (!(await isApiAvailable())) return null
    try {
      const res = await apiFetch(`${API_BASE_URL}/pages/${pageId}/versions/${version}`)
      if (!res.ok) return null
      const data: ApiPageVersion = await res.json()
      return apiVersionToWikiPageVersion(data)
    } catch {
      return null
    }
  },

  async restoreVersion(pageId: string, version: number): Promise<ApiUpdatePageResponse | null> {
    if (!(await isApiAvailable())) return null
    try {
      const res = await apiFetch(`${API_BASE_URL}/pages/${pageId}/versions/${version}/restore`, {
        method: 'POST',
      })
      if (!res.ok) return null
      return await res.json()
    } catch {
      return null
    }
  },

  async fetchComments(pageId: string): Promise<WikiComment[] | null> {
    if (!(await isApiAvailable())) return null
    try {
      const res = await apiFetch(`${API_BASE_URL}/pages/${pageId}/comments`)
      if (!res.ok) return null
      const data: WikiComment[] = await res.json()
      return data.map(mapApiComment)
    } catch {
      return null
    }
  },

  async createComment(
    pageId: string,
    author: string,
    text: string,
    anchor?: Partial<CommentAnchorPayload> | null
  ): Promise<WikiComment | null> {
    if (!(await isApiAvailable())) return null
    try {
      const res = await apiFetch(`${API_BASE_URL}/pages/${pageId}/comments`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          author,
          text,
          anchorFrom: anchor?.anchorFrom ?? 0,
          anchorTo: anchor?.anchorTo ?? 0,
          anchorText: anchor?.anchorText ?? '',
        }),
      })
      if (!res.ok) return null
      const data: WikiComment = await res.json()
      return mapApiComment(data)
    } catch {
      return null
    }
  },

  async deleteComment(pageId: string, commentId: string): Promise<boolean> {
    if (!(await isApiAvailable())) return false
    try {
      const url = new URL(`${API_BASE_URL}/pages/${pageId}/comments`, window.location.origin)
      url.searchParams.set('commentId', commentId)
      const res = await apiFetch(url.pathname + url.search, {
        method: 'DELETE',
      })
      return res.status === 204
    } catch {
      return false
    }
  },

  async searchPages(query: string, limit = 20): Promise<WikiSearchResult[] | null> {
    if (!(await isApiAvailable())) return null
    try {
      const res = await apiFetch(`${API_BASE_URL}/pages/search`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ text: query }),
      })
      if (!res.ok) return null
      return await res.json()
    } catch {
      return null
    }
  },

  async fetchGraph(): Promise<WikiGraph | null> {
    if (!(await isApiAvailable())) return null
    try {
      const res = await apiFetch(`${API_BASE_URL}/pages/graph`)
      if (!res.ok) return null
      return await res.json()
    } catch {
      return null
    }
  },

  async completeText(prompt: string, context?: string | null): Promise<string | null> {
    if (!(await isApiAvailable())) return null
    try {
      const res = await apiFetch(`${API_BASE_URL}/ai/complete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          prompt,
          ...(context !== undefined ? { context } : {}),
        }),
      })
      if (!res.ok) return null
      const data: AiResponse = await res.json()
      return data.result
    } catch {
      return null
    }
  },

  async summarizePage(pageId: string): Promise<string | null> {
    if (!(await isApiAvailable())) return null
    try {
      const res = await apiFetch(`${API_BASE_URL}/ai/summarize`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ pageId }),
      })
      if (!res.ok) return null
      const data: AiResponse = await res.json()
      return data.result
    } catch {
      return null
    }
  },

  async suggestNextBlock(context: string): Promise<string | null> {
    if (!(await isApiAvailable())) return null
    try {
      const res = await apiFetch(`${API_BASE_URL}/ai/suggest`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ context }),
      })
      if (!res.ok) return null
      const data: AiResponse = await res.json()
      return data.result
    } catch {
      return null
    }
  },

  async fetchTables(): Promise<MWSTableSummary[] | null> {
    if (!(await isApiAvailable())) return null
    try {
      const res = await apiFetch(`${API_BASE_URL}/tables`)
      if (!res.ok) return null
      const data: ApiTableSummary[] = await res.json()
      return data.map(apiTableSummaryToSummary)
    } catch {
      return null
    }
  },

  async fetchTablesResult(): Promise<ApiResult<MWSTableSummary[]>> {
    if (!(await isApiAvailable())) return unavailableResult<MWSTableSummary[]>()
    const result = await fetchApiResult<ApiTableSummary[]>(`${API_BASE_URL}/tables`)
    return {
      data: result.data?.map(apiTableSummaryToSummary) ?? null,
      error: result.error,
    }
  },

  async fetchTable(id: string): Promise<MWSTable | null> {
    if (!(await isApiAvailable())) return null
    try {
      const res = await apiFetch(`${API_BASE_URL}/tables/${id}`)
      if (!res.ok) return null
      const data: ApiTable = await res.json()
      return apiTableToWikiTable(data)
    } catch {
      return null
    }
  },

  async fetchTableResult(id: string): Promise<ApiResult<MWSTable>> {
    if (!(await isApiAvailable())) return unavailableResult<MWSTable>()
    const result = await fetchApiResult<ApiTable>(`${API_BASE_URL}/tables/${id}`)
    return {
      data: result.data ? apiTableToWikiTable(result.data) : null,
      error: result.error,
    }
  },

  async createTable(name: string, columns: CreateTableColumnInput[] = []): Promise<MWSTable | null> {
    if (!(await isApiAvailable())) return null
    try {
      const res = await apiFetch(`${API_BASE_URL}/tables`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, columns }),
      })
      if (!res.ok) return null
      const data: ApiTable = await res.json()
      return apiTableToWikiTable(data)
    } catch {
      return null
    }
  },

  async updateTable(id: string, name: string): Promise<MWSTable | null> {
    if (!(await isApiAvailable())) return null
    try {
      const res = await apiFetch(`${API_BASE_URL}/tables/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name }),
      })
      if (!res.ok) return null
      const data: ApiTable = await res.json()
      return apiTableToWikiTable(data)
    } catch {
      return null
    }
  },

  async deleteTable(id: string): Promise<boolean> {
    if (!(await isApiAvailable())) return false
    try {
      const res = await apiFetch(`${API_BASE_URL}/tables/${id}`, { method: 'DELETE' })
      return res.status === 204
    } catch {
      return false
    }
  },

  async addTableColumn(tableId: string, column: CreateTableColumnInput): Promise<MWSTable['columns'][number] | null> {
    if (!(await isApiAvailable())) return null
    try {
      const res = await apiFetch(`${API_BASE_URL}/tables/${tableId}/columns`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(column),
      })
      if (!res.ok) return null
      const data: ApiTableColumn = await res.json()
      return {
        id: data.id,
        tableId: data.table_id,
        name: data.name,
        type: data.type,
        position: data.position,
      }
    } catch {
      return null
    }
  },

  async deleteTableColumn(tableId: string, columnId: string): Promise<boolean> {
    if (!(await isApiAvailable())) return false
    try {
      const res = await apiFetch(`${API_BASE_URL}/tables/${tableId}/columns/${columnId}`, {
        method: 'DELETE',
      })
      return res.status === 204
    } catch {
      return false
    }
  },

  async addTableRow(tableId: string): Promise<MWSTable['rows'][number] | null> {
    if (!(await isApiAvailable())) return null
    try {
      const res = await apiFetch(`${API_BASE_URL}/tables/${tableId}/rows`, {
        method: 'POST',
      })
      if (!res.ok) return null
      const data: ApiTableRow = await res.json()
      return {
        id: data.id,
        tableId: data.table_id,
        createdAt: data.created_at ?? null,
        values: data.values ?? {},
      }
    } catch {
      return null
    }
  },

  async updateTableRow(tableId: string, rowId: string, values: Record<string, string>): Promise<MWSTable['rows'][number] | null> {
    if (!(await isApiAvailable())) return null
    try {
      const res = await apiFetch(`${API_BASE_URL}/tables/${tableId}/rows/${rowId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ values }),
      })
      if (!res.ok) return null
      const data: ApiTableRow = await res.json()
      return {
        id: data.id,
        tableId: data.table_id,
        createdAt: data.created_at ?? null,
        values: data.values ?? {},
      }
    } catch {
      return null
    }
  },

  async deleteTableRow(tableId: string, rowId: string): Promise<boolean> {
    if (!(await isApiAvailable())) return false
    try {
      const res = await apiFetch(`${API_BASE_URL}/tables/${tableId}/rows/${rowId}`, {
        method: 'DELETE',
      })
      return res.status === 204
    } catch {
      return false
    }
  },
}


export type WsEventHandler = (event: PageEvent) => void
export type ConnectionStatusHandler = (connected: boolean) => void
type WsOutgoingEvent = WsUpdateEvent | WsCursorEvent

export class CollaborationSocket {
  private ws: WebSocket | null = null
  private pageId: string
  private readonly profile = loadCollaborationProfile()
  private handlers: Set<WsEventHandler> = new Set()
  private statusHandlers: Set<ConnectionStatusHandler> = new Set()
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private _connected = false
  private _disposed = false
  private _retryCount = 0
  private closeWhenOpen = false

  constructor(pageId: string) {
    this.pageId = pageId
  }

  get connected() { return this._connected }
  get userId() { return this.profile.userId }
  get name() { return this.profile.name }
  get color() { return this.profile.color }

  private setConnected(next: boolean) {
    if (this._connected === next) return
    this._connected = next
    this.statusHandlers.forEach((handler) => handler(next))
  }

  async connect() {
    if (this.ws || this._disposed) return

    try {
      const url = new URL(`${WS_BASE_URL}/pages/${this.pageId}`)
      url.searchParams.set('userId', this.profile.userId)
      url.searchParams.set('name', this.profile.name)
      url.searchParams.set('color', this.profile.color)
      const ws = new WebSocket(url.toString())
      this.ws = ws
      this.closeWhenOpen = false

      ws.onopen = () => {
        if (this._disposed || this.closeWhenOpen) {
          ws.close()
          return
        }
        this.setConnected(true)
        this._retryCount = 0
      }

      ws.onmessage = (e) => {
        try {
          const event: PageEvent = JSON.parse(e.data)
          this.handlers.forEach((h) => h(event))
        } catch {}
      }

      ws.onclose = () => {
        this.setConnected(false)
        if (this.ws === ws) {
          this.ws = null
        }
        if (!this._disposed) {
          const delay = Math.min(3000 * Math.pow(2, this._retryCount), 30000)
          this._retryCount++
          this.reconnectTimer = setTimeout(() => this.connect(), delay)
        }
      }

      ws.onerror = () => {
        this.setConnected(false)
        ws.close()
      }
    } catch {
      this.setConnected(false)
    }
  }

  send(event: WsOutgoingEvent) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return
    this.ws.send(JSON.stringify(event))
  }

  sendUpdate(title: string, content: JSONContent, version: number) {
    this.send({
      type: 'update',
      userId: this.profile.userId,
      payload: { title, content: tiptapToApiBlocks(content), version },
    })
  }

  sendCursor(anchor: number, head: number) {
    this.send({
      type: 'cursor',
      userId: this.profile.userId,
      cursor: { anchor, head },
    })
  }

  on(handler: WsEventHandler) {
    this.handlers.add(handler)
    return () => { this.handlers.delete(handler) }
  }

  onStatusChange(handler: ConnectionStatusHandler) {
    this.statusHandlers.add(handler)
    return () => { this.statusHandlers.delete(handler) }
  }

  disconnect() {
    this._disposed = true
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer)
    if (this.ws?.readyState === WebSocket.CONNECTING) {
      this.closeWhenOpen = true
    } else {
      this.ws?.close()
      this.ws = null
    }
    this.setConnected(false)
    this.handlers.clear()
    this.statusHandlers.clear()
  }
}

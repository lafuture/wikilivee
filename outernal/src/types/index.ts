import type { JSONContent } from '@tiptap/react'

export interface WikiPage {
  id: string
  title: string
  parentId?: string | null
  icon: string
  cover: string | null
  content: JSONContent
  createdAt: number
  updatedAt: number
  version?: number
}

export interface MWSTable {
  id: string
  name: string
  createdAt: string | null
  updatedAt: string | null
  columns: MWSTableColumn[]
  rows: MWSTableRow[]
}

export interface MWSTableColumn {
  id: string
  tableId?: string
  name: string
  type: 'text' | 'number' | 'date' | 'select'
  position?: number
}

export interface MWSTableRow {
  id: string
  tableId?: string
  createdAt?: string | null
  values: Record<string, unknown>
}

export interface MWSTableSummary {
  id: string
  name: string
  createdAt: string | null
  updatedAt: string | null
}

export interface Backlink {
  sourcePageId: string
  sourcePageTitle: string
}

export interface WikiPageVersionSummary {
  version: number
  savedAt: string
}

export interface WikiPageVersion {
  pageId: string
  version: number
  title: string
  content: JSONContent
  savedAt: string
}

export interface WikiComment {
  id: string
  pageId: string
  author: string
  text: string
  anchorFrom: number
  anchorTo: number
  anchorText: string
  createdAt: string
}

export interface WikiSearchResult {
  pageId: string
  title: string
  snippet: string
  updatedAt: string
}

export interface WikiGraphNode {
  id: string
  title: string
}

export interface WikiGraphEdge {
  source: string
  target: string
}

export interface WikiGraph {
  nodes: WikiGraphNode[]
  edges: WikiGraphEdge[]
}

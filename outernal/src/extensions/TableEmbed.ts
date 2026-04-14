import { Node } from '@tiptap/react'
import type { MWSTable } from '../types'
import { api } from '../utils/api'

function escapeHtml(value: unknown): string {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')
}

function renderTablePreview(container: HTMLElement, table: MWSTable) {
  const header = table.columns
    .map((column) => `<th>${escapeHtml(column.name)}</th>`)
    .join('')

  const rows = table.rows.slice(0, 5)
    .map((row) => {
      const cells = table.columns
        .map((column) => `<td>${escapeHtml(row.values?.[column.id])}</td>`)
        .join('')
      return `<tr>${cells}</tr>`
    })
    .join('')

  container.innerHTML = `
    <div class="table-embed-preview">
      <table class="table-embed-table">
        <thead>
          <tr>${header}</tr>
        </thead>
        <tbody>
          ${rows || `<tr><td colspan="${Math.max(table.columns.length, 1)}">No rows</td></tr>`}
        </tbody>
      </table>
    </div>
  `
}

function renderTableError(container: HTMLElement, message: string) {
  container.innerHTML = `<div class="table-embed-placeholder">${escapeHtml(message)}</div>`
}

export const TableEmbed = Node.create({
  name: 'tableEmbed',
  group: 'block',
  atom: true,

  addAttributes() {
    return {
      tableId: { default: null },
      tableName: { default: '' },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-table-embed]' }]
  },

  renderHTML({ HTMLAttributes }) {
    return ['div', { 'data-table-embed': '', 'data-table-id': HTMLAttributes.tableId, class: 'table-embed-wrapper' }, `Table: ${HTMLAttributes.tableName}`]
  },

  addNodeView() {
    return ({ node }) => {
      const dom = document.createElement('div')
      dom.classList.add('table-embed-wrapper')
      dom.setAttribute('data-table-embed', '')
      dom.contentEditable = 'false'
      dom.innerHTML = `
        <div class="table-embed-header">
          <span class="table-embed-icon">📊</span>
          <span class="table-embed-name">${node.attrs.tableName || 'MWS Table'}</span>
          <span class="table-embed-id">#${node.attrs.tableId || ''}</span>
        </div>
        <div class="table-embed-body" id="table-${node.attrs.tableId}">
          <div class="table-embed-placeholder">Loading table data from MWS Tables API...</div>
        </div>
      `
      const container = dom.querySelector('.table-embed-body')
      if (container instanceof HTMLElement) {
        const tableId = node.attrs.tableId
        if (!tableId) {
          renderTableError(container, 'Table ID is missing.')
        } else {
          api.fetchTableResult(tableId).then((result) => {
            if (result.data) {
              renderTablePreview(container, result.data)
              return
            }

            if (result.error?.status === 404) {
              renderTableError(container, 'Tables endpoint is not available on the backend.')
              return
            }

            if (result.error?.status === 502) {
              renderTableError(container, 'MWS Tables API is unavailable.')
              return
            }

            renderTableError(container, result.error?.message || 'Failed to load table.')
          })
        }
      }
      return { dom }
    }
  },
})

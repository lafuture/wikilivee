import { Node, mergeAttributes } from '@tiptap/react'

export const WikiLink = Node.create({
  name: 'wikiLink',
  group: 'block',
  atom: true,

  addAttributes() {
    return {
      pageId: { default: null },
      pageTitle: { default: '' },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-wiki-link]' }]
  },

  renderHTML({ HTMLAttributes }) {
    return [
      'div',
      mergeAttributes(HTMLAttributes, {
        'data-wiki-link': '',
        'data-page-id': HTMLAttributes.pageId,
        'data-page-title': HTMLAttributes.pageTitle,
        class: 'wiki-link',
      }),
      `[[${HTMLAttributes['data-page-title'] || HTMLAttributes.pageTitle || ''}]]`,
    ]
  },

  addNodeView() {
    return ({ node }) => {
      const dom = document.createElement('div')
      dom.classList.add('wiki-link')
      dom.setAttribute('data-wiki-link', '')
      dom.setAttribute('data-page-id', node.attrs.pageId)
      dom.setAttribute('data-page-title', node.attrs.pageTitle)
      dom.textContent = `[[${node.attrs.pageTitle}]]`
      dom.contentEditable = 'false'
      dom.addEventListener('click', () => {
        window.dispatchEvent(
          new CustomEvent('wiki-navigate', { detail: { pageId: node.attrs.pageId } })
        )
      })
      return { dom }
    }
  },
})

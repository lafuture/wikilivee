import { BubbleMenu } from '@tiptap/react/menus'
import type { Editor } from '@tiptap/react'
import './BubbleToolbar.css'

interface BubbleToolbarProps {
  editor: Editor
}

export function BubbleToolbar({ editor }: BubbleToolbarProps) {
  return (
    <BubbleMenu
      editor={editor}
      className="bubble-toolbar"
    >
      <button
        className={`bubble-btn ${editor.isActive('bold') ? 'is-active' : ''}`}
        onClick={() => editor.chain().focus().toggleBold().run()}
        title="Bold"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M6 4h8a4 4 0 0 1 2.83 6.83A4 4 0 0 1 15 20H6V4zm3 7h5a1.5 1.5 0 0 0 0-3H9v3zm0 3v3h6a1.5 1.5 0 0 0 0-3H9z"/></svg>
      </button>
      <button
        className={`bubble-btn ${editor.isActive('italic') ? 'is-active' : ''}`}
        onClick={() => editor.chain().focus().toggleItalic().run()}
        title="Italic"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M10 4h8v2h-2.21l-3.42 12H15v2H7v-2h2.21l3.42-12H10V4z"/></svg>
      </button>
      <button
        className={`bubble-btn ${editor.isActive('strike') ? 'is-active' : ''}`}
        onClick={() => editor.chain().focus().toggleStrike().run()}
        title="Strikethrough"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M3 12h18v2H3v-2zm3-7h12v2H6V5zm4 14h4v-2h-4v2z"/></svg>
      </button>
      <button
        className={`bubble-btn ${editor.isActive('code') ? 'is-active' : ''}`}
        onClick={() => editor.chain().focus().toggleCode().run()}
        title="Code"
      >
        <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M8.293 6.293 2.586 12l5.707 5.707 1.414-1.414L5.414 12l4.293-4.293-1.414-1.414zm7.414 0-1.414 1.414L18.586 12l-4.293 4.293 1.414 1.414L21.414 12l-5.707-5.707z"/></svg>
      </button>
      <div className="bubble-sep" />
      <button
        className={`bubble-btn ${editor.isActive('heading', { level: 1 }) ? 'is-active' : ''}`}
        onClick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()}
        title="Heading 1"
      >
        <span className="bubble-text">H1</span>
      </button>
      <button
        className={`bubble-btn ${editor.isActive('heading', { level: 2 }) ? 'is-active' : ''}`}
        onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
        title="Heading 2"
      >
        <span className="bubble-text">H2</span>
      </button>
    </BubbleMenu>
  )
}

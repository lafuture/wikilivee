import { Extension } from '@tiptap/react'
import { Node as ProseMirrorNode } from '@tiptap/pm/model'
import { Plugin, PluginKey } from '@tiptap/pm/state'
import { Decoration, DecorationSet } from '@tiptap/pm/view'
import type { CursorPayload, PresenceUser } from '../utils/api'

export interface RemotePresenceCursor extends PresenceUser, CursorPayload {}

type RemotePresenceState = Record<string, RemotePresenceCursor>

export const remotePresencePluginKey = new PluginKey<RemotePresenceState>('remotePresence')

function clampPosition(position: number, doc: ProseMirrorNode): number {
  return Math.max(0, Math.min(position, doc.content.size))
}

function createCaret(cursor: RemotePresenceCursor): HTMLElement {
  const wrapper = document.createElement('span')
  wrapper.className = 'remote-presence-caret'
  wrapper.style.setProperty('--user-color', cursor.color)

  const line = document.createElement('span')
  line.className = 'remote-presence-caret-line'

  const label = document.createElement('span')
  label.className = 'remote-presence-label'
  label.textContent = cursor.name

  wrapper.append(line, label)
  return wrapper
}

function createDecorations(doc: ProseMirrorNode, cursors: RemotePresenceCursor[]): DecorationSet {
  const decorations: Decoration[] = []

  for (const cursor of cursors) {
    const anchor = clampPosition(cursor.anchor, doc)
    const head = clampPosition(cursor.head, doc)
    const from = Math.min(anchor, head)
    const to = Math.max(anchor, head)

    if (from !== to) {
      decorations.push(
        Decoration.inline(from, to, {
          class: 'remote-presence-selection',
          style: `--user-color: ${cursor.color};`,
        })
      )
    }

    decorations.push(
      Decoration.widget(head, () => createCaret(cursor), {
        key: `${cursor.userId}:${anchor}:${head}`,
        side: 1,
      })
    )
  }

  return DecorationSet.create(doc, decorations)
}

export const RemotePresence = Extension.create({
  name: 'remotePresence',

  addProseMirrorPlugins() {
    return [
      new Plugin<RemotePresenceState>({
        key: remotePresencePluginKey,
        state: {
          init: () => ({}),
          apply(tr, value) {
            let next = value

            if (tr.docChanged) {
              next = Object.fromEntries(
                Object.values(next).map((cursor) => [
                  cursor.userId,
                  {
                    ...cursor,
                    anchor: tr.mapping.map(cursor.anchor),
                    head: tr.mapping.map(cursor.head),
                  },
                ])
              )
            }

            const meta = tr.getMeta(remotePresencePluginKey) as { cursors?: RemotePresenceCursor[] } | undefined
            if (meta?.cursors) {
              return Object.fromEntries(meta.cursors.map((cursor) => [cursor.userId, cursor]))
            }

            return next
          },
        },
        props: {
          decorations(state) {
            const cursors = remotePresencePluginKey.getState(state)
            return createDecorations(state.doc, Object.values(cursors ?? {}))
          },
        },
      }),
    ]
  },
})

import { ReactRenderer } from '@tiptap/react'
import tippy, { type Instance } from 'tippy.js'
import { SlashMenu } from '../components/SlashMenu/SlashMenu'

export function slashSuggestion() {
  return {
    render: () => {
      let component: ReactRenderer | null = null
      let popup: Instance[] | null = null

      return {
        onStart: (props: any) => {
          component = new ReactRenderer(SlashMenu, {
            props,
            editor: props.editor,
          })

          if (!props.clientRect) return

          popup = tippy('body', {
            getReferenceClientRect: props.clientRect,
            appendTo: () => document.body,
            content: component.element,
            showOnCreate: true,
            interactive: true,
            trigger: 'manual',
            placement: 'bottom-start',
          })
        },

        onUpdate(props: any) {
          component?.updateProps(props)
          if (popup && props.clientRect) {
            popup[0].setProps({
              getReferenceClientRect: props.clientRect,
            })
          }
        },

        onKeyDown(props: any) {
          if (props.event.key === 'Escape') {
            popup?.[0]?.hide()
            return true
          }
          return (component?.ref as any)?.onKeyDown(props)
        },

        onExit() {
          popup?.[0]?.destroy()
          component?.destroy()
        },
      }
    },
  }
}

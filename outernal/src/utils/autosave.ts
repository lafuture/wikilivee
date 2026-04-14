import type { JSONContent } from '@tiptap/react'

type SaveFn = (content: JSONContent) => void

export function createAutosave(saveFn: SaveFn, delay = 500) {
  let timer: ReturnType<typeof setTimeout> | null = null

  const save = (content: JSONContent) => {
    if (timer) clearTimeout(timer)
    timer = setTimeout(() => {
      saveFn(content)
      timer = null
    }, delay)
  }

  const flush = (content: JSONContent) => {
    if (timer) clearTimeout(timer)
    saveFn(content)
  }

  const cancel = () => {
    if (timer) clearTimeout(timer)
  }

  return { save, flush, cancel }
}

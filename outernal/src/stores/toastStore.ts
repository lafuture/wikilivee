import { create } from 'zustand'

export type ToastTone = 'info' | 'success' | 'error'

export interface ToastItem {
  id: string
  message: string
  tone: ToastTone
}

interface ToastStore {
  toasts: ToastItem[]
  pushToast: (message: string, tone?: ToastTone, duration?: number) => void
  removeToast: (id: string) => void
}

function createToastId() {
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`
}

export const useToastStore = create<ToastStore>((set) => ({
  toasts: [],

  pushToast: (message, tone = 'info', duration = 4000) => {
    const id = createToastId()
    set((state) => ({
      toasts: [...state.toasts, { id, message, tone }],
    }))

    window.setTimeout(() => {
      set((state) => ({
        toasts: state.toasts.filter((toast) => toast.id !== id),
      }))
    }, duration)
  },

  removeToast: (id) => {
    set((state) => ({
      toasts: state.toasts.filter((toast) => toast.id !== id),
    }))
  },
}))

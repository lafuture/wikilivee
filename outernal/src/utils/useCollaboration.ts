import { useEffect, useRef, useState, useCallback } from 'react'
import {
  CollaborationSocket,
  getDerivedPresenceUser,
  type PageEvent,
  type PresenceUser,
} from './api'
import type { JSONContent } from '@tiptap/react'

export interface RemoteSelection extends PresenceUser {
  anchor: number
  head: number
  updatedAt: number
}

interface UseCollaborationOptions {
  pageId: string
  onRemoteUpdate?: (event: Extract<PageEvent, { type: 'update' }>) => void
}

export function useCollaboration({ pageId, onRemoteUpdate }: UseCollaborationOptions) {
  const socketRef = useRef<CollaborationSocket | null>(null)
  const onRemoteUpdateRef = useRef(onRemoteUpdate)
  const peersRef = useRef<PresenceUser[]>([])
  const cleanupTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const [self, setSelf] = useState<PresenceUser | null>(null)
  const [connected, setConnected] = useState(false)
  const [peers, setPeers] = useState<PresenceUser[]>([])
  const [remoteSelections, setRemoteSelections] = useState<RemoteSelection[]>([])

  useEffect(() => {
    onRemoteUpdateRef.current = onRemoteUpdate
  }, [onRemoteUpdate])

  useEffect(() => {
    peersRef.current = peers
  }, [peers])

  useEffect(() => {
    const socket = new CollaborationSocket(pageId)
    socketRef.current = socket
    const selfUserId = socket.userId
    setSelf({ userId: socket.userId, name: socket.name, color: socket.color })

    const unsubEvents = socket.on((event) => {
      switch (event.type) {
        case 'presence':
          setPeers(event.users.filter((user) => user.userId !== selfUserId))
          setRemoteSelections((current) =>
            current.filter((cursor) => event.users.some((user) => user.userId === cursor.userId))
          )
          break
        case 'cursor':
          if (event.userId === selfUserId) return
          setRemoteSelections((current) => {
            const peer = peersRef.current.find((user) => user.userId === event.userId)
            const existing = current.find((cursor) => cursor.userId === event.userId)
            const fallback = existing ?? peer ?? getDerivedPresenceUser(event.userId)

            const nextCursor: RemoteSelection = {
              userId: fallback.userId,
              name: fallback.name,
              color: fallback.color,
              anchor: event.cursor.anchor,
              head: event.cursor.head,
              updatedAt: Date.now(),
            }

            if (existing) {
              return current.map((cursor) => cursor.userId === event.userId ? nextCursor : cursor)
            }
            return [...current, nextCursor]
          })
          setPeers((current) => {
            if (current.some((user) => user.userId === event.userId)) return current
            return [...current, getDerivedPresenceUser(event.userId)]
          })
          break
        case 'update':
          if (event.userId !== selfUserId && onRemoteUpdateRef.current) {
            onRemoteUpdateRef.current(event)
          }
          break
        case 'join':
          if (event.userId === selfUserId) return
          setPeers((current) => {
            if (current.some((user) => user.userId === event.userId)) return current
            return [...current, getDerivedPresenceUser(event.userId)]
          })
          break
        case 'leave':
          setPeers((current) => current.filter((user) => user.userId !== event.userId))
          setRemoteSelections((current) => current.filter((cursor) => cursor.userId !== event.userId))
          break
      }
    })

    const unsubStatus = socket.onStatusChange(setConnected)
    socket.connect()
    cleanupTimerRef.current = setInterval(() => {
      const cutoff = Date.now() - 15_000
      setRemoteSelections((current) => current.filter((cursor) => cursor.updatedAt >= cutoff))
    }, 5_000)

    return () => {
      if (cleanupTimerRef.current) clearInterval(cleanupTimerRef.current)
      unsubEvents()
      unsubStatus()
      socket.disconnect()
      socketRef.current = null
      setSelf(null)
      setConnected(false)
      setPeers([])
      setRemoteSelections([])
    }
  }, [pageId])

  useEffect(() => {
    setRemoteSelections((current) =>
      current.map((cursor) => {
        const peer = peers.find((user) => user.userId === cursor.userId)
        return peer ? { ...cursor, name: peer.name, color: peer.color } : cursor
      })
    )
  }, [peers])

  const sendUpdate = useCallback(
    (title: string, content: JSONContent, version: number) => {
      socketRef.current?.sendUpdate(title, content, version)
    },
    []
  )

  const sendCursor = useCallback((anchor: number, head: number) => {
    socketRef.current?.sendCursor(anchor, head)
  }, [])

  return { connected, self, peers, remoteSelections, sendUpdate, sendCursor }
}

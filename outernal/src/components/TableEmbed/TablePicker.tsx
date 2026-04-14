import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { api } from '../../utils/api'
import './TablePicker.css'

interface TablePickerProps {
  onSelect: (tableId: string, tableName: string) => void
  onClose: () => void
}

const MOCK_TABLES = [
  { id: 'tbl_001', name: 'Sprint Tasks' },
  { id: 'tbl_002', name: 'Team Members' },
  { id: 'tbl_003', name: 'Bug Tracker' },
  { id: 'tbl_004', name: 'Product Roadmap' },
]

export function TablePicker({ onSelect, onClose }: TablePickerProps) {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const [tables, setTables] = useState<{ id: string; name: string }[]>(MOCK_TABLES)
  const [loading, setLoading] = useState(true)
  const [isOffline, setIsOffline] = useState(false)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    api.fetchTablesResult().then((result) => {
      if (cancelled) return
      if (result.data) {
        setTables(result.data)
      } else {
        const useMocks = import.meta.env.DEV
        setTables(useMocks ? MOCK_TABLES : [])
        setIsOffline(true)
        if (result.error?.status === 404) {
          setErrorMessage(t('tablePicker.endpointMissing'))
        } else if (result.error?.status === 502) {
          setErrorMessage(t('tablePicker.unavailable'))
        } else if (result.error?.message) {
          setErrorMessage(`${t('tablePicker.errorPrefix')} ${result.error.message}`)
        } else {
          setErrorMessage(t('tablePicker.unavailable'))
        }
      }
      setLoading(false)
    })
    return () => { cancelled = true }
  }, [t])

  const filtered = tables.filter((tbl) =>
    tbl.name.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <div className="table-picker-overlay" onClick={onClose}>
      <div className="table-picker" onClick={(e) => e.stopPropagation()}>
        <div className="table-picker-header">
          <h3>{t('tablePicker.title')}</h3>
          <button className="table-picker-close" onClick={onClose}>
            &times;
          </button>
        </div>
        {isOffline && (
          <div className="table-picker-offline">
            {import.meta.env.DEV ? t('tablePicker.offline') : errorMessage}
          </div>
        )}
        <input
          className="table-picker-search"
          type="text"
          placeholder={t('tablePicker.search')}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          autoFocus
        />
        <div className="table-picker-list">
          {loading ? (
            <div className="table-picker-empty">{t('tablePicker.loading')}</div>
          ) : filtered.length ? (
            filtered.map((table) => (
              <button
                key={table.id}
                className="table-picker-item"
                onClick={() => onSelect(table.id, table.name)}
              >
                <span className="table-picker-icon">📊</span>
                <span>{table.name}</span>
                <span className="table-picker-id">{table.id}</span>
              </button>
            ))
          ) : (
            <div className="table-picker-empty">{t('tablePicker.empty')}</div>
          )}
        </div>
      </div>
    </div>
  )
}

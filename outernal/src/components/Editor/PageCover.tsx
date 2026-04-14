import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import './PageCover.css'

const COVERS = [
  'linear-gradient(135deg, #e03e3e 0%, #ff6b6b 100%)',
  'linear-gradient(135deg, #2f80ed 0%, #56ccf2 100%)',
  'linear-gradient(135deg, #0f7b6c 0%, #56d6c1 100%)',
  'linear-gradient(135deg, #9b51e0 0%, #d896ff 100%)',
  'linear-gradient(135deg, #f2994a 0%, #f7c948 100%)',
  'linear-gradient(135deg, #37352f 0%, #6b6b6b 100%)',
  'linear-gradient(135deg, #e44d90 0%, #c850c0 50%, #4158d0 100%)',
  'linear-gradient(135deg, #08aeea 0%, #2af598 100%)',
]

interface PageCoverProps {
  cover: string | null
  onSelect: (cover: string | null) => void
}

export function PageCover({ cover, onSelect }: PageCoverProps) {
  const { t } = useTranslation()
  const [showPicker, setShowPicker] = useState(false)

  return (
    <>
      {cover ? (
        <div className="page-cover" style={{ background: cover }}>
          <div className="page-cover-actions">
            <button className="cover-action-btn" onClick={() => setShowPicker(true)}>
              {t('editor.changeCover', 'Change cover')}
            </button>
            <button className="cover-action-btn cover-action-remove" onClick={() => onSelect(null)}>
              {t('editor.removeCover', 'Remove')}
            </button>
          </div>
        </div>
      ) : (
        <div className="page-cover-placeholder">
          <button className="cover-add-btn" onClick={() => setShowPicker(true)}>
            {t('editor.addCover', 'Add cover')}
          </button>
        </div>
      )}

      {showPicker && (
        <div className="cover-picker-overlay" onClick={() => setShowPicker(false)}>
          <div className="cover-picker" onClick={(e) => e.stopPropagation()}>
            <div className="cover-picker-grid">
              {COVERS.map((c) => (
                <button
                  key={c}
                  className={`cover-picker-item ${c === cover ? 'is-active' : ''}`}
                  style={{ background: c }}
                  onClick={() => { onSelect(c); setShowPicker(false) }}
                />
              ))}
            </div>
          </div>
        </div>
      )}
    </>
  )
}

import { useState } from 'react'
import './IconPicker.css'

const EMOJI_GROUPS = [
  ['📄', '📝', '📋', '📒', '📓', '📔', '📕', '📗', '📘', '📙'],
  ['💡', '🎯', '🚀', '⭐', '🔖', '🏷️', '💎', '🔮', '🎨', '🎭'],
  ['🏠', '🌍', '🌈', '☀️', '🌙', '⚡', '🔥', '💧', '🌿', '🍀'],
  ['❤️', '🧡', '💛', '💚', '💙', '💜', '🖤', '🤍', '🩷', '🩵'],
  ['📊', '📈', '📉', '🗂️', '📁', '📂', '🗃️', '📦', '🗄️', '🏗️'],
  ['🎵', '🎶', '🎸', '🎹', '🥁', '🎤', '🎧', '📸', '🎬', '🎮'],
  ['🔧', '🔨', '⚙️', '🛠️', '🔬', '🔭', '💻', '🖥️', '📱', '⌨️'],
]

interface IconPickerProps {
  current: string
  onSelect: (icon: string) => void
  onClose: () => void
}

export function IconPicker({ current, onSelect, onClose }: IconPickerProps) {
  const [hoveredIcon, setHoveredIcon] = useState<string | null>(null)

  return (
    <div className="icon-picker-overlay" onClick={onClose}>
      <div className="icon-picker" onClick={(e) => e.stopPropagation()}>
        <div className="icon-picker-preview">
          <span className="icon-picker-current">{hoveredIcon || current}</span>
        </div>
        <div className="icon-picker-grid">
          {EMOJI_GROUPS.flat().map((emoji) => (
            <button
              key={emoji}
              className={`icon-picker-item ${emoji === current ? 'is-current' : ''}`}
              onClick={() => { onSelect(emoji); onClose() }}
              onMouseEnter={() => setHoveredIcon(emoji)}
              onMouseLeave={() => setHoveredIcon(null)}
            >
              {emoji}
            </button>
          ))}
        </div>
        <button className="icon-picker-random" onClick={() => {
          const all = EMOJI_GROUPS.flat()
          onSelect(all[Math.floor(Math.random() * all.length)])
          onClose()
        }}>
          🎲 Random
        </button>
      </div>
    </div>
  )
}

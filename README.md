# WikiLivee

Вики-редактор на базе [TipTap](https://tiptap.dev/) (ProseMirror) с страницами, встраиванием таблиц, совместным редактированием и AI-инструментами.

## Горячие клавиши

**Mod** — на macOS это **⌘ Command**, на Windows и Linux — **Ctrl**.

### Редактор (TipTap StarterKit и расширения)

| Действие | Сочетание |
|----------|-----------|
| Отменить | Mod+Z; на русской раскладке также Mod+Я |
| Повторить | Mod+Shift+Z или Mod+Y; на русской раскладке Mod+Shift+Я |
| Жирный | Mod+B |
| Курсив | Mod+I |
| Подчёркнутый | Mod+U |
| Зачёркнутый | Mod+Shift+S |
| Инлайн-код | Mod+E |
| Заголовок уровня 1–3 | Mod+Alt+1 / Mod+Alt+2 / Mod+Alt+3 |
| Цитата (blockquote) | Mod+Shift+B |
| Блок кода | Mod+Alt+C |
| Мягкий перенос строки внутри абзаца | Shift+Enter или Mod+Enter |

### Меню по символу «/» (slash)

| Действие | Сочетание |
|----------|-----------|
| Открыть меню блоков | Ввести **`/`** в начале строки или после пробела (как настроено в редакторе) |
| Переместить выбор | **↑** / **↓** |
| Вставить выбранный блок | **Enter** |
| Закрыть меню без вставки | **Escape** |

Через меню доступны те же сценарии, что и в подсказках: абзац, заголовки, списки, вставка таблицы, ссылки на страницу и др. (см. интерфейс редактора).

### Оболочка приложения

| Действие | Сочетание |
|----------|-----------|
| Закрыть боковую панель (граф, доступ, версии и т.п.) | **Escape** (когда панель открыта) |
| Закрыть панель AI | **Escape** (когда открыта) |
| С поля заголовка страницы перейти в начало текста редактора | **Enter** |

---

Сочетания в первой таблице соответствуют пакетам `@tiptap/starter-kit`, `@tiptap/extension-bold`, `@tiptap/extension-italic`, `@tiptap/extension-underline`, `@tiptap/extension-strike`, `@tiptap/extension-code`, `@tiptap/extension-heading`, `@tiptap/extension-blockquote`, `@tiptap/extension-code-block`, `@tiptap/extension-hard-break`, `@tiptap/extensions` (undo/redo). При обновлении TipTap набор клавиш может слегка измениться — ориентируйтесь на документацию [TipTap Keyboard shortcuts](https://tiptap.dev/docs/editor/getting-started/overview).

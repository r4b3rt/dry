package termui

// Cursor manages a cursor.
type cursor interface {
	HideCursor()
	ShowCursor(int, int)
}

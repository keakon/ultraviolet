package uv

import (
	"testing"
	"unsafe"
)

// A screen buffer keeps one cell per screen position, and every buffer write
// copies a cell, so the layout of a cell drives both the memory and the copy
// cost of a screen. These bounds keep the layout from growing back.
func TestCellLayoutSize(t *testing.T) {
	cell := unsafe.Sizeof(Cell{})
	if cell > 48 {
		t.Errorf("Cell size = %d bytes, want at most 48", cell)
	}
	style := unsafe.Sizeof(Style{})
	if style > 16 {
		t.Errorf("Style size = %d bytes, want at most 16", style)
	}
	color := unsafe.Sizeof(Color{})
	if color > 4 {
		t.Errorf("Color size = %d bytes, want at most 4", color)
	}
	link := unsafe.Sizeof(Link{})
	if link > 8 {
		t.Errorf("Link size = %d bytes, want at most 8", link)
	}
}

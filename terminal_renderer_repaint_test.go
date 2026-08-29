package uv

import (
	"bytes"
	"image/color"
	"strings"
	"testing"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
)

// repaintCardBg mimics a message card's surface background.
var repaintCardBg = color.RGBA{R: 0x4e, G: 0x4e, B: 0x4e, A: 0xff}

func repaintBlank() *Cell {
	return &Cell{Content: " ", Width: 1, Style: Style{Bg: repaintCardBg}}
}

// fillRepaintRow writes the given cells starting at column 0 of the row,
// advancing by each cell's width, then fills the rest of the row with styled
// blanks when fill is true.
func fillRepaintRow(buf *RenderBuffer, y int, fill bool, cells ...Cell) {
	x := 0
	for _, c := range cells {
		cc := c
		buf.SetCell(x, y, &cc)
		x += c.Width
	}
	if fill {
		for ; x < buf.Width(); x++ {
			buf.SetCell(x, y, repaintBlank())
		}
	}
	buf.TouchLine(0, y, buf.Width())
}

func newRepaintRenderer(t *testing.T) (*TerminalRenderer, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	r := NewTerminalRenderer(&buf, []string{"TERM=xterm-256color"})
	r.SetColorProfile(colorprofile.TrueColor)
	return r, &buf
}

// A row holding a wide cell must be repainted through erase-to-end-of-line so
// the trailing background reaches the physical end of the line even when the
// terminal advances a wide glyph by a different width than the model measured.
func TestRepaintLineWideCellEraseToPhysicalEndOfLine(t *testing.T) {
	r, buf := newRepaintRenderer(t)
	r.EnterAltScreen()
	cellbuf := NewRenderBuffer(20, 1)
	fillRepaintRow(cellbuf, 0, true,
		Cell{Content: "a", Width: 1},
		Cell{Content: "1️⃣", Width: 2}, // keycap: often painted one column wide
		Cell{Content: "b", Width: 1},
	)
	r.Render(cellbuf)
	r.Flush()

	out := buf.String()
	// The row is erased before its content is rewritten, and the erase is
	// styled with the row's own background so the tail keeps the surface.
	styledErase := "\x1b[48;2;78;78;78m" + ansi.EraseLineRight
	if !strings.Contains(out, styledErase) {
		t.Fatalf("expected background-styled erase-to-end-of-line, got %q", out)
	}
	if !strings.Contains(out, "a") || !strings.Contains(out, "b") {
		t.Fatalf("content cells missing from output: %q", out)
	}
	// The content is written after the erase, from a known cursor state.
	if i := strings.Index(out, "\r"); i < 0 || strings.Index(out, "a") < i {
		t.Fatalf("content not rewritten after carriage-return anchor: %q", out)
	}
}

// A full-width content row (no trailing blanks) with a wide cell must also go
// through the erase-and-rewrite path: a cell-level diff would write the row
// from a drifted cursor and leave the last column unpainted.
func TestRepaintLineFullWidthContentRow(t *testing.T) {
	r, buf := newRepaintRenderer(t)
	r.EnterAltScreen()
	cellbuf := NewRenderBuffer(10, 1)
	fillRepaintRow(cellbuf, 0, false,
		Cell{Content: "a", Width: 1},
		Cell{Content: "b", Width: 1},
		Cell{Content: "1️⃣", Width: 2},
		Cell{Content: "c", Width: 1},
		Cell{Content: "d", Width: 1},
		Cell{Content: "e", Width: 1},
		Cell{Content: "f", Width: 1},
		Cell{Content: "g", Width: 1},
		Cell{Content: "h", Width: 1},
	)
	r.Render(cellbuf)
	r.Flush()

	out := buf.String()
	if !strings.Contains(out, ansi.EraseLineRight) {
		t.Fatalf("expected erase-to-end-of-line repaint, got %q", out)
	}
	for _, ch := range []string{"a", "c", "h"} {
		if !strings.Contains(out, ch) {
			t.Fatalf("content cell %q missing from output: %q", ch, out)
		}
	}
}

// When the wide cell disappears from the row, the repaint must still happen
// once: the previous frame's paint may have drifted, and the diff path would
// inherit that drift.
func TestRepaintLineWideCellRemoved(t *testing.T) {
	r, buf := newRepaintRenderer(t)
	r.EnterAltScreen()
	cellbuf := NewRenderBuffer(10, 1)
	fillRepaintRow(cellbuf, 0, true,
		Cell{Content: "a", Width: 1},
		Cell{Content: "1️⃣", Width: 2},
		Cell{Content: "b", Width: 1},
	)
	r.Render(cellbuf)
	r.Flush()

	buf.Reset()
	cellbuf2 := NewRenderBuffer(10, 1)
	fillRepaintRow(cellbuf2, 0, true,
		Cell{Content: "a", Width: 1},
		Cell{Content: "b", Width: 1},
		Cell{Content: "c", Width: 1},
	)
	r.Render(cellbuf2)
	r.Flush()

	out := buf.String()
	if !strings.Contains(out, ansi.EraseLineRight) {
		t.Fatalf("expected repaint after wide cell removal, got %q", out)
	}
	if !strings.Contains(out, "abc") {
		t.Fatalf("rewritten content missing: %q", out)
	}
}

// Rows without wide cells keep the cell-level diff path.
func TestRepaintLineNotTriggeredWithoutWideCells(t *testing.T) {
	r, buf := newRepaintRenderer(t)
	r.EnterAltScreen()
	cellbuf := NewRenderBuffer(10, 1)
	fillRepaintRow(cellbuf, 0, true,
		Cell{Content: "a", Width: 1},
		Cell{Content: "b", Width: 1},
	)
	r.Render(cellbuf)
	r.Flush()

	out := buf.String()
	if strings.Contains(out, "\r") {
		t.Fatalf("plain row should not be anchored with carriage return: %q", out)
	}
	if !strings.Contains(out, "ab") {
		t.Fatalf("content missing: %q", out)
	}
}

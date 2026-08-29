package uv

import (
	"bytes"
	"image/color"
	"regexp"
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

// Inline layout: the card surface ends before the terminal width, leaving a
// margin of unstyled blanks. The erase must use the surface's background (so
// the surface tail is correct without per-cell writes), and the margin must be
// restored afterwards from an exactly anchored position, without carrying the
// surface background past the surface edge.
func TestRepaintLineWidthCellWithOuterMargin(t *testing.T) {
	r, buf := newRepaintRenderer(t)
	r.EnterAltScreen()
	cellbuf := NewRenderBuffer(12, 1)
	fillRepaintRow(cellbuf, 0, true,
		Cell{Content: "a", Width: 1},
		Cell{Content: "1️⃣", Width: 2},
		Cell{Content: "b", Width: 1},
	)
	// Shrink the surface to columns 0..8: overwrite columns 9..11 with
	// unstyled blanks (the outer margin).
	for x := 9; x < 12; x++ {
		empty := EmptyCell
		cellbuf.SetCell(x, 0, &empty)
	}
	cellbuf.TouchLine(0, 0, 12)
	r.Render(cellbuf)
	r.Flush()

	out := buf.String()
	// The surface is erased with its own background.
	if !strings.Contains(out, "\x1b[48;2;78;78;78m"+ansi.EraseLineRight) {
		t.Fatalf("expected surface-background erase, got %q", out)
	}
	if !strings.Contains(out, "a1️⃣b") {
		t.Fatalf("content missing: %q", out)
	}
	// The margin is restored by a second erase from the exact surface end.
	marginAnchor := "\r" + ansi.CursorForward(9)
	i := strings.Index(out, marginAnchor)
	if i < 0 {
		t.Fatalf("expected margin restore anchored at the surface end, got %q", out)
	}
	tail := out[i:]
	if !strings.Contains(tail, ansi.EraseLineRight) {
		t.Fatalf("expected margin erase after the anchor, got %q", tail)
	}
	if strings.Contains(tail, "\x1b[48;") {
		t.Fatalf("margin erase must not carry the surface background: %q", tail)
	}
}

// csiSequence matches ANSI CSI sequences, used to inspect the visible text of
// a rendered output.
var csiSequence = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

func stripANSI(s string) string {
	return csiSequence.ReplaceAllString(s, "")
}

// Cells inside the surface that already match the erased pen (blank padding
// between two text segments on a card row) must still be written: the cursor
// only advances by writing, and skipping one would shift every later write on
// the row one column left.
func TestRepaintLineInteriorBlankKeepsPositioning(t *testing.T) {
	r, buf := newRepaintRenderer(t)
	r.EnterAltScreen()
	cellbuf := NewRenderBuffer(12, 1)
	fillRepaintRow(cellbuf, 0, true,
		Cell{Content: "1️⃣", Width: 2},
		Cell{Content: "a", Width: 1},
		Cell{Content: " ", Width: 1, Style: Style{Bg: repaintCardBg}},
		Cell{Content: "b", Width: 1},
	)
	r.Render(cellbuf)
	r.Flush()

	out := stripANSI(buf.String())
	if !strings.Contains(out, "1️⃣a b") {
		t.Fatalf("interior blank must be written between content cells, got %q", out)
	}
}

// A wide cell at the very start of a full-width row: the repaint writes the
// row from column 0 and the wide glyph is the first written cell.
func TestRepaintLineWideCellAtLineStart(t *testing.T) {
	r, buf := newRepaintRenderer(t)
	r.EnterAltScreen()
	cellbuf := NewRenderBuffer(6, 1)
	fillRepaintRow(cellbuf, 0, false,
		Cell{Content: "1️⃣", Width: 2},
		Cell{Content: "a", Width: 1},
		Cell{Content: "b", Width: 1},
		Cell{Content: "c", Width: 1},
		Cell{Content: "d", Width: 1},
	)
	r.Render(cellbuf)
	r.Flush()

	out := buf.String()
	if !strings.Contains(out, ansi.EraseLineRight) {
		t.Fatalf("expected erase-to-end-of-line repaint, got %q", out)
	}
	if !strings.Contains(stripANSI(out), "1️⃣abcd") {
		t.Fatalf("content missing: %q", out)
	}
}

// A wide row followed by a changed plain row: the move to the next row must
// be anchored (carriage return) so the drifted cursor left by the repaint
// cannot corrupt the next row's position.
func TestRepaintLineCrossRowMove(t *testing.T) {
	r, buf := newRepaintRenderer(t)
	r.EnterAltScreen()
	cellbuf := NewRenderBuffer(10, 2)
	fillRepaintRow(cellbuf, 0, true,
		Cell{Content: "1️⃣", Width: 2},
		Cell{Content: "a", Width: 1},
	)
	fillRepaintRow(cellbuf, 1, true,
		Cell{Content: "b", Width: 1},
		Cell{Content: "c", Width: 1},
	)
	r.Render(cellbuf)
	r.Flush()

	out := buf.String()
	if !strings.Contains(out, "\r\nbc") {
		t.Fatalf("expected CR-anchored move to the next row, got %q", out)
	}
}

// Repainting is deterministic: when a wide row is touched but both frames
// agree, the row already shows the repainted pixels and the frame needs no
// output at all.
func TestRepaintLineSkipWhenFramesAgree(t *testing.T) {
	r, buf := newRepaintRenderer(t)
	r.EnterAltScreen()
	cellbuf := NewRenderBuffer(10, 1)
	fillRepaintRow(cellbuf, 0, true,
		Cell{Content: "1️⃣", Width: 2},
		Cell{Content: "a", Width: 1},
		Cell{Content: "b", Width: 1},
	)
	r.Render(cellbuf)
	r.Flush()

	buf.Reset()
	cellbuf2 := NewRenderBuffer(10, 1)
	fillRepaintRow(cellbuf2, 0, true,
		Cell{Content: "1️⃣", Width: 2},
		Cell{Content: "a", Width: 1},
		Cell{Content: "b", Width: 1},
	)
	r.Render(cellbuf2)
	r.Flush()

	if out := buf.String(); out != "" {
		t.Fatalf("identical wide row should be skipped, got %q", out)
	}
}

// After a width-changing resize the model must adopt the repainted row so a
// later identical frame is skipped: the resize frame repaints with the new
// width and the renderer's resize block syncs the model afterwards, so the
// next identical frame needs no output at all.
func TestRepaintLineResizeSyncsModel(t *testing.T) {
	r, buf := newRepaintRenderer(t)
	r.EnterAltScreen()

	render := func(width int, content string) string {
		buf.Reset()
		cellbuf := NewRenderBuffer(width, 1)
		fillRepaintRow(cellbuf, 0, true,
			Cell{Content: "1️⃣", Width: 2},
			Cell{Content: content, Width: 1},
		)
		r.Render(cellbuf)
		r.Flush()
		return buf.String()
	}

	render(10, "a")
	// Resize wider with different content: repaint with the new width.
	out2 := render(12, "b")
	if !strings.Contains(out2, ansi.EraseLineRight) || !strings.Contains(out2, "b") {
		t.Fatalf("expected repaint on resize, got %q", out2)
	}
	// The model matches the new buffer now: identical frames are skipped.
	if out3 := render(12, "b"); out3 != "" {
		t.Fatalf("expected skip after resize, got %q", out3)
	}
}

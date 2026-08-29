package uv

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// The screen parser must keep grapheme clusters whole: a base rune followed by
// zero-width sequences (variation selectors, combining enclosing marks) is one
// cluster measured by the screen's width method. Dropping the zero-width tail
// — the decoder reports it as a separate width-0 sequence — loses the emoji
// presentation and shortens the row by one column against width methods that
// charge the cluster, which leaves card backgrounds short of their edge.

func TestParseKeepsKeycapClusterWhole(t *testing.T) {
	sb := NewScreenBuffer(8, 1)
	sb.Method = ansi.GraphemeWidth
	NewStyledString("ab1️⃣c").Draw(&sb, Rect(0, 0, 8, 1))

	row := sb.RenderBuffer.Line(0)
	if c := row.At(2); c.Content != "1️⃣" || c.Width != 2 {
		t.Fatalf("keycap cluster not stored whole: content=%q width=%d, want %q width 2", c.Content, c.Width, "1️⃣")
	}
	if c := row.At(3); !c.IsZero() {
		t.Fatalf("wide cluster continuation column not reserved: content=%q width=%d", c.Content, c.Width)
	}
	if c := row.At(4); c.Content != "c" {
		t.Fatalf("cluster pushed the following cell: content=%q at x=4", c.Content)
	}
}

func TestParseKeycapWidthFollowsScreenMethod(t *testing.T) {
	sb := NewScreenBuffer(8, 1)
	sb.Method = ansi.WcWidth
	NewStyledString("ab1️⃣c").Draw(&sb, Rect(0, 0, 8, 1))

	row := sb.RenderBuffer.Line(0)
	if c := row.At(2); c.Content != "1️⃣" || c.Width != 1 {
		t.Fatalf("wcwidth cluster: content=%q width=%d, want %q width 1", c.Content, c.Width, "1️⃣")
	}
	if c := row.At(3); c.Content != "c" {
		t.Fatalf("cluster pushed the following cell: content=%q at x=3", c.Content)
	}
}

// A combining mark separated from its base by an escape sequence must fold
// into the base's cell instead of being dropped or clobbering the next cell.
func TestParseFoldsMarkAcrossEscape(t *testing.T) {
	sb := NewScreenBuffer(4, 1)
	sb.Method = ansi.GraphemeWidth
	NewStyledString("e\x1b[31ḿ\x1b[mx").Draw(&sb, Rect(0, 0, 4, 1))

	row := sb.RenderBuffer.Line(0)
	if c := row.At(0); c.Content != "é" {
		t.Fatalf("combining mark not folded into base: content=%q", c.Content)
	}
	if c := row.At(1); c.Content != "x" {
		t.Fatalf("mark clobbered the following cell: content=%q", c.Content)
	}
}

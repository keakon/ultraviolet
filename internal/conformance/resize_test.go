package conformance_test

import (
	"testing"

	"github.com/keakon/ultraviolet/internal/conformance"
)

// A frame that collapses to no rows between two renders and grows back has
// already lost the rows it dropped by the time the renderer is handed the
// buffer again: neither Render saw the collapse, and the rows that return are
// blank. Their old content is still in the renderer's model, so a row the
// buffer does not report as changed is exactly the row the screen keeps
// showing.
func TestInlineFrameCollapseAndRegrowErasesLostRows(t *testing.T) {
	const wave = "\U0001f44b\U0001f3ff" // four columns under legacy widths

	progs := []struct {
		name string
		prog conformance.Program
	}{
		// The shape the nightly fuzz target minimised to: the frame grows back
		// one row taller, and the drawn row only happens to be the new one.
		{
			name: "grows back taller",
			prog: conformance.Program{
				Width: 23, Height: 2, GraphemeWidth: true, Inline: true,
				Ops: []conformance.Op{
					{Kind: conformance.OpDrawLine, Y: 0, Text: wave},
					{Kind: conformance.OpRender},
					{Kind: conformance.OpResize, W: 23, H: 0},
					{Kind: conformance.OpResize, W: 23, H: 3},
					{Kind: conformance.OpDrawLine, Y: 1, Text: wave},
				},
			},
		},
		// The frame returns to the shape it had, so not even a frame resize
		// reaches the renderer: only the buffer's own report of the loss can
		// make it revisit the row.
		{
			name: "back to the old shape",
			prog: conformance.Program{
				Width: 23, Height: 2, GraphemeWidth: true, Inline: true,
				Ops: []conformance.Op{
					{Kind: conformance.OpDrawLine, Y: 0, Text: wave},
					{Kind: conformance.OpRender},
					{Kind: conformance.OpResize, W: 23, H: 0},
					{Kind: conformance.OpResize, W: 23, H: 2},
				},
			},
		},
	}

	for _, tc := range progs {
		t.Run(tc.name, func(t *testing.T) {
			for _, o := range oracles {
				got := runIncremental(t, tc.prog, o.mk)
				want := runFullRepaint(t, tc.prog, o.mk)
				compare(t, tc.prog, o.name, got, want)
			}
		})
	}
}

package uv

import (
	"image/color"
	"strings"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
)

// EmptyCell is a cell with a single space, width of 1, and no style or link.
var EmptyCell = Cell{Content: " ", Width: 1}

// Cell represents a single cell in the terminal screen.
type Cell struct {
	// Content is the [Cell]'s content, which consists of a single grapheme
	// cluster. Most of the time, this will be a single rune as well, but it
	// can also be a combination of runes that form a grapheme cluster.
	Content string

	// The style of the cell. Nil style means no style. Zero value prints a
	// reset sequence.
	Style Style

	// Link is the hyperlink of the cell.
	Link Link

	// Width is the mono-spaced width of the grapheme cluster.
	Width int
}

// NewCell creates a new cell from the given string grapheme. It will only use
// the first grapheme in the string and ignore the rest. The width of the cell
// is determined using the given width method.
func NewCell(method WidthMethod, gr string) *Cell {
	if len(gr) == 0 {
		return &Cell{}
	}
	if gr == " " {
		return EmptyCell.Clone()
	}
	return &Cell{
		Content: gr,
		Width:   method.StringWidth(gr),
	}
}

// String returns the string content of the cell excluding any styles, links,
// and escape sequences.
func (c *Cell) String() string {
	return c.Content
}

// Equal returns whether the cell is equal to the other cell.
func (c *Cell) Equal(o *Cell) bool {
	return o != nil &&
		c.Width == o.Width &&
		c.Content == o.Content &&
		c.Style.Equal(&o.Style) &&
		c.Link.Equal(o.Link)
}

// IsZero returns whether the cell is an empty cell.
func (c *Cell) IsZero() bool {
	return *c == Cell{}
}

// isWidePlaceholder reports whether the cell is the continuation column of a
// wide cell, marked with a zero display width.
func (c *Cell) isWidePlaceholder() bool {
	return c != nil && c.Width == 0
}

// Clone returns a copy of the cell.
func (c *Cell) Clone() (n *Cell) {
	n = new(Cell)
	*n = *c
	return
}

// Empty makes the cell an empty cell by setting its content to a single space
// and width to 1.
func (c *Cell) Empty() {
	c.Content = " "
	c.Width = 1
}

// NewLink creates a new hyperlink with the given URL and parameters.
func NewLink(url string, params ...string) Link {
	return newLink(url, strings.Join(params, ":"))
}

// newLink builds a link from its raw fields. Both fields must be empty for the
// result to be the zero link, so that an empty link compares equal to no link.
func newLink(url, params string) Link {
	if url == "" && params == "" {
		return Link{}
	}
	return Link{data: &linkData{
		URL:    url,
		Params: params,
	}}
}

// Link represents a hyperlink in the terminal screen. The zero value is a link
// with no URL.
//
// A link holds a single pointer to its URL and parameters: a screen buffer
// keeps a link for every cell, and hyperlinks are rare, so cells share one
// link instead of embedding its strings.
type Link struct {
	data *linkData
}

// linkData holds the URL and parameters of a [Link].
type linkData struct {
	URL    string
	Params string
}

// URL returns the link's URL, or an empty string if the link is unset.
func (h Link) URL() string {
	if h.data == nil {
		return ""
	}
	return h.data.URL
}

// Params returns the link's parameters, or an empty string if the link is
// unset.
func (h Link) Params() string {
	if h.data == nil {
		return ""
	}
	return h.data.Params
}

// String returns a string representation of the hyperlink.
func (h Link) String() string {
	return h.URL()
}

// Equal returns whether the hyperlink is equal to the other hyperlink.
func (h Link) Equal(o Link) bool {
	if h.data == o.data {
		return true
	}
	if h.data == nil || o.data == nil {
		return false
	}
	return *h.data == *o.data
}

// IsZero returns whether the hyperlink is empty.
func (h Link) IsZero() bool {
	return h.data == nil || *h.data == linkData{}
}

// These are the available text attributes that can be combined to create
// different styles.
const (
	AttrBold = 1 << iota
	AttrFaint
	AttrItalic
	AttrBlink
	AttrRapidBlink // Not widely supported
	AttrReverse
	AttrConceal
	AttrStrikethrough

	AttrReset = 0
)

// AttrSlowBlink is an alias for AttrBlink.
//
// Deprecated: Use [AttrBlink] instead.
const (
	AttrSlowBlink = AttrBlink
)

// Underline is the style of underline to use for text.
type Underline = ansi.Underline

// UnderlineStyle is the style of underline to use for text.
//
// Deprecated: Use [Underline] instead.
type UnderlineStyle = ansi.Underline

// These are the available underline styles.
const (
	UnderlineNone   = ansi.UnderlineNone
	UnderlineSingle = ansi.UnderlineSingle
	UnderlineDouble = ansi.UnderlineDouble
	UnderlineCurly  = ansi.UnderlineCurly
	UnderlineDotted = ansi.UnderlineDotted
	UnderlineDashed = ansi.UnderlineDashed
)

// These are the available underline styles.
//
// Deprecated: Use the constants from [Underline] instead.
const (
	UnderlineStyleNone   = ansi.UnderlineNone
	UnderlineStyleSingle = ansi.UnderlineSingle
	UnderlineStyleDouble = ansi.UnderlineDouble
	UnderlineStyleCurly  = ansi.UnderlineCurly
	UnderlineStyleDotted = ansi.UnderlineDotted
	UnderlineStyleDashed = ansi.UnderlineDashed
)

// colorKind discriminates the representation of a [Color].
type colorKind uint8

const (
	colorKindNone colorKind = iota
	colorKindBasic
	colorKindIndexed
	colorKindRGB
)

// Color is a terminal color: an ANSI 3/4-bit or 8-bit palette index, or a
// 24-bit RGB value. The zero value means no color, which renders as the
// terminal's default color.
//
// The whole color fits in one word: a screen buffer stores a cell for every
// position, and every cell is copied on each buffer write, so the size of a
// [Color] directly drives the memory and copy cost of a screen.
type Color struct {
	v uint32 // kind in the top 2 bits, payload in the rest
}

// ColorFrom returns the [Color] matching c, which may be an [ansi.BasicColor],
// an [ansi.IndexedColor], or any other [color.Color] handled as a 24-bit RGB
// value. It returns the zero [Color] if c is nil.
func ColorFrom(c color.Color) Color {
	switch c := c.(type) {
	case nil:
		return Color{}
	case Color:
		return c
	case ansi.BasicColor:
		return Color{v: uint32(colorKindBasic)<<30 | uint32(c)}
	case ansi.IndexedColor:
		return Color{v: uint32(colorKindIndexed)<<30 | uint32(c)}
	default:
		r, g, b, _ := c.RGBA()
		return Color{v: uint32(colorKindRGB)<<30 | (r>>8)<<16 | (g>>8)<<8 | (b >> 8)}
	}
}

func (c Color) kind() colorKind {
	return colorKind(c.v >> 30)
}

func (c Color) payload() uint32 {
	return c.v & (1<<30 - 1)
}

// IsZero returns whether the color is unset.
func (c Color) IsZero() bool {
	return c.v == 0
}

// Equal returns whether the two colors are the same color. Colors of different
// representations compare equal when they resolve to the same RGB color.
func (c Color) Equal(o Color) bool {
	if c.v == o.v {
		return true
	}
	if c.v == 0 || o.v == 0 {
		return false
	}
	if c.kind() != o.kind() {
		cr, cg, cb, ca := c.RGBA()
		or, og, ob, oa := o.RGBA()
		return cr == or && cg == og && cb == ob && ca == oa
	}
	return false
}

// RGBA returns the red, green, blue, and alpha components of the color. It
// implements [color.Color].
func (c Color) RGBA() (r, g, b, a uint32) {
	switch c.kind() {
	case colorKindBasic:
		return ansi.BasicColor(c.payload()).RGBA()
	case colorKindIndexed:
		return ansi.IndexedColor(c.payload()).RGBA()
	case colorKindRGB:
		rgb := c.payload()
		r, g, b = (rgb>>16&0xff)*0x101, (rgb>>8&0xff)*0x101, (rgb&0xff)*0x101
		return r, g, b, 0xffff
	default:
		return 0, 0, 0, 0
	}
}

// color returns the color as a [color.Color] for the ANSI encoder. It returns
// nil if the color is unset.
func (c Color) color() color.Color {
	switch c.kind() {
	case colorKindBasic:
		return ansi.BasicColor(c.payload())
	case colorKindIndexed:
		return ansi.IndexedColor(c.payload())
	case colorKindRGB:
		return c
	default:
		return nil
	}
}

// Style represents the style of a cell.
type Style struct {
	Fg             Color
	Bg             Color
	UnderlineColor Color
	Underline      Underline
	Attrs          uint8
}

// Equal returns true if the style is equal to the other style.
func (s *Style) Equal(o *Style) bool {
	return s.Attrs == o.Attrs &&
		s.Underline == o.Underline &&
		s.Fg.Equal(o.Fg) &&
		s.Bg.Equal(o.Bg) &&
		s.UnderlineColor.Equal(o.UnderlineColor)
}

// Styled wraps the given string with the style's ANSI sequences and resets.
func (s *Style) Styled(str string) string {
	if s.IsZero() {
		return str
	}
	return s.String() + str + ansi.ResetStyle
}

// String returns the ANSI SGR sequence for the style.
func (s *Style) String() string {
	if s.IsZero() {
		return ansi.ResetStyle
	}

	var b ansi.Style

	if s.Attrs != 0 { //nolint:nestif
		if s.Attrs&AttrBold != 0 {
			b = b.Bold()
		}
		if s.Attrs&AttrFaint != 0 {
			b = b.Faint()
		}
		if s.Attrs&AttrItalic != 0 {
			b = b.Italic(true)
		}
		if s.Attrs&AttrBlink != 0 {
			b = b.Blink(true)
		}
		if s.Attrs&AttrRapidBlink != 0 {
			b = b.RapidBlink(true)
		}
		if s.Attrs&AttrReverse != 0 {
			b = b.Reverse(true)
		}
		if s.Attrs&AttrConceal != 0 {
			b = b.Conceal(true)
		}
		if s.Attrs&AttrStrikethrough != 0 {
			b = b.Strikethrough(true)
		}
	}
	if s.Underline != UnderlineStyleNone {
		switch s.Underline {
		case UnderlineStyleSingle:
			b = b.Underline(true)
		case UnderlineStyleDouble:
			b = b.UnderlineStyle(UnderlineStyleDouble)
		case UnderlineStyleCurly:
			b = b.UnderlineStyle(UnderlineStyleCurly)
		case UnderlineStyleDotted:
			b = b.UnderlineStyle(UnderlineStyleDotted)
		case UnderlineStyleDashed:
			b = b.UnderlineStyle(UnderlineStyleDashed)
		}
	}
	if !s.Fg.IsZero() {
		b = b.ForegroundColor(s.Fg.color())
	}
	if !s.Bg.IsZero() {
		b = b.BackgroundColor(s.Bg.color())
	}
	if !s.UnderlineColor.IsZero() {
		b = b.UnderlineColor(s.UnderlineColor.color())
	}

	return b.String()
}

// Diff returns the ANSI sequence that sets the style as a diff from
// another style.
func (s *Style) Diff(from *Style) string {
	return StyleDiff(from, s)
}

// StyleDiff returns the SGR ANSI sequence necessary to transition from the
// "from" style to the "to" style.
func StyleDiff(from, to *Style) string {
	if from == nil && to == nil {
		return ""
	}
	if from != nil && to != nil && from.Equal(to) {
		return ""
	}
	if from == nil {
		return to.String()
	}
	if to == nil || to.IsZero() {
		// Resetting all styles is cheaper than calculating diffs.
		// "\x1b[m" is 3 bytes vs potentially much longer sequences.
		return ansi.ResetStyle
	}

	// TODO: Optimize further by checking if a full reset is cheaper than
	// calculating diffs. Often, it might be cheaper to reset everything and
	// then set the desired styles rather than calculating diffs. Also more
	// compatible with terminals that have buggy SGR implementations.

	var b ansi.Style

	if !from.Fg.Equal(to.Fg) {
		b = b.ForegroundColor(to.Fg.color())
	}

	if !from.Bg.Equal(to.Bg) {
		b = b.BackgroundColor(to.Bg.color())
	}

	if !from.UnderlineColor.Equal(to.UnderlineColor) {
		// TODO: Investigate this. For backward compatibility, we might want to
		// set this at the end instead. Because on terminals that don't support
		// underline color, this might mess up some other attributes if set in
		// the middle.
		b = b.UnderlineColor(to.UnderlineColor.color())
	}

	fromBold := from.Attrs&AttrBold != 0
	fromFaint := from.Attrs&AttrFaint != 0
	fromItalic := from.Attrs&AttrItalic != 0
	fromUnderline := from.Underline != UnderlineStyleNone
	fromBlink := from.Attrs&AttrBlink != 0
	fromRapidBlink := from.Attrs&AttrRapidBlink != 0
	fromReverse := from.Attrs&AttrReverse != 0
	fromConceal := from.Attrs&AttrConceal != 0
	fromStrikethrough := from.Attrs&AttrStrikethrough != 0
	toBold := to.Attrs&AttrBold != 0
	toFaint := to.Attrs&AttrFaint != 0
	toItalic := to.Attrs&AttrItalic != 0
	toUnderline := to.Underline != UnderlineStyleNone
	toBlink := to.Attrs&AttrBlink != 0
	toRapidBlink := to.Attrs&AttrRapidBlink != 0
	toReverse := to.Attrs&AttrReverse != 0
	toConceal := to.Attrs&AttrConceal != 0
	toStrikethrough := to.Attrs&AttrStrikethrough != 0

	// We perform the resets first since they are single attributes and
	// shouldn't interfere with others being set.

	boldChanged := fromBold != toBold
	faintChanged := fromFaint != toFaint
	if boldChanged || faintChanged {
		if fromBold && !toBold || fromFaint && !toFaint {
			b = b.Normal()
			boldChanged = true
			faintChanged = true
		}
	}

	italicChanged := fromItalic != toItalic
	if italicChanged && !toItalic {
		b = b.Italic(false)
	}

	underlineChanged := fromUnderline != toUnderline || from.Underline != to.Underline
	if underlineChanged && !toUnderline {
		b = b.Underline(false)
	}

	blinkChanged := fromBlink != toBlink
	rapidBlinkChanged := fromRapidBlink != toRapidBlink
	if blinkChanged || rapidBlinkChanged {
		if fromBlink && !toBlink || fromRapidBlink && !toRapidBlink {
			b = b.Blink(false)
			blinkChanged = true
			rapidBlinkChanged = true
		}
	}

	reverseChanged := fromReverse != toReverse
	if reverseChanged && !toReverse {
		b = b.Reverse(false)
	}

	concealChanged := fromConceal != toConceal
	if concealChanged && !toConceal {
		b = b.Conceal(false)
	}

	strikethroughChanged := fromStrikethrough != toStrikethrough
	if strikethroughChanged && !toStrikethrough {
		b = b.Strikethrough(false)
	}

	if boldChanged && toBold {
		b = b.Bold()
	}

	if faintChanged && toFaint {
		b = b.Faint()
	}

	if italicChanged && toItalic {
		b = b.Italic(true)
	}

	if underlineChanged && toUnderline && to.Underline == UnderlineStyleSingle {
		// We only handle single underline here since others require more
		// specific handling at the end.
		b = b.Underline(true)
	}

	if blinkChanged && toBlink {
		b = b.Blink(true)
	}

	if rapidBlinkChanged && toRapidBlink {
		b = b.RapidBlink(true)
	}

	if reverseChanged && toReverse {
		b = b.Reverse(true)
	}

	if concealChanged && toConceal {
		b = b.Conceal(true)
	}

	if strikethroughChanged && toStrikethrough {
		b = b.Strikethrough(true)
	}

	// Handle special underline styles.
	if underlineChanged && toUnderline && to.Underline > UnderlineStyleSingle {
		b = b.UnderlineStyle(to.Underline)
	}

	return b.String()
}

// IsZero returns true if the style is empty.
func (s *Style) IsZero() bool {
	return *s == Style{}
}

// ConvertStyle converts a style to respect the given color profile.
func ConvertStyle(s Style, p colorprofile.Profile) Style {
	switch p {
	case colorprofile.TrueColor:
		return s
	case colorprofile.ANSI, colorprofile.ANSI256:
	case colorprofile.Ascii:
		s.Fg = Color{}
		s.Bg = Color{}
		s.UnderlineColor = Color{}
	case colorprofile.NoTTY:
		return Style{}
	}

	s.Fg = convertColor(s.Fg, p)
	s.Bg = convertColor(s.Bg, p)
	s.UnderlineColor = convertColor(s.UnderlineColor, p)
	return s
}

// convertColor converts a single color to respect the given color profile.
func convertColor(c Color, p colorprofile.Profile) Color {
	switch c.kind() {
	case colorKindNone, colorKindBasic:
		return c
	case colorKindIndexed:
		if p == colorprofile.ANSI {
			return ColorFrom(ansi.Convert16(ansi.IndexedColor(c.payload())))
		}
		return c
	default:
		switch p {
		case colorprofile.ANSI256:
			return ColorFrom(ansi.Convert256(c))
		case colorprofile.ANSI:
			return ColorFrom(ansi.Convert16(c))
		default:
			return c
		}
	}
}

// ConvertLink converts a hyperlink to respect the given color profile.
func ConvertLink(h Link, p colorprofile.Profile) Link {
	if p == colorprofile.NoTTY {
		return Link{}
	}
	return h
}

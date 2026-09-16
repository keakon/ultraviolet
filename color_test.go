package uv

import (
	"image/color"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

var testColors = []color.Color{
	nil,
	ansi.Black,
	ansi.Red,
	ansi.BrightBlue,
	ansi.BrightWhite,
	ansi.IndexedColor(0),
	ansi.IndexedColor(33),
	ansi.IndexedColor(255),
	ansi.TrueColor(0x010203),
	ansi.RGBColor{R: 1, G: 2, B: 3},
	color.RGBA{R: 1, G: 2, B: 3, A: 255},
	color.NRGBA{R: 1, G: 2, B: 3, A: 255},
	color.CMYK{C: 1, M: 2, Y: 3, K: 4},
	color.Gray{Y: 128},
}

// TestColorFromString checks that a compact color encodes exactly like the
// original color it was built from.
func TestColorFromString(t *testing.T) {
	for _, c := range testColors {
		if c == nil {
			if !ColorFrom(c).IsZero() {
				t.Errorf("ColorFrom(nil) = %#v, want the zero color", ColorFrom(nil))
			}
			continue
		}

		want := ansi.Style{}.ForegroundColor(c)
		style := Style{Fg: ColorFrom(c)}
		if got := style.String(); got != want.String() {
			t.Errorf("foreground %#v: got %q, want %q", c, got, want.String())
		}
		want = ansi.Style{}.BackgroundColor(c)
		style = Style{Bg: ColorFrom(c)}
		if got := style.String(); got != want.String() {
			t.Errorf("background %#v: got %q, want %q", c, got, want.String())
		}
		want = ansi.Style{}.UnderlineColor(c)
		style = Style{UnderlineColor: ColorFrom(c)}
		if got := style.String(); got != want.String() {
			t.Errorf("underline color %#v: got %q, want %q", c, got, want.String())
		}
	}
}

func TestColorEqual(t *testing.T) {
	cases := []struct {
		name string
		a, b color.Color
		want bool
	}{
		{"unset", nil, nil, true},
		{"unset against set", nil, ansi.Red, false},
		{"same basic", ansi.Red, ansi.Red, true},
		{"different basic", ansi.Red, ansi.Green, false},
		{"same indexed", ansi.IndexedColor(33), ansi.IndexedColor(33), true},
		{"different indexed", ansi.IndexedColor(33), ansi.IndexedColor(34), false},
		{"same rgb", color.RGBA{R: 1, G: 2, B: 3, A: 255}, color.RGBA{R: 1, G: 2, B: 3, A: 255}, true},
		{"different rgb", color.RGBA{R: 1, G: 2, B: 3, A: 255}, color.RGBA{R: 1, G: 2, B: 4, A: 255}, false},
		{"rgb across types", color.RGBA{R: 1, G: 2, B: 3, A: 255}, color.NRGBA{R: 1, G: 2, B: 3, A: 255}, true},
		{"basic against matching rgb", ansi.Red, color.RGBA{R: 0x80, A: 255}, true},
		{"indexed against matching rgb", ansi.IndexedColor(33), color.RGBA{B: 0xff, G: 0x87, A: 255}, true},
		{"indexed against other rgb", ansi.IndexedColor(33), color.RGBA{B: 0xff, A: 255}, false},
		{"black against unset", ansi.Black, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, b := ColorFrom(tc.a), ColorFrom(tc.b)
			if got := a.Equal(b); got != tc.want {
				t.Errorf("%#v.Equal(%#v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
			if got := b.Equal(a); got != tc.want {
				t.Errorf("%#v.Equal(%#v) = %v, want %v", tc.b, tc.a, got, tc.want)
			}
		})
	}
}

func TestColorRGBA(t *testing.T) {
	cases := []struct {
		name       string
		c          color.Color
		r, g, b, a uint32
	}{
		{"unset", nil, 0, 0, 0, 0},
		{"black", ansi.Black, 0, 0, 0, 0xffff},
		{"basic", ansi.Red, 0x8080, 0, 0, 0xffff},
		{"indexed", ansi.IndexedColor(33), 0, 0x8787, 0xffff, 0xffff},
		{"rgb", color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 255}, 0x1212, 0x3434, 0x5656, 0xffff},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, g, b, a := ColorFrom(tc.c).RGBA()
			if r != tc.r || g != tc.g || b != tc.b || a != tc.a {
				t.Errorf("RGBA() = %#x, %#x, %#x, %#x, want %#x, %#x, %#x, %#x",
					r, g, b, a, tc.r, tc.g, tc.b, tc.a)
			}
		})
	}
}

func TestLink(t *testing.T) {
	link := NewLink("https://example.com", "id=1")
	if link.URL() != "https://example.com" || link.Params() != "id=1" {
		t.Errorf("NewLink() = %q, %q", link.URL(), link.Params())
	}
	if link.String() != "https://example.com" {
		t.Errorf("Link.String() = %q", link.String())
	}
	if link.IsZero() {
		t.Error("Link.IsZero() = true")
	}
	if !link.Equal(NewLink("https://example.com", "id=1")) {
		t.Error("equal links do not compare equal")
	}
	if link.Equal(NewLink("https://example.com")) {
		t.Error("links with different parameters compare equal")
	}
	var zero Link
	if NewLink("").IsZero() != true || !zero.IsZero() {
		t.Error("empty links are not zero")
	}
	if zero.URL() != "" || zero.Params() != "" || zero.String() != "" {
		t.Error("the zero link is not empty")
	}
	if !zero.Equal(zero) || zero.Equal(link) {
		t.Error("zero link comparison is wrong")
	}
}

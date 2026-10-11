package font

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/tsawler/tabula/core"
)

func TestAdobeGlyphNames(t *testing.T) {
	cases := []struct {
		name, want string
		known      bool
	}{
		{"theta", "θ", true}, {"Gbreve", "Ğ", true}, {"afii10017", "А", true},
		{"alefhebrew", "א", true}, {"alefarabic", "ا", true},
		{"dalethatafpatah", "דֲ", true},
		{"A.swash", "A", true}, {"f_f_i", "ffi", true},
		{"Lcommaaccent_uni20AC0308_u1040C.alternate", "Ļ€\u0308\U0001040C", true},
		{"uni00410301", "A\u0301", true}, {"u1F600", "😀", true},
		{"u000000", "\x00", true}, {"u10FFFF", "\U0010FFFF", true},
		{"uniD801DC0C", "", false}, {"uni20ac", "", false},
		{"uni123", "", false}, {"uni0041D800", "", false},
		{"uD800", "", false}, {"u110000", "", false}, {"u1f600", "", false},
		{"u123", "", false}, {"u0000041", "", false},
		{".notdef", "", true}, {"unknownGlyph", "", false},
		{"unknown_A", "A", true}, {"Omega", "Ω", true},
		{"Omegagreek", "Ω", true}, {"mu", "µ", true}, {"mugreek", "μ", true},
		{"epsilon1", "ϵ", true}, {"parenleftbigg", "(", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, known := glyphToUnicode(c.name, false)
			if got != c.want || known != c.known {
				t.Fatalf("got (%q,%v), want (%q,%v)", got, known, c.want, c.known)
			}
		})
	}
}

func TestAdobeSnapshotCoverage(t *testing.T) {
	for _, list := range []struct {
		file  string
		zapf  bool
		count int
	}{
		{"glyphlist.txt", false, 4281}, {"zapfdingbats.txt", true, 201},
	} {
		t.Run(list.file, func(t *testing.T) {
			f, err := os.Open("data/agl/" + list.file)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			scan := bufio.NewScanner(f)
			count := 0
			for scan.Scan() {
				line := scan.Text()
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				fields := strings.Split(line, ";")
				var want strings.Builder
				for _, code := range strings.Fields(fields[1]) {
					r, err := strconv.ParseInt(code, 16, 32)
					if err != nil {
						t.Fatal(err)
					}
					want.WriteRune(rune(r))
				}
				got, ok := glyphToUnicode(fields[0], list.zapf)
				if !ok || got != want.String() {
					t.Errorf("%s = %q, want %q", fields[0], got, want.String())
				}
				// Suffix handling must work for every imported entry, including sequences.
				alternate, ok := glyphToUnicode(fields[0]+".alt", list.zapf)
				if !ok || alternate != got {
					t.Errorf("suffix changed %s", fields[0])
				}
				count++
			}
			if err := scan.Err(); err != nil {
				t.Fatal(err)
			}
			if count != list.count {
				t.Errorf("entries=%d, want %d", count, list.count)
			}
		})
	}
}

func TestGlyphSequencesThroughCustomEncoding(t *testing.T) {
	e := NewCustomEncodingFromGlyphs(WinAnsiEncoding, map[byte]string{
		'A': "f_f_i", 'B': "uni00410301", 'C': ".notdef", 'D': "unknownGlyph", 'E': "alefhebrew",
	})
	if got := e.DecodeString([]byte("ABCDE")); got != "ffiA\u0301Dא" {
		t.Fatalf("sequence lost: %q", got)
	}
	if got := e.Decode('A'); got != utf8.RuneError {
		t.Fatalf("multi-character Decode=%U", got)
	}
	if got := e.Decode('C'); got != 0 {
		t.Fatalf(".notdef Decode=%U", got)
	}
}

func TestZapfGlyphNamesAreFontSpecific(t *testing.T) {
	if _, ok := glyphToUnicode("a1", false); ok {
		t.Fatal("Zapf name enabled for ordinary font")
	}
	e := NewCustomEncodingFromGlyphs(ZapfDingbatsEncoding, map[byte]string{'A': "a1"})
	if got := e.DecodeString([]byte("A")); got != "✁" {
		t.Fatalf("Zapf decode=%q", got)
	}
	f := NewFont("F1", "ABCDEF+ZapfDingbats", "Type1")
	if err := f.applyEncodingDifferences(core.Array{core.Int(65), core.Name("a1")}); err != nil {
		t.Fatal(err)
	}
	if got := f.DecodeString([]byte("A")); got != "✁" {
		t.Fatalf("font context lost: %q", got)
	}
}

func TestToUnicodeTakesPriorityOverGlyphNames(t *testing.T) {
	f := NewFont("F1", "Test", "Type1")
	if err := f.applyEncodingDifferences(core.Array{core.Int(65), core.Name("f_f_i"), core.Name("theta")}); err != nil {
		t.Fatal(err)
	}
	cmap := NewCMap()
	cmap.charMappings[65] = "X"
	f.ToUnicodeCMap = cmap
	if got := f.DecodeString([]byte("AB")); got != "Xθ" {
		t.Fatalf("mapping priority=%q", got)
	}
}

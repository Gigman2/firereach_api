package domain

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type hashFixture struct {
	Name  string `json:"name"`
	Input struct {
		Slug    string   `json:"slug"`
		Title   string   `json:"title"`
		Summary string   `json:"summary"`
		Body    string   `json:"body"`
		Steps   []Step   `json:"steps"`
		Sources []Source `json:"sources"`
	} `json:"input"`
	Expected string `json:"expected"`
}

func loadHashFixtures(t *testing.T) []hashFixture {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", "content-hash-fixtures.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixtures: %v", err)
	}

	var fixtures []hashFixture
	if err := json.Unmarshal(raw, &fixtures); err != nil {
		t.Fatalf("parse fixtures: %v", err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no fixtures loaded")
	}

	return fixtures
}

func TestContentHashMatchesSharedFixtures(t *testing.T) {
	fixtures := loadHashFixtures(t)

	for _, f := range fixtures {
		t.Run(f.Name, func(t *testing.T) {
			got := ContentHash(
				f.Input.Slug, f.Input.Title, f.Input.Summary, f.Input.Body,
				f.Input.Steps, f.Input.Sources,
			)
			if got != f.Expected {
				t.Errorf("hash mismatch\n  got:  %s\n  want: %s", got, f.Expected)
			}
		})
	}
}

// U+0085 is unicode.IsSpace in Go but not \s in JavaScript; U+FEFF is the
// reverse. Both must survive as content or the two implementations diverge.
func TestContentHashPreservesNonASCIISpace(t *testing.T) {
	h := func(title string) string {
		return ContentHash("", title, "", "", nil, nil)
	}

	withSpace := h("a b")
	if h("a\u0085b") == withSpace {
		t.Error("U+0085 was collapsed as whitespace; it must be preserved as content")
	}
	if h("a\ufeffb") == withSpace {
		t.Error("U+FEFF was collapsed as whitespace; it must be preserved as content")
	}
}

// TestUnicodeFixturesAreGenuinelyDistinct guards the "unicode-decomposed" and
// "unicode-composed" fixtures against ever again being written as the same
// bytes. Those two fixtures only prove NFC normalization works if they
// differ byte-wise going in and still match hashes coming out. A prior
// file-writing round-trip once silently normalized the decomposed title into
// the composed form, which made the equivalence assertion vacuous: the whole
// suite still passed even with the norm.NFC.String call deleted from
// contenthash.go.
func TestUnicodeFixturesAreGenuinelyDistinct(t *testing.T) {
	fixtures := loadHashFixtures(t)

	var decomposed, composed *hashFixture
	for i := range fixtures {
		switch fixtures[i].Name {
		case "unicode-decomposed":
			decomposed = &fixtures[i]
		case "unicode-composed":
			composed = &fixtures[i]
		}
	}
	if decomposed == nil || composed == nil {
		t.Fatal("expected both unicode-decomposed and unicode-composed fixtures to be present")
	}

	if decomposed.Input.Title == composed.Input.Title {
		t.Fatal("fixtures are byte-identical; the NFC equivalence test proves nothing")
	}

	got := ContentHash("", decomposed.Input.Title, "", "", nil, nil)
	want := ContentHash("", composed.Input.Title, "", "", nil, nil)
	if got != want {
		t.Error("NFC normalization is not collapsing the two forms")
	}
}

package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"golang.org/x/text/unicode/norm"
)

const hashFieldSeparator = "\x1f"

// isHashSpace reports the closed ASCII set the content hash treats as
// whitespace.
//
// Deliberately NOT unicode.IsSpace. That matches U+0085 (NEL), which
// JavaScript's \s does not, while JavaScript's \s matches U+FEFF, which
// unicode.IsSpace does not. Either divergence would make the Node build
// script and this function disagree, which silently demotes every reviewed
// item to "awaiting review" forever. Any other space-like rune is content.
func isHashSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ':
		return true
	default:
		return false
	}
}

// normalizeHashField applies NFC, collapses runs of ASCII whitespace to a
// single space, and trims leading and trailing spaces. Collapsing before
// trimming means an all-whitespace field normalizes to "" rather than " ".
func normalizeHashField(s string) string {
	s = norm.NFC.String(s)

	var b strings.Builder
	b.Grow(len(s))
	pendingSpace := false

	for _, r := range s {
		if isHashSpace(r) {
			pendingSpace = true
			continue
		}
		if pendingSpace && b.Len() > 0 {
			b.WriteByte(' ')
		}
		pendingSpace = false
		b.WriteRune(r)
	}

	return b.String()
}

// ContentHash digests the reviewer-visible prose of an item: its slug, title,
// summary, body, ordered steps, and ordered sources.
//
// Tags, contextual_trigger, category, subcategory, and the review fields are
// excluded on purpose. The hash answers exactly one question — is this still
// the text the reviewer read? — so retagging an item or moving where it
// surfaces must not invalidate a clinical sign-off.
//
// Takes explicit fields rather than a SafetyContent so it stays a pure
// function of exactly what it hashes.
func ContentHash(slug, title, summary, body string, steps []Step, sources []Source) string {
	fields := make([]string, 0, 4+len(steps)*2+len(sources)*4)
	fields = append(fields, slug, title, summary, body)

	for _, s := range steps {
		fields = append(fields, s.Title, s.Body)
	}
	for _, s := range sources {
		fields = append(fields, s.Title, s.Publisher, s.Year, s.URL)
	}

	for i, f := range fields {
		fields[i] = normalizeHashField(f)
	}

	sum := sha256.Sum256([]byte(strings.Join(fields, hashFieldSeparator)))
	return hex.EncodeToString(sum[:])
}

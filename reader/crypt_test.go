package reader

import (
	"strings"
	"testing"

	"github.com/tsawler/tabula/core"
)

// TestDecryptEmptyPassword verifies the standard security handler decrypts
// content for every method qpdf emits with an empty user/owner password.
// Fixtures are a one-page "Encrypted Hello World" PDF encrypted with qpdf.
func TestDecryptEmptyPassword(t *testing.T) {
	for _, name := range []string{"rc4_40", "rc4_128", "aesv2", "aesv3"} {
		t.Run(name, func(t *testing.T) {
			r, err := Open("testdata/encrypted/" + name + ".pdf")
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			defer r.Close()
			if r.security == nil {
				t.Fatal("expected an active security handler")
			}
			if got := pageContentText(t, r); !strings.Contains(got, "Encrypted Hello World") {
				t.Errorf("decrypted content missing expected text; got %q", got)
			}
		})
	}
}

// TestUnencryptedHasNoSecurity confirms a plain PDF needs no decryption.
func TestUnencryptedHasNoSecurity(t *testing.T) {
	r, err := Open("testdata/encrypted/plain.pdf")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer r.Close()
	if r.security != nil {
		t.Fatal("plain PDF should have no security handler")
	}
	if got := pageContentText(t, r); !strings.Contains(got, "Encrypted Hello World") {
		t.Errorf("content missing expected text; got %q", got)
	}
}

// pageContentText returns the decoded content stream(s) of the first page.
func pageContentText(t *testing.T, r *Reader) string {
	t.Helper()
	page, err := r.GetPage(0)
	if err != nil {
		t.Fatalf("GetPage(0): %v", err)
	}
	contents, err := page.Contents()
	if err != nil {
		t.Fatalf("Contents: %v", err)
	}
	var b strings.Builder
	for _, obj := range contents {
		stream, ok := obj.(*core.Stream)
		if !ok {
			continue
		}
		data, err := stream.Decode()
		if err != nil {
			t.Fatalf("decode content stream: %v", err)
		}
		b.Write(data)
	}
	return b.String()
}

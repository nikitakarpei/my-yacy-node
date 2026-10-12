package articlefootnotes_test

import (
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/articlefootnotes"
)

func renderedNotesOf(t *testing.T, article string) []string {
	t.Helper()
	root, err := html.Parse(strings.NewReader(article))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	found := articlefootnotes.NotesOf(root)
	notes := make([]string, 0, len(found))
	for _, note := range found {
		var rendered strings.Builder
		if err := html.Render(&rendered, note); err != nil {
			t.Fatalf("render: %v", err)
		}
		notes = append(notes, rendered.String())
	}
	return notes
}

func TestNotesOfFindsTheNoteAMarkerLinksTo(t *testing.T) {
	notes := renderedNotesOf(t,
		`<p>TCP.<sup id="ref-1"><a href="#note-1">[1]</a></sup></p>`+
			`<ol><li id="note-1"><a href="#ref-1">↑</a> Stevens.</li><li id="other">Index</li></ol>`)
	if len(notes) != 1 || notes[0] != `<li id="note-1"><a href="#ref-1">↑</a> Stevens.</li>` {
		t.Fatalf("notes = %q", notes)
	}
}

func TestNotesOfFindsTheNoteOfALinkHoldingASup(t *testing.T) {
	notes := renderedNotesOf(t,
		`<p>TCP.<a href="#fn1"><sup>1</sup></a></p><p id="fn1">Stevens.</p>`)
	if len(notes) != 1 || notes[0] != `<p id="fn1">Stevens.</p>` {
		t.Fatalf("notes = %q", notes)
	}
}

func TestNotesOfKeepsOutAnElementThatHoldsItsOwnMarker(t *testing.T) {
	notes := renderedNotesOf(t,
		`<div id="top"><p>TCP.<sup><a href="#top">↑</a></sup></p></div>`)
	if len(notes) != 0 {
		t.Fatalf("an element holding its own marker became a note: %q", notes)
	}
}

func TestNotesOfKeepsOutAMarkerThatAnotherMarkerLinksTo(t *testing.T) {
	notes := renderedNotesOf(
		t,
		`<p>TCP.<sup><a href="#ref-2">[1]</a></sup><sup id="ref-2"><a href="#note-2">[2]</a></sup></p>`+
			`<ol><li id="note-2">Postel.</li></ol>`,
	)
	if len(notes) != 1 || notes[0] != `<li id="note-2">Postel.</li>` {
		t.Fatalf("notes = %q", notes)
	}
}

func TestNotesOfKeepsOutALinkBackToANote(t *testing.T) {
	notes := renderedNotesOf(t,
		`<p>TCP.<a href="#note-1" id="ref-1">0</a></p>`+
			`<p><sup><a href="#ref-1" id="note-1">0</a></sup>Stevens.</p>`)
	if len(notes) != 0 {
		t.Fatalf("a link back to a note became a note: %q", notes)
	}
}

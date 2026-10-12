package articlefootnotes

import (
	"golang.org/x/net/html"
)

func NotesOf(article *html.Node) []*html.Node {
	markersPerNoteID := markersPerNoteIDOf(article)
	marks := footnoteMarksOf(article)
	var notes []*html.Node
	for node := range article.Descendants() {
		markers, linkedTo := markersPerNoteID[attributeOf(node, "id")]
		if linkedTo && !holdsAnyOf(node, markers) && !marks.holdOrContain(node) {
			notes = append(notes, node)
		}
	}
	return notes
}

func markersPerNoteIDOf(article *html.Node) map[string][]*html.Node {
	markersPerNoteID := map[string][]*html.Node{}
	for node := range article.Descendants() {
		if !isFootnoteMarker(node) {
			continue
		}
		for _, link := range linksOf(node) {
			noteID := targetIDOf(link)
			markersPerNoteID[noteID] = append(markersPerNoteID[noteID], node)
		}
	}
	return markersPerNoteID
}

func linksOf(marker *html.Node) []*html.Node {
	if isSamePageLink(marker) {
		return []*html.Node{marker}
	}
	var links []*html.Node
	for node := range marker.Descendants() {
		if isSamePageLink(node) {
			links = append(links, node)
		}
	}
	return links
}

func holdsAnyOf(container *html.Node, nodes []*html.Node) bool {
	for _, node := range nodes {
		if node == container {
			return true
		}
		for ancestor := range node.Ancestors() {
			if ancestor == container {
				return true
			}
		}
	}
	return false
}

package render

import (
	"io"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// HTML walks a parsed HTML tree and emits a human-readable plain-text
// rendering: tags are stripped, links are formatted as `text [href]`,
// block elements introduce paragraph breaks, and runs of whitespace are
// collapsed.
func HTML(r io.Reader) string {
	doc, err := html.Parse(r)
	if err != nil {
		return ""
	}

	var b strings.Builder
	walk(doc, &b)

	out := b.String()
	out = inlineWS.ReplaceAllString(out, " ")
	out = blankLines.ReplaceAllString(out, "\n\n")
	return strings.TrimSpace(out)
}

var (
	inlineWS   = regexp.MustCompile(`[ \t]+`)
	blankLines = regexp.MustCompile(`\n[ \t]*\n[\s\n]*`)
)

var blockElements = map[atom.Atom]bool{
	atom.P: true, atom.Div: true, atom.Br: true, atom.Hr: true,
	atom.H1: true, atom.H2: true, atom.H3: true,
	atom.H4: true, atom.H5: true, atom.H6: true,
	atom.Li: true, atom.Tr: true, atom.Ul: true, atom.Ol: true,
	atom.Article: true, atom.Section: true, atom.Header: true,
	atom.Footer: true, atom.Nav: true, atom.Main: true, atom.Aside: true,
	atom.Pre: true, atom.Blockquote: true, atom.Table: true,
}

var skipElements = map[atom.Atom]bool{
	atom.Script: true, atom.Style: true, atom.Noscript: true,
	atom.Iframe: true, atom.Svg: true, atom.Object: true,
	atom.Embed: true, atom.Canvas: true,
}

func walk(n *html.Node, b *strings.Builder) {
	switch n.Type {
	case html.TextNode:
		b.WriteString(n.Data)
		return
	case html.ElementNode:
		if skipElements[n.DataAtom] {
			return
		}
		switch n.DataAtom {
		case atom.A:
			text := strings.TrimSpace(extractText(n))
			href := getAttr(n, "href")
			b.WriteString(text)
			if href != "" && href != text && !strings.HasPrefix(href, "#") {
				b.WriteString(" [")
				b.WriteString(href)
				b.WriteString("]")
			}
			return
		case atom.Title:
			b.WriteString("\n# ")
			b.WriteString(strings.TrimSpace(extractText(n)))
			b.WriteString("\n\n")
			return
		case atom.Img:
			alt := getAttr(n, "alt")
			if alt != "" {
				b.WriteString("[image: ")
				b.WriteString(alt)
				b.WriteString("]")
			}
			return
		}
		if blockElements[n.DataAtom] {
			b.WriteString("\n")
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, b)
	}

	if n.Type == html.ElementNode && blockElements[n.DataAtom] {
		b.WriteString("\n")
	}
}

func extractText(n *html.Node) string {
	var b strings.Builder
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return b.String()
}

func getAttr(n *html.Node, name string) string {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val
		}
	}
	return ""
}

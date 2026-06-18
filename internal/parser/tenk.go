package parser

import (
	"bytes"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

type Section struct {
	ID    string // e.g. "item_1", "item_1a", "item_7"
	Title string // e.g. "Item 1. Business"
	Text  string // extracted plain text
}

var itemPattern = regexp.MustCompile(`(?i)^item\s+(\d+[a-z]?)\.?\s`)

func ParseTenK(data []byte) ([]Section, error) {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var sections []Section
	var current *Section
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && isHeading(n.Data) {
			text := strings.TrimSpace(extractText(n))
			if m := itemPattern.FindStringSubmatch(text); m != nil {
				if current != nil {
					sections = append(sections, *current)
				}
				id := "item_" + strings.ToLower(m[1])
				current = &Section{ID: id, Title: text}
			}
		} else if n.Type == html.TextNode && current != nil {
			t := strings.TrimSpace(n.Data)
			if t != "" {
				current.Text += t + " "
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if current != nil {
		sections = append(sections, *current)
	}

	for i := range sections {
		sections[i].Text = strings.TrimSpace(sections[i].Text)
	}
	return sections, nil
}

func isHeading(tag string) bool {
	return tag == "h1" || tag == "h2" || tag == "h3" || tag == "h4"
}

func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(extractText(c))
	}
	return sb.String()
}

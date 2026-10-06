package scxml

import (
	"encoding/xml"
	"errors"
	"io"
	"strings"
)

// node mirrors the xml-js `Element` shape the JS converter walks: an element
// with a name, attributes and child nodes, or a non-blank text node.
// Comments are dropped: every place the JS converter visits children either
// filters by element name or skips `type === 'comment'`.
type node struct {
	text     bool
	name     string
	attrs    map[string]string
	children []*node
}

// attr mirrors `element.attributes[name]`; ok reports presence
// (`name in element.attributes`).
func (n *node) attr(name string) (string, bool) {
	v, ok := n.attrs[name]
	return v, ok
}

// childrenNamed mirrors `elements.filter((el) => el.name === name)`.
func (n *node) childrenNamed(names ...string) []*node {
	var out []*node
	for _, c := range n.children {
		for _, name := range names {
			if !c.text && c.name == name {
				out = append(out, c)
				break
			}
		}
	}
	return out
}

// parseXML mirrors xml2js(xml): it returns the document node whose children
// are the top-level elements.
func parseXML(src string) (*node, error) {
	dec := xml.NewDecoder(strings.NewReader(src))
	doc := &node{}
	stack := []*node{doc}
	for {
		// RawToken keeps namespace prefixes literal ("conf:pass"), as
		// xml-js names elements and attributes.
		tok, err := dec.RawToken()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		parent := stack[len(stack)-1]
		switch t := tok.(type) {
		case xml.StartElement:
			el := &node{name: qualifiedName(t.Name), attrs: map[string]string{}}
			for _, a := range t.Attr {
				el.attrs[qualifiedName(a.Name)] = a.Value
			}
			parent.children = append(parent.children, el)
			stack = append(stack, el)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			// xml-js drops whitespace-only text between elements.
			if strings.TrimSpace(string(t)) != "" {
				parent.children = append(parent.children, &node{text: true})
			}
		}
	}
	return doc, nil
}

func qualifiedName(n xml.Name) string {
	if n.Space == "" {
		return n.Local
	}
	return n.Space + ":" + n.Local
}

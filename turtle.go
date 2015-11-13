package turtle

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	iriURL    = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"
	stopRunes = " \n\t,;."
)

type Triple struct {
	Subj, Pred, Obj, Type, Lang string
}

func Parse(in string) ([]Triple, error) {
	p := &parser{
		body:   in,
		prefix: make(map[string]string),
	}
	if err := p.parse(); err != nil {
		return nil, err
	}
	return p.triples, nil
}

type parser struct {
	triples []Triple
	base    string
	prefix  map[string]string
	body    string
	i       int
}

func (p *parser) parse() error {
	for p.i < len(p.body) {
		lastI := p.i
		if err := p.parseExpr(); err != nil {
			return err
		}
		if p.i == lastI {
			return fmt.Errorf("parser stuck in infinite loop %#v", p.body[p.i:])
		}
	}
	return nil
}

func (p *parser) parseExpr() error {
	p.skipWhitespace()
	c := p.c()
	if len(c) == 0 {
		return nil
	}
	if strings.HasPrefix(c, "@base") {
		p.i += 6
		url, typ, _ := p.parseObj()
		if typ != "iri" {
			return fmt.Errorf("@base expected IRI. Found %#v", url)
		}
		p.base = url
		t, typ, _ := p.parseObj()
		if typ != "end" {
			return fmt.Errorf("@base should only have one argument. Found %#v", t)
		}
	} else if strings.HasPrefix(c, "@prefix") {
		p.i += 8
		prefix, typ, _ := p.parseObj()
		if typ != "prefix" {
			return fmt.Errorf("@prefix expected prefix. Found %#v", prefix)
		}
		url, typ, _ := p.parseObj()
		if typ != "iri" {
			return fmt.Errorf("@prefix expected IRI. Found %#v", url)
		}
		p.prefix[prefix] = url
		t, typ, _ := p.parseObj()
		if typ != "end" {
			return fmt.Errorf("@prefix should only have two arguments. Found %#v", t)
		}
	} else {
		subj, typ, _ := p.parseObj()
		if typ != "iri" {
			return fmt.Errorf("triple subject needs to be iri. Found %#v", subj)
		}
	Pred:
		for {
			pred, typ, _ := p.parseObj()
			if pred == "a" {
				pred = iriURL
			} else if typ != "iri" {
				return fmt.Errorf("triple predicate needs to be iri. Found %#v", pred)
			}
			for {
				obj, typ, lang := p.parseObj()
				if typ == "end" || typ == "pred" || typ == "obj" {
					return fmt.Errorf("triple needs subject. Found %#v", obj)
				}
				p.triples = append(p.triples, Triple{subj, pred, obj, "", lang})
				ctrl, typ, _ := p.parseObj()
				switch typ {
				case "end":
					break Pred
				case "pred":
					continue Pred
				case "obj":
					continue
				default:
					return fmt.Errorf("triple expected control character. Found %#v", ctrl)
				}
			}
		}
	}
	return nil
}

// parseObject returns body, type, lang
func (p *parser) parseObj() (string, string, string) {
	p.skipWhitespace()
	c := p.c()
	if c[0] == '"' || c[0] == '\'' {
		i := 1
	For:
		for ; i < len(c); i++ {
			switch c[i] {
			case '\\':
				i++
			case c[0]:
				break For
			}
		}
		p.i += i + 1

		s := "\"" + c[1:i] + "\""
		json.Unmarshal([]byte(s), &s)
		lang := ""
		if c[i+1] == '@' {
			c = p.c()
			i := strings.IndexAny(c, stopRunes)
			p.i += i
			lang = c[1:i]
		} else if c[i+1:i+3] == "^^" {
			// TODO(d4l3k): Implement RDF types.
		}
		return s, "string", lang
	} else if strings.HasPrefix(c, "<") {
		i := strings.IndexRune(c, '>')
		p.i += i + 1
		url := c[1:i]
		if !strings.Contains(url, "://") {
			url = p.base + url
		}
		return url, "iri", ""
	} else if strings.HasPrefix(c, ".") {
		p.i += 1
		return ".", "end", ""
	} else if strings.HasPrefix(c, ";") {
		p.i += 1
		return ";", "pred", ""
	} else if strings.HasPrefix(c, ",") {
		p.i += 1
		return ",", "obj", ""
	} else {
		i := strings.IndexAny(c, stopRunes)
		p.i += i
		obj := c[:i]
		psym := strings.IndexRune(obj, ':')
		if psym != -1 {
			if psym == len(obj)-1 {
				return obj, "prefix", ""
			}
			prefix := obj[:psym+1]
			if expand, ok := p.prefix[prefix]; ok {
				return expand + obj[psym+1:], "iri", ""
			}
		}
		return obj, "", ""
	}
}

func (p *parser) c() string {
	return p.body[p.i:]
}

func (p *parser) skipWhitespace() {
For:
	for p.i < len(p.body) {
		switch p.body[p.i] {
		case ' ', '\n', '\t':
			p.i++
		case '#':
			i := strings.IndexRune(p.c(), '\n')
			if i == -1 {
				p.i = len(p.body)
			} else {
				p.i += i
			}
		default:
			break For
		}
	}
}

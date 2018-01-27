package turtle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	iriURL    = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"
	stopRunes = " \n\t,;"
)

type objType int

const (
	typeUnknown objType = iota
	typeString
	typeIRI
	typeEnd
	typePred
	typeObj
	typePrefix
	typeBag
	typeBagEnd
)

type Triple struct {
	Subj, Pred, Obj, Type, Lang string
}

func Parse(in []byte) ([]Triple, error) {
	p := &parser{
		body: in,
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
	body    []byte
	i       int
}

func (p *parser) parse() error {
	p.prefix = make(map[string]string)
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
	if bytes.HasPrefix(c, []byte("@base")) {
		p.i += 6
		url, typ, _ := p.parseObj()
		if typ != typeIRI {
			return fmt.Errorf("@base expected IRI. Found %#v", url)
		}
		p.base = url
		t, typ, _ := p.parseObj()
		if typ != typeEnd {
			return fmt.Errorf("@base should only have one argument. Found %#v", t)
		}
	} else if bytes.HasPrefix(c, []byte("@prefix")) {
		p.i += 8
		prefix, typ, _ := p.parseObj()
		if typ != typePrefix {
			return fmt.Errorf("@prefix expected prefix. Found %#v", prefix)
		}
		url, typ, _ := p.parseObj()
		if typ != typeIRI {
			return fmt.Errorf("@prefix expected IRI. Found %#v", url)
		}
		p.prefix[prefix] = url
		t, typ, _ := p.parseObj()
		if typ != typeEnd {
			return fmt.Errorf("@prefix should only have two arguments. Found %#v", t)
		}
	} else {
		subj, typ, _ := p.parseObj()
		if typ != typeIRI {
			return fmt.Errorf("triple subject needs to be iri. Found %#v", subj)
		}
		var origSubj string
	Pred:
		for {
			pred, typ, _ := p.parseObj()
			if pred == "a" {
				pred = iriURL
			} else if typ != typeIRI {
				return fmt.Errorf("triple predicate needs to be iri. Found %#v", pred)
			}
			for {
				obj, typ, lang := p.parseObj()
				if typ == typeBag {
					p.skipWhitespace()

					blankNodeId := "_:" + subj + "_" + pred
					// add triple of subject -> pred -> <blank node string>
					p.triples = append(p.triples, Triple{subj, pred, blankNodeId, "", lang})
					// set that string as subject
					origSubj = subj
					subj = blankNodeId
					break // start processing the children in the bag
				}

				if typ == typeEnd || typ == typePred || typ == typeObj || typ == typeUnknown {
					return fmt.Errorf("triple needs subject. Found %#v with type %d", obj, typ)
				}
				p.triples = append(p.triples, Triple{subj, pred, obj, "", lang})
				ctrl, typ, _ := p.parseObj()

				switch typ {
				case typeEnd:
					break Pred
				case typePred:
					continue Pred
				case typeObj:
					continue
				case typeBagEnd:
					subj = origSubj
					if ctrl == "." {
						break Pred
					} else {
						continue Pred
					}
				default:
					return fmt.Errorf("triple expected control character. Found %#v with type %d at %d", ctrl, typ, p.i)
				}
			}
		}
	}
	return nil
}

// parseObject returns body, type, lang
func (p *parser) parseObj() (string, objType, string) {
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

		s := "\"" + string(c[1:i]) + "\""
		json.Unmarshal([]byte(s), &s)
		lang := ""
		if c[i+1] == '@' {
			c = p.c()
			i := bytes.IndexAny(c, stopRunes)
			if c[i-1] == '.' {
				i--
			}
			p.i += i
			lang = string(c[1:i])
		} else if bytes.Equal(c[i+1:i+3], []byte("^^")) {
			// TODO(d4l3k): Implement RDF types.
		}
		return s, typeString, lang
	} else if bytes.HasPrefix(c, []byte("<")) {
		i := bytes.IndexRune(c, '>')
		p.i += i + 1
		url := string(c[1:i])
		if !strings.Contains(url, "://") {
			url = p.base + url
		}
		return url, typeIRI, ""
	} else if bytes.HasPrefix(c, []byte(".")) {
		p.i += 1
		return ".", typeEnd, ""
	} else if bytes.HasPrefix(c, []byte(";")) {
		p.i += 1
		return ";", typePred, ""
	} else if bytes.HasPrefix(c, []byte(",")) {
		p.i += 1
		return ",", typeObj, ""
	} else if bytes.HasPrefix(c, []byte("];")) {
		p.i += 2
		return ";", typeBagEnd, ""
	} else if bytes.HasPrefix(c, []byte("].")) {
		p.i += 2
		return ".", typeBagEnd, ""
	} else {
		i := bytes.IndexAny(c, stopRunes)
		if c[i-1] == '.' {
			i--
		}
		p.i += i
		obj := string(c[:i])
		psym := strings.IndexRune(obj, ':')
		bagSym := strings.IndexRune(obj, '[')
		if psym != -1 {
			if psym == len(obj)-1 {
				return obj, typePrefix, ""
			}
			prefix := obj[:psym+1]
			if expand, ok := p.prefix[prefix]; ok {
				return expand + obj[psym+1:], typeIRI, ""
			}
		}
		if bagSym != -1 {
			p.i += bagSym
			return "", typeBag, ""
		}
		return obj, typeUnknown, ""
	}
}

func (p *parser) c() []byte {
	return p.body[p.i:]
}

func (p *parser) skipWhitespace() {
For:
	for p.i < len(p.body) {
		switch p.body[p.i] {
		case ' ', '\n', '\t':
			p.i++
		case '#':
			i := bytes.IndexRune(p.c(), '\n')
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

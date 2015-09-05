package turtle

import "strings"

type Triple struct {
	Subj, Pred, Obj string
}

func Parse(in string) []*Triple {
	var triples []*Triple
	prefixes := make(map[string]string)
	switch {
	case strings.HasPrefix(in, "@prefix "):
		in = parsePrefix(prefixes, in)
	}
	return triples
}

func parsePrefix(prefixes map[string]string, in string) string {
	in = in[8:]
	bits := strings.SplitN(in, ": ", 1)
	prefix := bits[0]
	in = bits[1]
	var body string
	body, in = parseVal(prefixes, in)
	prefixes[prefix] = body
	return in[1:]
}

func parseURL(in string) (string, string) {
	in = in[1:]
	i := strings.Index(in, ">")
	if i == -1 {
		return in, ""
	}
	return in[:i], in[i+1:]
}

func parseVal(prefixes map[string]string, in string) (string, string) {
	if in[0] == '<' {
		return parseURL(in)
	}
	i := strings.IndexAny(in, " ;.\n")
	raw := in[:i]
	left := in[i+1:]
	if j := strings.Index(raw, ":"); j >= 0 {
		return prefixes[raw[:j]] + raw[j+1:], left
	}
	return raw, left
}

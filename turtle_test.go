package turtle

import (
	"io/ioutil"
	"testing"

	"github.com/d4l3k/messagediff"
)

func TestParse(t *testing.T) {
	testData := []struct {
		file string
		want []Triple
		diff bool
	}{
		{
			"testdata/example.turtle",
			[]Triple{
				{
					"http://example.org/#green-goblin",
					"http://www.perceive.net/schemas/relationship/enemyOf",
					"http://example.org/#spiderman",
					"", "",
				},
				{
					"http://example.org/#green-goblin",
					"http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
					"http://xmlns.com/foaf/0.1/Person",
					"", "",
				},
				{
					"http://example.org/#green-goblin",
					"http://xmlns.com/foaf/0.1/name",
					"Green Goblin",
					"", "",
				},
				{
					"http://example.org/#spiderman",
					"http://www.perceive.net/schemas/relationship/enemyOf",
					"http://example.org/#green-goblin",
					"", "",
				},
				{
					"http://example.org/#spiderman",
					"http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
					"http://xmlns.com/foaf/0.1/Person",
					"", "",
				},
				{
					"http://example.org/#spiderman",
					"http://xmlns.com/foaf/0.1/name",
					"Spiderman \"Wow\"",
					"", "",
				},
				{
					"http://example.org/#spiderman",
					"http://xmlns.com/foaf/0.1/name",
					"Человек-паук",
					"", "ru",
				},
			},
			true,
		},
		/*{
			"testdata/02mjmr.turtle",
			nil,
			false,
		},*/
	}
	for i, td := range testData {
		rdf, err := ioutil.ReadFile(td.file)
		if err != nil {
			t.Fatal(err)
		}
		out, err := Parse(rdf)
		if err != nil {
			t.Fatal(err)
		}
		if !td.diff {
			continue
		}
		if diff, equal := messagediff.PrettyDiff(td.want, out); !equal {
			t.Errorf("%d. Parse(%s) = %#v; diff %s", i, td.file, out, diff)
		}

	}
}

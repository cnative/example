package e2e

import (
	"testing"

	"github.com/tidwall/gjson"
)

//Matcher result matcher
type Matcher struct {
	mustEqual   map[string]string //jsonpath:value
	mustHave    []string          //jsonpath
	mustContain map[string]string //jsonpath key
}

// Match match json
func (m *Matcher) Match(t *testing.T, json string) {
	t.Helper()

	if json == "" {
		t.Error("got empty string, expected json response")
	}

	for _, path := range m.mustHave {
		result := gjson.Get(json, path)

		if !result.Exists() {
			t.Errorf("got empty/nil for %s, expect non empty/nil", path)
		}
	}

	for path, expect := range m.mustEqual {
		result := gjson.Get(json, path)
		if result.Exists() {
			val := result.String()
			if val != expect {
				t.Errorf("have %v, expect %v", val, expect)
			}
		} else {
			t.Errorf("got empty/nil for %s, expect %s", expect, path)
		}
	}

	for path, expect := range m.mustContain {
		result := gjson.Get(json, path)
		if result.Exists() {
			if !result.IsArray() {
				t.Errorf("have %v, expect array of strings", result.Type)
			}

			vals := result.Array()
			if !contains(vals, expect) {
				t.Errorf("have %v, expect %v", vals, expect)
			}
		} else {
			t.Errorf("got empty/nil for %s, expect %s", expect, path)
		}
	}
}

func contains(s []gjson.Result, val string) bool {
	for _, a := range s {
		if a.String() == val {
			return true
		}
	}
	return false
}

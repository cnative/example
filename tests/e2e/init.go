package e2e

import (
	"strings"

	"github.com/tidwall/gjson"
)

func init() {
	gjson.AddModifier("case", func(json, arg string) string {
		if arg == "upper" {
			return strings.ToUpper(json)
		}
		if arg == "lower" {
			return strings.ToLower(json)
		}
		return json
	})
}

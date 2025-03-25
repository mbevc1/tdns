package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/fatih/color"
)

// Define color styles
var (
	grey   = color.New(color.FgHiBlack).SprintFunc()
	green  = color.New(color.FgHiGreen).SprintFunc()
	greenL = color.New(color.FgGreen).SprintFunc()
	blue   = color.New(color.FgHiBlue).SprintFunc()
	yellow = color.New(color.FgHiYellow).SprintFunc()
	bold   = color.New(color.Bold).SprintFunc()
	cyan   = color.New(color.FgCyan).SprintFunc()
)

func FormatMap(input interface{}) string {
	return formatValue(input)
}

// Recursive formatter for value
func formatValue(val interface{}) string {
	switch v := val.(type) {
	case map[string]interface{}:
		var sb strings.Builder
		sb.WriteString("[")

		// Collect and sort keys
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		// Use sorted keys to build string
		for i, k := range keys {
			innerVal := v[k]
			sb.WriteString(fmt.Sprintf("%s:%s", grey(k), formatValue(innerVal)))
			if i < len(keys)-1 {
				sb.WriteString(", ")
			}
		}

		sb.WriteString("]")
		return sb.String()

	case string:
		return green(v)

	case float64:
		return blue(fmt.Sprintf("%.0f", v))

	case int, int64:
		return blue(fmt.Sprintf("%v", v))

	case bool:
		return yellow(fmt.Sprintf("%v", v))

	default:
		return fmt.Sprintf("%v", v) // no color fallback
	}
}

// Package output is the single output contract: data to stdout, human chatter to stderr,
// stable JSON, --format json|plain|tsv, --select projection, and --limit bounding.
// See contract.md §1 and §6.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
)

type Format string

const (
	FormatJSON  Format = "json"
	FormatPlain Format = "plain"
	FormatTSV   Format = "tsv"
)

// Writer routes all command output. Data goes to Stdout; Info/Warn go to Stderr.
type Writer struct {
	Stdout io.Writer
	Stderr io.Writer
	Format Format
	Color  bool
	Limit  int
	Select []string
}

// Info writes a human-facing message to stderr (keeps stdout parseable).
func (w *Writer) Info(format string, a ...any) {
	fmt.Fprintf(w.Stderr, format+"\n", a...)
}

// Emit renders a value to stdout per the active format, after applying --select and --limit.
func (w *Writer) Emit(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var g any
	if err := json.Unmarshal(b, &g); err != nil {
		return err
	}
	if len(w.Select) > 0 {
		g = applySelect(g, w.Select)
	}
	g = w.applyLimit(g)

	switch w.Format {
	case FormatJSON:
		return w.encodeJSON(g)
	case FormatTSV:
		return w.renderDelimited(g, "\t", false)
	default:
		return w.renderDelimited(g, "\t", true)
	}
}

// EmitJSON forces JSON output regardless of --format (used by `schema`).
func (w *Writer) EmitJSON(v any) error { return w.encodeJSON(v) }

func (w *Writer) encodeJSON(v any) error {
	enc := json.NewEncoder(w.Stdout)
	enc.SetEscapeHTML(false) // URLs must survive
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func (w *Writer) applyLimit(g any) any {
	if w.Limit <= 0 {
		return g
	}
	if arr, ok := g.([]any); ok && len(arr) > w.Limit {
		w.Info("note: output truncated to %d of %d items (use --limit to change)", w.Limit, len(arr))
		return arr[:w.Limit]
	}
	return g
}

func applySelect(g any, sel []string) any {
	if arr, ok := g.([]any); ok {
		out := make([]any, 0, len(arr))
		for _, e := range arr {
			out = append(out, selectObj(e, sel))
		}
		return out
	}
	return selectObj(g, sel)
}

func selectObj(e any, sel []string) any {
	m, ok := e.(map[string]any)
	if !ok {
		return e
	}
	out := map[string]any{}
	for _, p := range sel {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if v, ok := getPath(m, p); ok {
			out[p] = v
		}
	}
	return out
}

func getPath(m map[string]any, path string) (any, bool) {
	var cur any = m
	for _, part := range strings.Split(path, ".") {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = mm[part]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

func (w *Writer) renderDelimited(g any, sep string, aligned bool) error {
	out := w.Stdout
	var tw *tabwriter.Writer
	if aligned {
		tw = tabwriter.NewWriter(w.Stdout, 0, 2, 2, ' ', 0)
		out = tw
	}
	switch t := g.(type) {
	case []any:
		if len(t) == 0 {
			break
		}
		if _, ok := t[0].(map[string]any); ok {
			headers := unionKeys(t)
			fmt.Fprintln(out, strings.Join(headers, sep))
			for _, e := range t {
				m, _ := e.(map[string]any)
				row := make([]string, len(headers))
				for i, h := range headers {
					row[i] = scalarString(m[h])
				}
				fmt.Fprintln(out, strings.Join(row, sep))
			}
		} else {
			for _, e := range t {
				fmt.Fprintln(out, scalarString(e))
			}
		}
	case map[string]any:
		for _, k := range sortedKeys(t) {
			fmt.Fprintln(out, k+sep+scalarString(t[k]))
		}
	default:
		fmt.Fprintln(out, scalarString(g))
	}
	if tw != nil {
		return tw.Flush()
	}
	return nil
}

func unionKeys(arr []any) []string {
	seen := map[string]bool{}
	for _, e := range arr {
		if m, ok := e.(map[string]any); ok {
			for k := range m {
				seen[k] = true
			}
		}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func scalarString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return fmt.Sprintf("%v", t)
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

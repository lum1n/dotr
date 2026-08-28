// Package preview renders config files with syntax highlighting.
package preview

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/glamour"
	toml "github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

const (
	maxPreviewBytes = 256 << 10 // 256 KiB
	maxPreviewLines = 400
)

// ParseStatus is the structured-parse result for known formats.
type ParseStatus int

const (
	ParseNone ParseStatus = iota
	ParseOK
	ParseFail
)

// Result is highlighted preview text plus metadata.
type Result struct {
	Path      string
	Language  string
	Content   string
	Binary    bool
	Truncated bool
	Parse     ParseStatus
	ParseErr  string
	Err       error
}

// Render loads path and returns a terminal-highlighted preview.
func Render(path string, width int) Result {
	res := Result{Path: path}
	if width < 20 {
		width = 80
	}

	fi, err := os.Stat(path)
	if err != nil {
		res.Err = err
		res.Content = err.Error()
		return res
	}
	if fi.IsDir() {
		res.Content = "(directory)"
		return res
	}
	if fi.Size() > 2<<20 {
		res.Truncated = true
		res.Content = fmt.Sprintf("file too large for preview (%d bytes)", fi.Size())
		return res
	}

	data, err := os.ReadFile(path)
	if err != nil {
		res.Err = err
		res.Content = err.Error()
		return res
	}
	if int64(len(data)) > maxPreviewBytes {
		data = data[:maxPreviewBytes]
		res.Truncated = true
	}
	if isBinary(data) {
		res.Binary = true
		res.Language = "binary"
		res.Content = fmt.Sprintf("binary file (%d bytes)\n\n%s", fi.Size(), hexDump(data, 16))
		return res
	}

	text := string(data)
	if !utf8.ValidString(text) {
		res.Binary = true
		res.Content = "non-utf8 text; refusing to preview"
		return res
	}

	parseCheck(&res, path, data)

	lines := strings.Split(text, "\n")
	if len(lines) > maxPreviewLines {
		lines = lines[:maxPreviewLines]
		text = strings.Join(lines, "\n")
		res.Truncated = true
	}

	ext := strings.ToLower(filepath.Ext(path))
	base := strings.ToLower(filepath.Base(path))
	if ext == ".md" || ext == ".markdown" {
		rendered, err := renderMarkdown(text, width)
		if err != nil {
			res.Language = "markdown"
			res.Content = text
			return res
		}
		res.Language = "markdown"
		res.Content = rendered
		return res
	}

	lexer := lexers.Match(base)
	if lexer == nil {
		lexer = lexers.Match(path)
	}
	if lexer == nil {
		lexer = lexers.Analyse(text)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)
	res.Language = lexer.Config().Name

	style := styles.Get(StyleName)
	if style == nil {
		style = styles.Get("dracula")
	}
	if style == nil {
		style = styles.Fallback
	}
	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, text)
	if err != nil {
		res.Content = text
		return res
	}
	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		res.Content = text
		return res
	}
	res.Content = buf.String()
	if res.Truncated {
		res.Content += "\n\n… truncated"
	}
	return res
}

func parseCheck(res *Result, path string, data []byte) {
	ext := strings.ToLower(filepath.Ext(path))
	base := strings.ToLower(filepath.Base(path))
	var err error
	switch {
	case ext == ".json" || strings.HasSuffix(base, ".json"):
		if ext == ".jsonc" || strings.HasSuffix(base, ".jsonc") {
			return
		}
		var v any
		err = json.Unmarshal(data, &v)
	case ext == ".yaml" || ext == ".yml":
		var v any
		err = yaml.Unmarshal(data, &v)
	case ext == ".toml":
		var v any
		err = toml.Unmarshal(data, &v)
	default:
		return
	}
	if err != nil {
		res.Parse = ParseFail
		res.ParseErr = err.Error()
		return
	}
	res.Parse = ParseOK
}

// SetStyle overrides the chroma style name used by Render (empty = dracula).
var StyleName = "dracula"

// Badge returns a short parse indicator for the UI.
func (r Result) Badge() string {
	switch r.Parse {
	case ParseOK:
		return "✓"
	case ParseFail:
		return "✗"
	default:
		return ""
	}
}

func renderMarkdown(text string, width int) (string, error) {
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}
	return r.Render(text)
}

func isBinary(data []byte) bool {
	n := len(data)
	if n > 8000 {
		n = 8000
	}
	return bytes.IndexByte(data[:n], 0) >= 0
}

func hexDump(data []byte, rows int) string {
	const rowLen = 16
	var b strings.Builder
	limit := rows * rowLen
	if limit > len(data) {
		limit = len(data)
	}
	for i := 0; i < limit; i += rowLen {
		end := i + rowLen
		if end > limit {
			end = limit
		}
		chunk := data[i:end]
		fmt.Fprintf(&b, "%08x: ", i)
		for j := 0; j < rowLen; j++ {
			if j < len(chunk) {
				fmt.Fprintf(&b, "%02x ", chunk[j])
			} else {
				b.WriteString("   ")
			}
			if j == 7 {
				b.WriteByte(' ')
			}
		}
		b.WriteString(" |")
		for _, c := range chunk {
			if c >= 32 && c < 127 {
				b.WriteByte(c)
			} else {
				b.WriteByte('.')
			}
		}
		b.WriteString("|\n")
	}
	return b.String()
}

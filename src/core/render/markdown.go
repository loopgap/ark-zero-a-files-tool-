package render

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var md goldmark.Markdown
var mermaidBlockPattern = regexp.MustCompile(`(?s)<pre><code class="language-mermaid">(.*?)</code></pre>`)

func init() {
	md = goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Typographer),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithHardWraps(), html.WithXHTML()),
	)
}

const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; line-height: 1.6; padding: 20px; color: #d1d1d1; background: #1e1e1e; }
		pre { background: #0c0c0c; padding: 15px; border-radius: 6px; overflow-x: auto; }
		code { font-family: "JetBrains Mono", monospace; background: #2b2b2b; padding: 2px 4px; border-radius: 3px;}
		pre code { background: transparent; padding: 0; }
		a { color: #58a6ff; text-decoration: none; }
		a:hover { text-decoration: underline; }
		blockquote { border-left: 4px solid #444; margin: 0; padding-left: 15px; color: #888; }
		img { max-width: 100%%; }
		table { border-collapse: collapse; width: 100%%; }
		th, td { border: 1px solid #444; padding: 8px; }
		.mermaid { background: #0c0c0c; padding: 15px; border-radius: 6px; overflow-x: auto; white-space: pre-wrap; }
	</style>
</head>
<body>
%s
</body>
</html>`

// RenderMarkdown compiles markdown to standalone HTML without external runtime dependencies.
func RenderMarkdown(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return RenderMarkdownBytes(path, content)
}

// RenderMarkdownBytes compiles markdown content that is already loaded in memory.
func RenderMarkdownBytes(path string, content []byte) ([]byte, error) {
	if !strings.HasSuffix(strings.ToLower(path), ".md") {
		ext := filepath.Ext(path)
		if ext != "" {
			ext = ext[1:]
		}
		content = []byte(fmt.Sprintf("```%s\n%s\n```", ext, string(content)))
	}

	var buf bytes.Buffer
	if err := md.Convert(content, &buf); err != nil {
		return nil, err
	}

	htmlContent := rewriteMermaidBlocks(buf.String())
	finalHTML := fmt.Sprintf(htmlTemplate, htmlContent)
	return []byte(finalHTML), nil
}

func rewriteMermaidBlocks(content string) string {
	return mermaidBlockPattern.ReplaceAllString(content, `<pre class="mermaid">$1</pre>`)
}

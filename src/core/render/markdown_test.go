package render

import (
	"strings"
	"testing"
)

func TestRenderMarkdownBytesPreservesRegularCodeBlocks(t *testing.T) {
	html, err := RenderMarkdownBytes("notes.md", []byte("```go\nfmt.Println(\"hello\")\n```"))
	if err != nil {
		t.Fatalf("RenderMarkdownBytes returned error: %v", err)
	}
	output := string(html)
	if !strings.Contains(output, `<pre><code class="language-go">`) {
		t.Fatalf("expected regular fenced code block to remain intact: %s", output)
	}
	if !strings.Contains(output, `</code></pre>`) {
		t.Fatalf("expected regular fenced code block to keep closing tags: %s", output)
	}
}

func TestRenderMarkdownBytesRewritesOnlyMermaidBlocks(t *testing.T) {
	markdown := "```mermaid\ngraph TD\nA-->B\n```\n\n```txt\nplain\n```"
	html, err := RenderMarkdownBytes("diagram.md", []byte(markdown))
	if err != nil {
		t.Fatalf("RenderMarkdownBytes returned error: %v", err)
	}
	output := string(html)
	if !strings.Contains(output, `<pre class="mermaid">`) {
		t.Fatalf("expected mermaid block to be rewritten: %s", output)
	}
	if !strings.Contains(output, `<pre><code class="language-txt">`) {
		t.Fatalf("expected non-mermaid fenced block to remain intact: %s", output)
	}
}

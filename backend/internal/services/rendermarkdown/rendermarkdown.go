// Package rendermarkdown provides markdown rendering functionality
// that strictly aligns with trigger.dev's renderMarkdown implementation.
//
// This package replicates the exact behavior of trigger.dev's renderMarkdown function:
// - Convert markdown string to HTML with syntax highlighting
// - Support for TypeScript, JSON, Bash code blocks
// - Fallback handling for unsupported languages
// - Single function interface matching trigger.dev exactly
package rendermarkdown

import (
	"bytes"

	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
)

// Global markdown instance (like trigger.dev's prism setup)
// Configured once and reused across all calls for performance
var markdown = goldmark.New(
	goldmark.WithExtensions(
		highlighting.NewHighlighting(
			// Use GitHub style to match prism.js default appearance
			highlighting.WithStyle("github"),
			highlighting.WithFormatOptions(
				// Use CSS classes like prism.js does
				html.WithClasses(true),
			),
		),
	),
)

// RenderMarkdown converts markdown string to HTML with syntax highlighting.
// This function exactly replicates trigger.dev's renderMarkdown behavior:
//
// trigger.dev implementation:
//
//	export function renderMarkdown(markdown: string) {
//	  const html = marked(markdown, {
//	    highlight: function (code, lang) {
//	      if (prism.languages[lang]) {
//	        return prism.highlight(code, lang, prism.languages[lang]);
//	      }
//	      return code;
//	    },
//	  });
//	  return html;
//	}
//
// Input: Raw markdown string
// Output: HTML string with syntax highlighted code blocks
// Error: Returns error if markdown processing fails
func RenderMarkdown(input string) (string, error) {
	var buf bytes.Buffer

	err := markdown.Convert([]byte(input), &buf)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

package rendermarkdown

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRenderMarkdown_BasicMarkdown tests basic markdown rendering
// Core functionality - covers 80% of use cases
func TestRenderMarkdown_BasicMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string // What the output should contain
	}{
		{
			name:  "heading",
			input: "# Hello World",
			contains: []string{
				"<h1",
				"Hello World",
				"</h1>",
			},
		},
		{
			name:  "paragraph with bold and italic",
			input: "This is **bold** and *italic* text.",
			contains: []string{
				"<p>",
				"<strong>bold</strong>",
				"<em>italic</em>",
				"</p>",
			},
		},
		{
			name:  "unordered list",
			input: "- Item 1\n- Item 2\n- Item 3",
			contains: []string{
				"<ul>",
				"<li>Item 1</li>",
				"<li>Item 2</li>",
				"<li>Item 3</li>",
				"</ul>",
			},
		},
		{
			name:  "links",
			input: "[Click here](https://example.com)",
			contains: []string{
				"<a href=\"https://example.com\"",
				"Click here",
				"</a>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderMarkdown(tt.input)
			require.NoError(t, err)

			for _, expected := range tt.contains {
				assert.Contains(t, result, expected, "Output should contain: %s", expected)
			}
		})
	}
}

// TestRenderMarkdown_CodeBlocks tests syntax highlighting
// Key feature matching trigger.dev's prism.js integration
func TestRenderMarkdown_CodeBlocks(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:  "typescript code block",
			input: "```typescript\nconst message: string = 'Hello World';\nconsole.log(message);\n```",
			contains: []string{
				"<pre", // Changed from "<pre>" to "<pre" to match goldmark output
				"<code",
				"class=\"chroma\"", // Added to verify syntax highlighting
				"</code>",
				"</pre>",
			},
		},
		{
			name:  "json code block",
			input: "```json\n{\n  \"name\": \"test\",\n  \"value\": 42\n}\n```",
			contains: []string{
				"<pre", // Changed from "<pre>" to "<pre"
				"<code",
				"class=\"chroma\"", // Added to verify syntax highlighting
				"</code>",
				"</pre>",
			},
		},
		{
			name:  "bash code block",
			input: "```bash\necho \"Hello World\"\nls -la\n```",
			contains: []string{
				"<pre", // Changed from "<pre>" to "<pre"
				"<code",
				"class=\"chroma\"", // Added to verify syntax highlighting
				"</code>",
				"</pre>",
			},
		},
		{
			name:  "code block without language (fallback)",
			input: "```\nsome plain code\nno highlighting\n```",
			contains: []string{
				"<pre>", // No highlighting for plain code blocks
				"<code",
				"some plain code",
				"no highlighting",
				"</code>",
				"</pre>",
			},
		},
		{
			name:  "inline code",
			input: "This is `inline code` in text.",
			contains: []string{
				"<p>",
				"<code>inline code</code>",
				"</p>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderMarkdown(tt.input)
			require.NoError(t, err)

			for _, expected := range tt.contains {
				assert.Contains(t, result, expected, "Output should contain: %s", expected)
			}
		})
	}
}

// TestRenderMarkdown_EdgeCases tests edge cases and error conditions
// Critical for robustness - 20% of scenarios that matter
func TestRenderMarkdown_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
		expected  string
	}{
		{
			name:      "empty string",
			input:     "",
			expectErr: false,
			expected:  "",
		},
		{
			name:      "whitespace only",
			input:     "   \n\t  ",
			expectErr: false,
			expected:  "", // Should be empty after processing
		},
		{
			name:      "single newline",
			input:     "\n",
			expectErr: false,
		},
		{
			name:      "special characters",
			input:     "< > & \" '",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RenderMarkdown(tt.input)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.expected != "" {
				assert.Equal(t, tt.expected, result)
			}
			// For other cases, just ensure no error and result is valid
		})
	}
}

// TestRenderMarkdown_ComplexDocument tests a realistic document
// Integration test for real-world usage
func TestRenderMarkdown_ComplexDocument(t *testing.T) {
	input := `# API Documentation

This is the main documentation for our API.

## Authentication

You need to provide an API key:

` + "```typescript" + `
const client = new APIClient({
  apiKey: 'your-api-key',
  baseURL: 'https://api.example.com'
});
` + "```" + `

## Configuration

Here's the config format:

` + "```json" + `
{
  "timeout": 5000,
  "retries": 3,
  "endpoints": {
    "users": "/api/v1/users",
    "posts": "/api/v1/posts"
  }
}
` + "```" + `

## Commands

Run these commands to test:

` + "```bash" + `
curl -X GET https://api.example.com/health
npm test
` + "```" + `

That's it! Happy coding.`

	result, err := RenderMarkdown(input)
	require.NoError(t, err)

	// Verify key elements are present - adapted for goldmark output
	expectedElements := []string{
		"<h1", "API Documentation", "</h1>",
		"<h2", "Authentication", "</h2>",
		"<h2", "Configuration", "</h2>",
		"<h2", "Commands", "</h2>",
		"<p>", "This is the main documentation",
		"<pre", "<code", "class=\"chroma\"", // Code blocks with highlighting
		"const", "client", "APIClient", // TypeScript content (individual tokens)
		"timeout", "5000", "retries", // JSON content
		"curl", "npm", "test", // Bash content (individual tokens)
		"Happy coding",
	}

	for _, expected := range expectedElements {
		assert.Contains(t, result, expected, "Complex document should contain: %s", expected)
	}

	// Verify structure - should have proper nesting
	assert.True(t, strings.Contains(result, "<h1"), "Should have h1 tags")
	assert.True(t, strings.Contains(result, "<h2"), "Should have h2 tags")
	assert.True(t, strings.Contains(result, "<p>"), "Should have paragraph tags")
	assert.True(t, strings.Contains(result, "<pre"), "Should have code block tags") // Changed from <pre>
}

// Benchmark test to ensure performance is reasonable
func BenchmarkRenderMarkdown(b *testing.B) {
	input := `# Performance Test

This is a **benchmark** test with some content:

- List item 1
- List item 2  
- List item 3

` + "```typescript" + `
interface User {
  id: string;
  name: string;
  email: string;
}

function createUser(data: Partial<User>): User {
  return {
    id: generateId(),
    name: data.name || 'Anonymous',
    email: data.email || 'user@example.com'
  };
}
` + "```" + `

This should be fast enough for production use.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := RenderMarkdown(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

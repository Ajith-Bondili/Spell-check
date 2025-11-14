package guardrails

import (
	"testing"
)

func TestGuardrails_URLs(t *testing.T) {
	g := NewGuardrails()

	tests := []struct {
		name    string
		word    string
		context string
		skip    bool
	}{
		{
			name:    "HTTP URL",
			word:    "example",
			context: "visit http://example.com for more",
			skip:    true,
		},
		{
			name:    "HTTPS URL",
			word:    "github",
			context: "check out https://github.com/user/repo",
			skip:    true,
		},
		{
			name:    "WWW URL",
			word:    "google",
			context: "go to www.google.com",
			skip:    true,
		},
		{
			name:    "Domain only",
			word:    "example",
			context: "see example.com for details",
			skip:    true,
		},
		{
			name:    "Not part of URL",
			word:    "example",
			context: "this is an example sentence",
			skip:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skip, reason := g.ShouldSkipWord(tt.word, tt.context)
			if skip != tt.skip {
				t.Errorf("Expected skip=%v, got skip=%v (reason: %s)", tt.skip, skip, reason)
			}
			if skip {
				t.Logf("✓ Correctly skipped: %s (reason: %s)", tt.word, reason)
			}
		})
	}
}

func TestGuardrails_Emails(t *testing.T) {
	g := NewGuardrails()

	tests := []struct {
		name    string
		word    string
		context string
		skip    bool
	}{
		{
			name:    "Email address",
			word:    "test",
			context: "contact me at test@example.com",
			skip:    true,
		},
		{
			name:    "Email domain",
			word:    "gmail",
			context: "send to john@gmail.com please",
			skip:    true,
		},
		{
			name:    "Not email",
			word:    "test",
			context: "this is a test message",
			skip:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skip, reason := g.ShouldSkipWord(tt.word, tt.context)
			if skip != tt.skip {
				t.Errorf("Expected skip=%v, got skip=%v (reason: %s)", tt.skip, skip, reason)
			}
			if skip {
				t.Logf("✓ Correctly skipped email: %s", tt.word)
			}
		})
	}
}

func TestGuardrails_CodeVariables(t *testing.T) {
	g := NewGuardrails()

	tests := []struct {
		name string
		word string
		skip bool
	}{
		{
			name: "camelCase",
			word: "myVariable",
			skip: true,
		},
		{
			name: "PascalCase",
			word: "MyClass",
			skip: true,
		},
		{
			name: "snake_case",
			word: "my_variable",
			skip: true,
		},
		{
			name: "With numbers",
			word: "var123",
			skip: true,
		},
		{
			name: "Regular word",
			word: "variable",
			skip: false,
		},
		{
			name: "Capitalized word",
			word: "Variable",
			skip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skip, reason := g.ShouldSkipWord(tt.word, "")
			if skip != tt.skip {
				t.Errorf("Expected skip=%v for '%s', got skip=%v (reason: %s)",
					tt.skip, tt.word, skip, reason)
			}
			if skip {
				t.Logf("✓ Correctly identified as code: %s (reason: %s)", tt.word, reason)
			}
		})
	}
}

func TestGuardrails_Acronyms(t *testing.T) {
	g := NewGuardrails()

	tests := []struct {
		name string
		word string
		skip bool
	}{
		{
			name: "All caps acronym",
			word: "NASA",
			skip: true,
		},
		{
			name: "Another acronym",
			word: "HTTP",
			skip: true,
		},
		{
			name: "Short acronym",
			word: "AI",
			skip: true,
		},
		{
			name: "Regular word",
			word: "word",
			skip: false,
		},
		{
			name: "Single capital letter",
			word: "I",
			skip: false, // "I" is a word, not an acronym
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skip, reason := g.ShouldSkipWord(tt.word, "")
			if skip != tt.skip {
				t.Errorf("Expected skip=%v for '%s', got skip=%v (reason: %s)",
					tt.skip, tt.word, skip, reason)
			}
			if skip {
				t.Logf("✓ Correctly identified acronym: %s", tt.word)
			}
		})
	}
}

func TestGuardrails_SocialMedia(t *testing.T) {
	g := NewGuardrails()

	tests := []struct {
		name string
		word string
		skip bool
	}{
		{
			name: "Hashtag",
			word: "#programming",
			skip: true,
		},
		{
			name: "Mention",
			word: "@username",
			skip: true,
		},
		{
			name: "Regular word",
			word: "programming",
			skip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skip, reason := g.ShouldSkipWord(tt.word, "")
			if skip != tt.skip {
				t.Errorf("Expected skip=%v for '%s', got skip=%v (reason: %s)",
					tt.skip, tt.word, skip, reason)
			}
			if skip {
				t.Logf("✓ Correctly identified social media: %s (reason: %s)", tt.word, reason)
			}
		})
	}
}

func TestGuardrails_FilePaths(t *testing.T) {
	g := NewGuardrails()

	tests := []struct {
		name    string
		word    string
		context string
		skip    bool
	}{
		{
			name:    "Unix path",
			word:    "config",
			context: "edit /etc/config.txt file",
			skip:    true,
		},
		{
			name:    "Home directory",
			word:    "documents",
			context: "save to ~/documents/file.txt",
			skip:    true,
		},
		{
			name:    "Windows path",
			word:    "users",
			context: "located at C:\\Users\\name\\file",
			skip:    true,
		},
		{
			name:    "Not a path",
			word:    "config",
			context: "update the config settings",
			skip:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skip, reason := g.ShouldSkipWord(tt.word, tt.context)
			if skip != tt.skip {
				t.Errorf("Expected skip=%v, got skip=%v (reason: %s)", tt.skip, skip, reason)
			}
			if skip {
				t.Logf("✓ Correctly identified file path: %s", tt.word)
			}
		})
	}
}

func TestGuardrails_Numbers(t *testing.T) {
	g := NewGuardrails()

	tests := []struct {
		name string
		word string
		skip bool
	}{
		{
			name: "Integer",
			word: "123",
			skip: true,
		},
		{
			name: "Decimal",
			word: "3.14",
			skip: true,
		},
		{
			name: "Word with digits",
			word: "test123",
			skip: true,
		},
		{
			name: "Regular word",
			word: "test",
			skip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skip, reason := g.ShouldSkipWord(tt.word, "")
			if skip != tt.skip {
				t.Errorf("Expected skip=%v for '%s', got skip=%v (reason: %s)",
					tt.skip, tt.word, skip, reason)
			}
			if skip {
				t.Logf("✓ Correctly identified number: %s (reason: %s)", tt.word, reason)
			}
		})
	}
}

func TestGuardrails_HexColors(t *testing.T) {
	g := NewGuardrails()

	tests := []struct {
		name string
		word string
		skip bool
	}{
		{
			name: "Short hex",
			word: "#fff",
			skip: true,
		},
		{
			name: "Long hex",
			word: "#ff5733",
			skip: true,
		},
		{
			name: "Not hex color",
			word: "#hashtag",
			skip: true, // Still skipped because it's a hashtag
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skip, _ := g.ShouldSkipWord(tt.word, "")
			if skip != tt.skip {
				t.Errorf("Expected skip=%v for '%s', got skip=%v", tt.skip, tt.word, skip)
			}
		})
	}
}

func TestGuardrails_VersionNumbers(t *testing.T) {
	g := NewGuardrails()

	tests := []struct {
		name string
		word string
		skip bool
	}{
		{
			name: "Semantic version",
			word: "v1.2.3",
			skip: true,
		},
		{
			name: "Version without v",
			word: "2.0.1",
			skip: true,
		},
		{
			name: "Major.minor",
			word: "3.14",
			skip: true, // Caught as number
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skip, reason := g.ShouldSkipWord(tt.word, "")
			if skip != tt.skip {
				t.Errorf("Expected skip=%v for '%s', got skip=%v (reason: %s)",
					tt.skip, tt.word, skip, reason)
			}
			if skip {
				t.Logf("✓ Correctly identified version: %s (reason: %s)", tt.word, reason)
			}
		})
	}
}

func TestGuardrails_CodeContext(t *testing.T) {
	g := NewGuardrails()

	tests := []struct {
		name    string
		context string
		skip    bool
	}{
		{
			name:    "JavaScript code",
			context: "function myFunc() { return getData(); }",
			skip:    true,
		},
		{
			name:    "Python code",
			context: "def my_function(param): return value",
			skip:    true,
		},
		{
			name:    "Regular text",
			context: "This is a normal sentence with regular words",
			skip:    false,
		},
		{
			name:    "Text with some code terms",
			context: "I need to update the function later today",
			skip:    false, // Not enough code patterns
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skip, reason := g.ShouldSkipContext(tt.context)
			if skip != tt.skip {
				t.Errorf("Expected skip=%v, got skip=%v (reason: %s)", tt.skip, skip, reason)
			}
			if skip {
				t.Logf("✓ Correctly identified code context (reason: %s)", reason)
			}
		})
	}
}

func TestGuardrails_RealWorldScenarios(t *testing.T) {
	g := NewGuardrails()

	scenarios := []struct {
		name        string
		word        string
		context     string
		shouldSkip  bool
		description string
	}{
		{
			name:        "Gmail URL in email",
			word:        "gmail",
			context:     "send to john@gmail.com",
			shouldSkip:  true,
			description: "Should skip email addresses",
		},
		{
			name:        "GitHub repo",
			word:        "github",
			context:     "clone from https://github.com/user/repo",
			shouldSkip:  true,
			description: "Should skip URLs",
		},
		{
			name:        "Variable name",
			word:        "userName",
			context:     "set userName to john",
			shouldSkip:  true,
			description: "Should skip camelCase variables",
		},
		{
			name:        "Normal typo",
			word:        "teh",
			context:     "this is teh best",
			shouldSkip:  false,
			description: "Should NOT skip normal typos",
		},
		{
			name:        "API endpoint",
			word:        "api",
			context:     "call api.example.com/users",
			shouldSkip:  true,
			description: "Should skip URLs with subdomains",
		},
		{
			name:        "File extension",
			word:        "config",
			context:     "edit config.json file",
			shouldSkip:  false,
			description: "Should allow correction in normal text",
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			skip, reason := g.ShouldSkipWord(tt.word, tt.context)
			if skip != tt.shouldSkip {
				t.Errorf("%s: Expected skip=%v, got skip=%v (reason: %s)",
					tt.description, tt.shouldSkip, skip, reason)
			} else {
				t.Logf("✓ %s", tt.description)
				if skip {
					t.Logf("  Reason: %s", reason)
				}
			}
		})
	}
}

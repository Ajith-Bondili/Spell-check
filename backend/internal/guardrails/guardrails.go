package guardrails

import (
	"regexp"
	"strings"
)

// Guardrails determines when NOT to apply autocorrect
// This prevents corrections in inappropriate contexts like URLs, code, passwords, etc.
type Guardrails struct {
	// Compiled regex patterns for efficiency
	urlPattern       *regexp.Regexp
	emailPattern     *regexp.Regexp
	codePattern      *regexp.Regexp
	hashtagPattern   *regexp.Regexp
	mentionPattern   *regexp.Regexp
	filePathPattern  *regexp.Regexp
	hexColorPattern  *regexp.Regexp
	versionPattern   *regexp.Regexp
}

// NewGuardrails creates a new guardrails checker
func NewGuardrails() *Guardrails {
	return &Guardrails{
		// URL pattern: http://, https://, www., or domain.com
		urlPattern: regexp.MustCompile(`(?i)(https?://|www\.|[a-z0-9.-]+\.(com|org|net|edu|gov|io|co|ai|dev|app))`),

		// Email pattern
		emailPattern: regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),

		// Code-like patterns (variable names, function calls, etc.)
		// Matches: camelCase, snake_case, function(), object.property
		codePattern: regexp.MustCompile(`([a-z][a-zA-Z0-9]*[A-Z][a-zA-Z0-9]*|[a-z_]+_[a-z_]+|\w+\(\)|[a-zA-Z]+\.[a-zA-Z]+)`),

		// Social media patterns
		hashtagPattern: regexp.MustCompile(`#[a-zA-Z][a-zA-Z0-9_]*`),
		mentionPattern: regexp.MustCompile(`@[a-zA-Z][a-zA-Z0-9_]*`),

		// File paths
		filePathPattern: regexp.MustCompile(`([a-zA-Z]:\\|/|~/)[a-zA-Z0-9_/\\.-]+`),

		// Hex colors (#fff, #ffffff)
		hexColorPattern: regexp.MustCompile(`#[0-9a-fA-F]{3,6}`),

		// Version numbers (v1.2.3, 2.0.1)
		versionPattern: regexp.MustCompile(`v?\d+\.\d+(\.\d+)?`),
	}
}

// ShouldSkipWord determines if a word should NOT be autocorrected
func (g *Guardrails) ShouldSkipWord(word string, context string) (bool, string) {
	// 1. Check if word is all caps (likely acronym)
	if isAllCaps(word) && len(word) > 1 {
		return true, "all_caps_acronym"
	}

	// 2. Check if word is a number
	if isNumber(word) {
		return true, "number"
	}

	// 3. Check if word contains numbers (like "abc123")
	if containsDigits(word) {
		return true, "contains_digits"
	}

	// 4. Check if word is camelCase or PascalCase (code variable)
	if isCamelCase(word) || isPascalCase(word) {
		return true, "code_variable"
	}

	// 5. Check if word contains underscores (snake_case)
	if strings.Contains(word, "_") {
		return true, "snake_case"
	}

	// 6. Check if word is part of a URL
	if g.isPartOfURL(word, context) {
		return true, "url"
	}

	// 7. Check if word is part of an email
	if g.isPartOfEmail(word, context) {
		return true, "email"
	}

	// 8. Check if word is a hashtag
	if strings.HasPrefix(word, "#") {
		return true, "hashtag"
	}

	// 9. Check if word is a mention
	if strings.HasPrefix(word, "@") {
		return true, "mention"
	}

	// 10. Check if word looks like a file path
	if g.isPartOfFilePath(word, context) {
		return true, "file_path"
	}

	// 11. Check if word is a hex color
	if g.hexColorPattern.MatchString(word) {
		return true, "hex_color"
	}

	// 12. Check if word is a version number
	if g.versionPattern.MatchString(word) {
		return true, "version_number"
	}

	// 13. Check if word has special characters (likely code or formatting)
	if hasSpecialCharacters(word) {
		return true, "special_characters"
	}

	// Safe to autocorrect
	return false, ""
}

// ShouldSkipContext determines if the entire context should be skipped
func (g *Guardrails) ShouldSkipContext(context string) (bool, string) {
	// Check if context looks like code
	if g.looksLikeCode(context) {
		return true, "code_block"
	}

	// Check if context is predominantly URLs
	if g.isPredominantlyURLs(context) {
		return true, "url_heavy"
	}

	return false, ""
}

// Helper functions

func isAllCaps(s string) bool {
	hasLetter := false
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			return false // Found lowercase
		}
		if r >= 'A' && r <= 'Z' {
			hasLetter = true
		}
	}
	return hasLetter
}

func isNumber(s string) bool {
	matched, _ := regexp.MatchString(`^\d+(\.\d+)?$`, s)
	return matched
}

func containsDigits(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func isCamelCase(s string) bool {
	// Starts with lowercase, has uppercase letters
	if len(s) == 0 {
		return false
	}
	if s[0] < 'a' || s[0] > 'z' {
		return false
	}

	hasUpper := false
	for _, r := range s[1:] {
		if r >= 'A' && r <= 'Z' {
			hasUpper = true
			break
		}
	}
	return hasUpper
}

func isPascalCase(s string) bool {
	// Starts with uppercase, has more uppercase letters
	if len(s) == 0 {
		return false
	}
	if s[0] < 'A' || s[0] > 'Z' {
		return false
	}

	upperCount := 1
	for _, r := range s[1:] {
		if r >= 'A' && r <= 'Z' {
			upperCount++
		}
	}
	return upperCount > 1
}

func hasSpecialCharacters(s string) bool {
	// Characters that suggest code/technical content
	specialChars := []string{
		"$", "{", "}", "[", "]", "<", ">", "\\", "|",
		"=", "+", "*", "^", "%", "~", "`",
	}

	for _, char := range specialChars {
		if strings.Contains(s, char) {
			return true
		}
	}
	return false
}

func (g *Guardrails) isPartOfURL(word string, context string) bool {
	// Extract a window around the word
	lowerContext := strings.ToLower(context)
	lowerWord := strings.ToLower(word)

	// Find word position
	idx := strings.Index(lowerContext, lowerWord)
	if idx == -1 {
		return false
	}

	// Extract context window (50 chars before and after)
	start := max(0, idx-50)
	end := min(len(lowerContext), idx+len(lowerWord)+50)
	window := lowerContext[start:end]

	// Check if URL pattern appears in window
	return g.urlPattern.MatchString(window)
}

func (g *Guardrails) isPartOfEmail(word string, context string) bool {
	lowerContext := strings.ToLower(context)
	lowerWord := strings.ToLower(word)

	idx := strings.Index(lowerContext, lowerWord)
	if idx == -1 {
		return false
	}

	start := max(0, idx-50)
	end := min(len(lowerContext), idx+len(lowerWord)+50)
	window := lowerContext[start:end]

	return g.emailPattern.MatchString(window)
}

func (g *Guardrails) isPartOfFilePath(word string, context string) bool {
	lowerContext := strings.ToLower(context)
	lowerWord := strings.ToLower(word)

	idx := strings.Index(lowerContext, lowerWord)
	if idx == -1 {
		return false
	}

	start := max(0, idx-50)
	end := min(len(lowerContext), idx+len(lowerWord)+50)
	window := lowerContext[start:end]

	return g.filePathPattern.MatchString(window)
}

func (g *Guardrails) looksLikeCode(context string) bool {
	words := strings.Fields(context)
	if len(words) == 0 {
		return false
	}

	codeIndicators := 0

	// Count various code indicators
	for _, word := range words {
		// snake_case (Python style)
		if strings.Contains(word, "_") && len(strings.Split(word, "_")) > 1 {
			codeIndicators++
		}

		// camelCase or PascalCase
		if isCamelCase(word) || isPascalCase(word) {
			codeIndicators++
		}

		// Function calls
		if strings.Contains(word, "()") {
			codeIndicators++
		}

		// Common code keywords
		codeKeywords := []string{"def", "function", "return", "class", "import", "var", "let", "const"}
		for _, keyword := range codeKeywords {
			if word == keyword {
				codeIndicators++
			}
		}
	}

	// If more than 30% of words are code-like, it's probably code
	codeRatio := float64(codeIndicators) / float64(len(words))
	return codeRatio > 0.3
}

func (g *Guardrails) isPredominantlyURLs(context string) bool {
	urls := g.urlPattern.FindAllString(context, -1)
	words := strings.Fields(context)

	if len(words) == 0 {
		return false
	}

	urlRatio := float64(len(urls)) / float64(len(words))
	return urlRatio > 0.5
}

// Helper min/max functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

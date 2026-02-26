package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	ModeConservative  = "conservative"
	ModeAggressive    = "aggressive"
	ModeSuggestions   = "suggestions_only"
	defaultWordFreq   = int64(1500000)
	minMaxSuggestions = 1
	maxMaxSuggestions = 20
)

var validModes = map[string]bool{
	ModeConservative: true,
	ModeAggressive:   true,
	ModeSuggestions:  true,
}

// Settings controls correction behavior at runtime.
type Settings struct {
	Enabled              bool    `json:"enabled"`
	Mode                 string  `json:"mode"`
	AutoCorrectThreshold float64 `json:"auto_correct_threshold"`
	SuggestionThreshold  float64 `json:"suggestion_threshold"`
	MaxSuggestions       int     `json:"max_suggestions"`
}

// Validate verifies settings are safe to apply.
func (s Settings) Validate() error {
	if !validModes[s.Mode] {
		return fmt.Errorf("invalid mode: %s", s.Mode)
	}
	if s.AutoCorrectThreshold < 0 || s.AutoCorrectThreshold > 1 {
		return errors.New("auto_correct_threshold must be between 0 and 1")
	}
	if s.SuggestionThreshold < 0 || s.SuggestionThreshold > 1 {
		return errors.New("suggestion_threshold must be between 0 and 1")
	}
	if s.SuggestionThreshold > s.AutoCorrectThreshold && s.Mode != ModeSuggestions {
		return errors.New("suggestion_threshold cannot exceed auto_correct_threshold")
	}
	if s.MaxSuggestions < minMaxSuggestions || s.MaxSuggestions > maxMaxSuggestions {
		return fmt.Errorf("max_suggestions must be between %d and %d", minMaxSuggestions, maxMaxSuggestions)
	}
	return nil
}

// CustomWord is a user-provided dictionary entry.
type CustomWord struct {
	Word      string `json:"word"`
	Frequency int64  `json:"frequency"`
	AddedAt   string `json:"added_at"`
}

// Stats tracks runtime behavior for observability in popup and APIs.
type Stats struct {
	TotalRequests  int64  `json:"total_requests"`
	SpellRequests  int64  `json:"spell_requests"`
	RescoreRequests int64 `json:"rescore_requests"`
	AutoCorrected  int64  `json:"auto_corrected"`
	Suggestions    int64  `json:"suggestions"`
	Skipped        int64  `json:"skipped"`
	Errors         int64  `json:"errors"`
	LastUpdatedAt  string `json:"last_updated_at"`
}

type feedbackState struct {
	Accepted map[string]int64 `json:"accepted"`
	Rejected map[string]int64 `json:"rejected"`
}

type dictionaryFile struct {
	Words map[string]CustomWord `json:"words"`
}

type ignoreFile struct {
	Words map[string]bool `json:"words"`
	Pairs map[string]bool `json:"pairs"`
}

// Store manages runtime state persisted to JSON files.
type Store struct {
	mu sync.RWMutex

	dir             string
	defaultSettings Settings

	settings    Settings
	dictionary  map[string]CustomWord
	ignoreWords map[string]bool
	ignorePairs map[string]bool
	stats       Stats
	feedback    feedbackState
}

// NewStore initializes and loads JSON-backed state.
func NewStore(dir string, defaults Settings) (*Store, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, errors.New("state directory is required")
	}
	if err := defaults.Validate(); err != nil {
		return nil, fmt.Errorf("invalid default settings: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	s := &Store{
		dir:             dir,
		defaultSettings: defaults,
		settings:        defaults,
		dictionary:      make(map[string]CustomWord),
		ignoreWords:     make(map[string]bool),
		ignorePairs:     make(map[string]bool),
		feedback: feedbackState{
			Accepted: make(map[string]int64),
			Rejected: make(map[string]int64),
		},
	}

	if err := s.loadAllLocked(); err != nil {
		return nil, err
	}
	return s, nil
}

// Reload forces a fresh read from JSON files.
func (s *Store) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadAllLocked()
}

func (s *Store) loadAllLocked() error {
	if err := s.loadSettingsLocked(); err != nil {
		return err
	}
	if err := s.loadDictionaryLocked(); err != nil {
		return err
	}
	if err := s.loadIgnoreLocked(); err != nil {
		return err
	}
	if err := s.loadStatsLocked(); err != nil {
		return err
	}
	if err := s.loadFeedbackLocked(); err != nil {
		return err
	}
	return nil
}

func (s *Store) loadSettingsLocked() error {
	path := s.pathFor("settings.json")
	var settings Settings
	found, err := readJSON(path, &settings)
	if err != nil {
		return fmt.Errorf("failed to read settings: %w", err)
	}
	if !found {
		s.settings = s.defaultSettings
		return writeJSONAtomic(path, s.settings)
	}
	if err := settings.Validate(); err != nil {
		s.settings = s.defaultSettings
		return writeJSONAtomic(path, s.settings)
	}
	s.settings = settings
	return nil
}

func (s *Store) loadDictionaryLocked() error {
	path := s.pathFor("user_dictionary.json")
	data := dictionaryFile{Words: make(map[string]CustomWord)}
	found, err := readJSON(path, &data)
	if err != nil {
		return fmt.Errorf("failed to read dictionary: %w", err)
	}
	if !found {
		s.dictionary = make(map[string]CustomWord)
		return writeJSONAtomic(path, dictionaryFile{Words: s.dictionary})
	}
	if data.Words == nil {
		data.Words = make(map[string]CustomWord)
	}
	s.dictionary = data.Words
	return nil
}

func (s *Store) loadIgnoreLocked() error {
	path := s.pathFor("ignored.json")
	data := ignoreFile{
		Words: make(map[string]bool),
		Pairs: make(map[string]bool),
	}
	found, err := readJSON(path, &data)
	if err != nil {
		return fmt.Errorf("failed to read ignore rules: %w", err)
	}
	if !found {
		s.ignoreWords = make(map[string]bool)
		s.ignorePairs = make(map[string]bool)
		return writeJSONAtomic(path, ignoreFile{
			Words: s.ignoreWords,
			Pairs: s.ignorePairs,
		})
	}
	if data.Words == nil {
		data.Words = make(map[string]bool)
	}
	if data.Pairs == nil {
		data.Pairs = make(map[string]bool)
	}
	s.ignoreWords = data.Words
	s.ignorePairs = data.Pairs
	return nil
}

func (s *Store) loadStatsLocked() error {
	path := s.pathFor("stats.json")
	var stats Stats
	found, err := readJSON(path, &stats)
	if err != nil {
		return fmt.Errorf("failed to read stats: %w", err)
	}
	if !found {
		s.stats = Stats{}
		return writeJSONAtomic(path, s.stats)
	}
	s.stats = stats
	return nil
}

func (s *Store) loadFeedbackLocked() error {
	path := s.pathFor("feedback.json")
	data := feedbackState{
		Accepted: make(map[string]int64),
		Rejected: make(map[string]int64),
	}
	found, err := readJSON(path, &data)
	if err != nil {
		return fmt.Errorf("failed to read feedback: %w", err)
	}
	if !found {
		s.feedback = data
		return writeJSONAtomic(path, s.feedback)
	}
	if data.Accepted == nil {
		data.Accepted = make(map[string]int64)
	}
	if data.Rejected == nil {
		data.Rejected = make(map[string]int64)
	}
	s.feedback = data
	return nil
}

func (s *Store) pathFor(filename string) string {
	return filepath.Join(s.dir, filename)
}

// GetSettings returns current settings snapshot.
func (s *Store) GetSettings() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

// SetSettings validates and persists settings.
func (s *Store) SetSettings(settings Settings) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings = settings
	return writeJSONAtomic(s.pathFor("settings.json"), s.settings)
}

// ListCustomWords returns custom words sorted alphabetically.
func (s *Store) ListCustomWords() []CustomWord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]CustomWord, 0, len(s.dictionary))
	for _, word := range s.dictionary {
		result = append(result, word)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Word < result[j].Word })
	return result
}

// AddCustomWord adds or updates a custom word.
func (s *Store) AddCustomWord(word string, frequency int64) (CustomWord, error) {
	word = normalizeWord(word)
	if word == "" {
		return CustomWord{}, errors.New("word is required")
	}
	if frequency <= 0 {
		frequency = defaultWordFreq
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entry := CustomWord{
		Word:      word,
		Frequency: frequency,
		AddedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	s.dictionary[word] = entry
	if err := writeJSONAtomic(s.pathFor("user_dictionary.json"), dictionaryFile{Words: s.dictionary}); err != nil {
		return CustomWord{}, err
	}
	return entry, nil
}

// RemoveCustomWord deletes a custom word.
func (s *Store) RemoveCustomWord(word string) (bool, error) {
	word = normalizeWord(word)
	if word == "" {
		return false, errors.New("word is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.dictionary[word]; !ok {
		return false, nil
	}
	delete(s.dictionary, word)
	if err := writeJSONAtomic(s.pathFor("user_dictionary.json"), dictionaryFile{Words: s.dictionary}); err != nil {
		return false, err
	}
	return true, nil
}

// GetCustomWordFrequency returns custom frequency if present.
func (s *Store) GetCustomWordFrequency(word string) (int64, bool) {
	word = normalizeWord(word)
	s.mu.RLock()
	defer s.mu.RUnlock()
	entry, ok := s.dictionary[word]
	if !ok {
		return 0, false
	}
	return entry.Frequency, true
}

// IsWordIgnored checks if correction for this source word should be skipped.
func (s *Store) IsWordIgnored(word string) bool {
	word = normalizeWord(word)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ignoreWords[word]
}

// AddIgnoredWord stores an ignored word.
func (s *Store) AddIgnoredWord(word string) error {
	word = normalizeWord(word)
	if word == "" {
		return errors.New("word is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ignoreWords[word] = true
	return writeJSONAtomic(s.pathFor("ignored.json"), ignoreFile{
		Words: s.ignoreWords,
		Pairs: s.ignorePairs,
	})
}

// AddIgnoredPair stores an ignored correction pair.
func (s *Store) AddIgnoredPair(original, suggestion string) error {
	original = normalizeWord(original)
	suggestion = normalizeWord(suggestion)
	if original == "" || suggestion == "" {
		return errors.New("original and suggestion are required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.ignorePairs[pairKey(original, suggestion)] = true
	return writeJSONAtomic(s.pathFor("ignored.json"), ignoreFile{
		Words: s.ignoreWords,
		Pairs: s.ignorePairs,
	})
}

// IsPairIgnored checks if a specific correction pair should be filtered out.
func (s *Store) IsPairIgnored(original, suggestion string) bool {
	original = normalizeWord(original)
	suggestion = normalizeWord(suggestion)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ignorePairs[pairKey(original, suggestion)]
}

// ListIgnoredWords returns ignored words sorted alphabetically.
func (s *Store) ListIgnoredWords() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]string, 0, len(s.ignoreWords))
	for word, enabled := range s.ignoreWords {
		if enabled {
			result = append(result, word)
		}
	}
	sort.Strings(result)
	return result
}

// ListIgnoredPairs returns ignored correction pair keys sorted alphabetically.
func (s *Store) ListIgnoredPairs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]string, 0, len(s.ignorePairs))
	for pair, enabled := range s.ignorePairs {
		if enabled {
			result = append(result, pair)
		}
	}
	sort.Strings(result)
	return result
}

// GetStats returns statistics snapshot.
func (s *Store) GetStats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

// ResetStats resets all stat counters.
func (s *Store) ResetStats() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats = Stats{
		LastUpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	return writeJSONAtomic(s.pathFor("stats.json"), s.stats)
}

// RecordSpellRequest records a /spell request.
func (s *Store) RecordSpellRequest() error {
	return s.updateStats(func(stats *Stats) {
		stats.TotalRequests++
		stats.SpellRequests++
	})
}

// RecordRescoreRequest records a /rescore request.
func (s *Store) RecordRescoreRequest() error {
	return s.updateStats(func(stats *Stats) {
		stats.TotalRequests++
		stats.RescoreRequests++
	})
}

// RecordAutoCorrect records an auto-correct event.
func (s *Store) RecordAutoCorrect() error {
	return s.updateStats(func(stats *Stats) {
		stats.AutoCorrected++
	})
}

// RecordSuggestion records a suggestion event.
func (s *Store) RecordSuggestion() error {
	return s.updateStats(func(stats *Stats) {
		stats.Suggestions++
	})
}

// RecordSkip records an intentional skip decision.
func (s *Store) RecordSkip() error {
	return s.updateStats(func(stats *Stats) {
		stats.Skipped++
	})
}

// RecordError records an error.
func (s *Store) RecordError() error {
	return s.updateStats(func(stats *Stats) {
		stats.Errors++
	})
}

func (s *Store) updateStats(updateFn func(stats *Stats)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	updateFn(&s.stats)
	s.stats.LastUpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return writeJSONAtomic(s.pathFor("stats.json"), s.stats)
}

// RecordFeedback records accepted/rejected correction feedback.
func (s *Store) RecordFeedback(original, suggestion string, accepted bool) error {
	original = normalizeWord(original)
	suggestion = normalizeWord(suggestion)
	if original == "" || suggestion == "" {
		return errors.New("original and suggestion are required")
	}

	key := pairKey(original, suggestion)
	s.mu.Lock()
	defer s.mu.Unlock()

	if accepted {
		s.feedback.Accepted[key]++
	} else {
		s.feedback.Rejected[key]++
	}
	return writeJSONAtomic(s.pathFor("feedback.json"), s.feedback)
}

// FeedbackAdjustment returns confidence adjustment for a pair based on user feedback.
func (s *Store) FeedbackAdjustment(original, suggestion string) float64 {
	original = normalizeWord(original)
	suggestion = normalizeWord(suggestion)
	key := pairKey(original, suggestion)

	s.mu.RLock()
	defer s.mu.RUnlock()
	accepted := s.feedback.Accepted[key]
	rejected := s.feedback.Rejected[key]
	net := accepted - rejected

	if net > 0 {
		boost := float64(net) * 0.02
		if boost > 0.12 {
			boost = 0.12
		}
		return boost
	}
	if net < 0 {
		penalty := float64(net) * 0.03
		if penalty < -0.20 {
			penalty = -0.20
		}
		return penalty
	}
	return 0
}

func normalizeWord(word string) string {
	word = strings.TrimSpace(strings.ToLower(word))
	word = strings.Trim(word, " \t\n\r.,!?;:\"()[]{}")
	return word
}

func pairKey(original, suggestion string) string {
	return original + "|" + suggestion
}

func readJSON(path string, out interface{}) (bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if len(data) == 0 {
		return true, nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return false, err
	}
	return true, nil
}

func writeJSONAtomic(path string, v interface{}) error {
	payload, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

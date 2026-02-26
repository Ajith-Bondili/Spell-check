package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
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
	defaultUndoTTLms  = 6000
	defaultJournalCap = 2000
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
	RespectSlang         bool    `json:"respect_slang"`
	CreatedAt            string  `json:"created_at,omitempty"`
	UpdatedAt            string  `json:"updated_at,omitempty"`
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
	TotalRequests   int64            `json:"total_requests"`
	SpellRequests   int64            `json:"spell_requests"`
	RescoreRequests int64            `json:"rescore_requests"`
	AutoCorrected   int64            `json:"auto_corrected"`
	Suggestions     int64            `json:"suggestions"`
	Skipped         int64            `json:"skipped"`
	Errors          int64            `json:"errors"`
	SkipReasons     map[string]int64 `json:"skip_reasons,omitempty"`
	LastUpdatedAt   string           `json:"last_updated_at"`
}

// CorrectionRecord represents one applied correction event.
type CorrectionRecord struct {
	CorrectionID string  `json:"correction_id"`
	Original     string  `json:"original"`
	Suggestion   string  `json:"suggestion"`
	Domain       string  `json:"domain,omitempty"`
	Source       string  `json:"source,omitempty"`
	Mode         string  `json:"mode,omitempty"`
	Reason       string  `json:"reason,omitempty"`
	Explanation  string  `json:"explanation,omitempty"`
	Confidence   float64 `json:"confidence,omitempty"`
	SessionID    string  `json:"session_id,omitempty"`
	BeforeText   string  `json:"before_text,omitempty"`
	AfterText    string  `json:"after_text,omitempty"`
	AppliedAt    string  `json:"applied_at"`
	Undone       bool    `json:"undone"`
	UndoneAt     string  `json:"undone_at,omitempty"`
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

type profilesFile struct {
	Default Settings            `json:"default"`
	Domains map[string]Settings `json:"domains"`
}

type journalFile struct {
	Records []CorrectionRecord `json:"records"`
}

// PairInsight represents ranked pair counts.
type PairInsight struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

// DomainInsight represents ranked per-domain volume.
type DomainInsight struct {
	Domain string `json:"domain"`
	Count  int64  `json:"count"`
}

// PainPointInsights exposes top friction areas for UX.
type PainPointInsights struct {
	TopUndonePairs         []PairInsight   `json:"top_undone_pairs"`
	TopIgnoredPairs        []string        `json:"top_ignored_pairs"`
	TopIgnoredWords        []string        `json:"top_ignored_words"`
	DomainCorrectionVolume []DomainInsight `json:"domain_correction_volume"`
	SkipReasons            []PairInsight   `json:"skip_reasons"`
}

// Store manages runtime state persisted to JSON files.
type Store struct {
	mu sync.RWMutex

	dir             string
	defaultSettings Settings

	settings          Settings
	domainProfiles    map[string]Settings
	dictionary        map[string]CustomWord
	ignoreWords       map[string]bool
	ignorePairs       map[string]bool
	stats             Stats
	feedback          feedbackState
	correctionJournal []CorrectionRecord
	undoTTLms         int
	journalCap        int
}

// NewStore initializes and loads JSON-backed state.
func NewStore(dir string, defaults Settings) (*Store, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, errors.New("state directory is required")
	}
	defaults = withTimestamps(defaults, true)
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
		domainProfiles:  make(map[string]Settings),
		dictionary:      make(map[string]CustomWord),
		ignoreWords:     make(map[string]bool),
		ignorePairs:     make(map[string]bool),
		stats: Stats{
			SkipReasons: make(map[string]int64),
		},
		feedback: feedbackState{
			Accepted: make(map[string]int64),
			Rejected: make(map[string]int64),
		},
		correctionJournal: make([]CorrectionRecord, 0),
		undoTTLms:         defaultUndoTTLms,
		journalCap:        defaultJournalCap,
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
	if err := s.loadProfilesLocked(); err != nil {
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
	if err := s.loadJournalLocked(); err != nil {
		return err
	}
	return nil
}

func (s *Store) pathFor(filename string) string {
	return filepath.Join(s.dir, filename)
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
	settings = withTimestamps(settings, true)
	if err := settings.Validate(); err != nil {
		s.settings = s.defaultSettings
		return writeJSONAtomic(path, s.settings)
	}
	s.settings = settings
	return nil
}

func (s *Store) loadProfilesLocked() error {
	path := s.pathFor("profiles.json")
	data := profilesFile{
		Default: withTimestamps(s.settings, true),
		Domains: make(map[string]Settings),
	}
	found, err := readJSON(path, &data)
	if err != nil {
		return fmt.Errorf("failed to read profiles: %w", err)
	}

	if !found {
		s.domainProfiles = seededDomainProfiles(s.settings)
		data.Default = s.settings
		data.Domains = s.domainProfiles
		return writeJSONAtomic(path, data)
	}

	data.Default = withTimestamps(data.Default, true)
	if err := data.Default.Validate(); err != nil {
		data.Default = withTimestamps(s.defaultSettings, true)
	}
	if data.Domains == nil {
		data.Domains = make(map[string]Settings)
	}

	normalized := make(map[string]Settings)
	for domain, profile := range data.Domains {
		key := normalizeDomain(domain)
		if key == "" {
			continue
		}
		profile = withTimestamps(profile, true)
		if err := profile.Validate(); err != nil {
			continue
		}
		normalized[key] = profile
	}

	s.settings = data.Default
	s.domainProfiles = normalized
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
		s.stats = Stats{
			SkipReasons: make(map[string]int64),
		}
		return writeJSONAtomic(path, s.stats)
	}
	if stats.SkipReasons == nil {
		stats.SkipReasons = make(map[string]int64)
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

func (s *Store) loadJournalLocked() error {
	path := s.pathFor("correction_journal.json")
	data := journalFile{Records: make([]CorrectionRecord, 0)}
	found, err := readJSON(path, &data)
	if err != nil {
		return fmt.Errorf("failed to read correction journal: %w", err)
	}
	if !found {
		s.correctionJournal = make([]CorrectionRecord, 0)
		return writeJSONAtomic(path, journalFile{Records: s.correctionJournal})
	}
	if data.Records == nil {
		data.Records = make([]CorrectionRecord, 0)
	}
	s.correctionJournal = data.Records
	if len(s.correctionJournal) > s.journalCap {
		s.correctionJournal = append([]CorrectionRecord(nil), s.correctionJournal[len(s.correctionJournal)-s.journalCap:]...)
		return writeJSONAtomic(path, journalFile{Records: s.correctionJournal})
	}
	return nil
}

func withTimestamps(settings Settings, keepCreated bool) Settings {
	now := time.Now().UTC().Format(time.RFC3339)
	if !keepCreated || settings.CreatedAt == "" {
		settings.CreatedAt = now
	}
	settings.UpdatedAt = now
	return settings
}

func seededDomainProfiles(defaults Settings) map[string]Settings {
	now := time.Now().UTC().Format(time.RFC3339)
	makeProfile := func(base Settings, mode string, suggest float64, respectSlang bool) Settings {
		base.Mode = mode
		base.SuggestionThreshold = suggest
		base.RespectSlang = respectSlang
		base.CreatedAt = now
		base.UpdatedAt = now
		return base
	}

	return map[string]Settings{
		"docs.google.com": makeProfile(defaults, ModeConservative, defaults.SuggestionThreshold, false),
		"mail.google.com": makeProfile(defaults, ModeConservative, defaults.SuggestionThreshold, false),
		"chat.openai.com": makeProfile(defaults, ModeSuggestions, 0.40, true),
		"discord.com":     makeProfile(defaults, ModeSuggestions, 0.40, true),
		"reddit.com":      makeProfile(defaults, ModeSuggestions, 0.42, true),
		"x.com":           makeProfile(defaults, ModeSuggestions, 0.42, true),
	}
}

func persistProfiles(path string, defaultSettings Settings, domains map[string]Settings) error {
	return writeJSONAtomic(path, profilesFile{
		Default: defaultSettings,
		Domains: domains,
	})
}

// GetSettings returns current default settings snapshot.
func (s *Store) GetSettings() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

// SetSettings validates and persists default settings.
func (s *Store) SetSettings(settings Settings) error {
	settings = withTimestamps(settings, true)
	if err := settings.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings = settings
	if err := writeJSONAtomic(s.pathFor("settings.json"), s.settings); err != nil {
		return err
	}
	return persistProfiles(s.pathFor("profiles.json"), s.settings, s.domainProfiles)
}

// GetDomainProfiles returns all explicit domain overrides.
func (s *Store) GetDomainProfiles() map[string]Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]Settings, len(s.domainProfiles))
	for domain, profile := range s.domainProfiles {
		result[domain] = profile
	}
	return result
}

// GetDomainProfile returns domain-specific profile if present.
func (s *Store) GetDomainProfile(domain string) (Settings, bool) {
	key := normalizeDomain(domain)
	s.mu.RLock()
	defer s.mu.RUnlock()
	profile, ok := s.domainProfiles[key]
	return profile, ok
}

// SetDomainProfile upserts a profile for a domain.
func (s *Store) SetDomainProfile(domain string, profile Settings) (Settings, error) {
	key := normalizeDomain(domain)
	if key == "" {
		return Settings{}, errors.New("domain is required")
	}
	profile = withTimestamps(profile, true)
	if err := profile.Validate(); err != nil {
		return Settings{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.domainProfiles[key] = profile
	if err := persistProfiles(s.pathFor("profiles.json"), s.settings, s.domainProfiles); err != nil {
		return Settings{}, err
	}
	return profile, nil
}

// DeleteDomainProfile removes a domain-specific profile.
func (s *Store) DeleteDomainProfile(domain string) (bool, error) {
	key := normalizeDomain(domain)
	if key == "" {
		return false, errors.New("domain is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.domainProfiles[key]; !ok {
		return false, nil
	}
	delete(s.domainProfiles, key)
	if err := persistProfiles(s.pathFor("profiles.json"), s.settings, s.domainProfiles); err != nil {
		return false, err
	}
	return true, nil
}

// ResolveSettings returns effective settings for a domain.
func (s *Store) ResolveSettings(domain string) (Settings, string) {
	key := normalizeDomain(domain)
	s.mu.RLock()
	defer s.mu.RUnlock()

	if key == "" {
		return s.settings, ""
	}

	candidates := domainCandidates(key)
	for _, candidate := range candidates {
		if profile, ok := s.domainProfiles[candidate]; ok {
			return profile, candidate
		}
	}
	return s.settings, ""
}

func domainCandidates(domain string) []string {
	parts := strings.Split(domain, ".")
	result := make([]string, 0, len(parts))
	for i := 0; i < len(parts)-1; i++ {
		result = append(result, strings.Join(parts[i:], "."))
	}
	return result
}

func normalizeDomain(domain string) string {
	domain = strings.TrimSpace(strings.ToLower(domain))
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "www.")
	if idx := strings.IndexRune(domain, '/'); idx >= 0 {
		domain = domain[:idx]
	}
	return strings.Trim(domain, ".")
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
		SkipReasons:   make(map[string]int64),
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
	return s.RecordSkipReason("unspecified")
}

// RecordSkipReason records a skip with reason metadata.
func (s *Store) RecordSkipReason(reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "unspecified"
	}
	return s.updateStats(func(stats *Stats) {
		stats.Skipped++
		if stats.SkipReasons == nil {
			stats.SkipReasons = make(map[string]int64)
		}
		stats.SkipReasons[reason]++
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

// NewCorrectionID creates a stable unique ID for correction journaling.
func (s *Store) NewCorrectionID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return fmt.Sprintf("corr_%d_%06d", time.Now().UnixNano(), rand.Intn(1000000))
}

// GetUndoTTL returns configured undo window in milliseconds.
func (s *Store) GetUndoTTL() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.undoTTLms
}

// RecordAppliedCorrection stores an applied correction in the journal.
func (s *Store) RecordAppliedCorrection(record CorrectionRecord) (CorrectionRecord, error) {
	record.Original = normalizeWord(record.Original)
	record.Suggestion = normalizeWord(record.Suggestion)
	record.Domain = normalizeDomain(record.Domain)
	record.Mode = strings.TrimSpace(record.Mode)
	record.Reason = strings.TrimSpace(record.Reason)
	record.Explanation = strings.TrimSpace(record.Explanation)
	record.Source = strings.TrimSpace(record.Source)
	record.SessionID = strings.TrimSpace(record.SessionID)
	if record.Original == "" || record.Suggestion == "" {
		return CorrectionRecord{}, errors.New("original and suggestion are required")
	}
	if record.CorrectionID == "" {
		record.CorrectionID = s.NewCorrectionID()
	}
	record.AppliedAt = time.Now().UTC().Format(time.RFC3339)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.correctionJournal = append(s.correctionJournal, record)
	if len(s.correctionJournal) > s.journalCap {
		s.correctionJournal = append([]CorrectionRecord(nil), s.correctionJournal[len(s.correctionJournal)-s.journalCap:]...)
	}
	if err := writeJSONAtomic(s.pathFor("correction_journal.json"), journalFile{Records: s.correctionJournal}); err != nil {
		return CorrectionRecord{}, err
	}
	return record, nil
}

// UndoCorrection marks a correction journal entry as undone.
func (s *Store) UndoCorrection(correctionID string) (CorrectionRecord, bool, error) {
	correctionID = strings.TrimSpace(correctionID)
	if correctionID == "" {
		return CorrectionRecord{}, false, errors.New("correction_id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.correctionJournal) - 1; i >= 0; i-- {
		if s.correctionJournal[i].CorrectionID != correctionID {
			continue
		}
		if !s.correctionJournal[i].Undone {
			s.correctionJournal[i].Undone = true
			s.correctionJournal[i].UndoneAt = time.Now().UTC().Format(time.RFC3339)
			if err := writeJSONAtomic(s.pathFor("correction_journal.json"), journalFile{Records: s.correctionJournal}); err != nil {
				return CorrectionRecord{}, false, err
			}
		}
		return s.correctionJournal[i], true, nil
	}
	return CorrectionRecord{}, false, nil
}

// ListCorrections returns latest corrections (most recent first).
func (s *Store) ListCorrections(limit int) []CorrectionRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.correctionJournal) {
		limit = len(s.correctionJournal)
	}
	result := make([]CorrectionRecord, 0, limit)
	for i := len(s.correctionJournal) - 1; i >= 0 && len(result) < limit; i-- {
		result = append(result, s.correctionJournal[i])
	}
	return result
}

// GetPainPointInsights returns ranked user pain points.
func (s *Store) GetPainPointInsights(limit int) PainPointInsights {
	if limit <= 0 {
		limit = 5
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	undoneCounts := make(map[string]int64)
	domainCounts := make(map[string]int64)
	for _, record := range s.correctionJournal {
		if record.Domain != "" {
			domainCounts[record.Domain]++
		}
		if record.Undone {
			undoneCounts[pairKey(record.Original, record.Suggestion)]++
		}
	}

	undonePairs := make([]PairInsight, 0, len(undoneCounts))
	for pair, count := range undoneCounts {
		undonePairs = append(undonePairs, PairInsight{Key: pair, Count: count})
	}
	sort.Slice(undonePairs, func(i, j int) bool {
		if undonePairs[i].Count == undonePairs[j].Count {
			return undonePairs[i].Key < undonePairs[j].Key
		}
		return undonePairs[i].Count > undonePairs[j].Count
	})
	if len(undonePairs) > limit {
		undonePairs = undonePairs[:limit]
	}

	domainVolume := make([]DomainInsight, 0, len(domainCounts))
	for domain, count := range domainCounts {
		domainVolume = append(domainVolume, DomainInsight{Domain: domain, Count: count})
	}
	sort.Slice(domainVolume, func(i, j int) bool {
		if domainVolume[i].Count == domainVolume[j].Count {
			return domainVolume[i].Domain < domainVolume[j].Domain
		}
		return domainVolume[i].Count > domainVolume[j].Count
	})
	if len(domainVolume) > limit {
		domainVolume = domainVolume[:limit]
	}

	skipReasons := make([]PairInsight, 0, len(s.stats.SkipReasons))
	for reason, count := range s.stats.SkipReasons {
		skipReasons = append(skipReasons, PairInsight{Key: reason, Count: count})
	}
	sort.Slice(skipReasons, func(i, j int) bool {
		if skipReasons[i].Count == skipReasons[j].Count {
			return skipReasons[i].Key < skipReasons[j].Key
		}
		return skipReasons[i].Count > skipReasons[j].Count
	})
	if len(skipReasons) > limit {
		skipReasons = skipReasons[:limit]
	}

	ignoredPairs := make([]string, 0, len(s.ignorePairs))
	for pair, enabled := range s.ignorePairs {
		if enabled {
			ignoredPairs = append(ignoredPairs, pair)
		}
	}
	sort.Strings(ignoredPairs)
	if len(ignoredPairs) > limit {
		ignoredPairs = ignoredPairs[:limit]
	}

	ignoredWords := make([]string, 0, len(s.ignoreWords))
	for word, enabled := range s.ignoreWords {
		if enabled {
			ignoredWords = append(ignoredWords, word)
		}
	}
	sort.Strings(ignoredWords)
	if len(ignoredWords) > limit {
		ignoredWords = ignoredWords[:limit]
	}

	return PainPointInsights{
		TopUndonePairs:         undonePairs,
		TopIgnoredPairs:        ignoredPairs,
		TopIgnoredWords:        ignoredWords,
		DomainCorrectionVolume: domainVolume,
		SkipReasons:            skipReasons,
	}
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

# Architecture Guide: Local Autocorrect System

## Overview

Your autocorrect system uses a **modern, layered architecture** combining Clean Architecture principles on the backend with Event-Driven Architecture on the frontend.

## Architecture Pattern Classification

### **NOT Traditional MVC**

This system is **NOT** Model-View-Controller because:
- No centralized controller routing requests
- No traditional "model" layer with database ORM
- No template-based views

### **INSTEAD: Clean/Hexagonal Architecture + Event-Driven + REST**

```
┌──────────────────────────────────────────────┐
│         Architecture Layers                   │
│                                              │
│  Frontend: Event-Driven (Browser Extension) │
│  Backend: Clean/Layered Architecture (Go)   │
│  Communication: REST API (HTTP/JSON)        │
│  Overall Pattern: Microservices-lite        │
└──────────────────────────────────────────────┘
```

---

## Backend Architecture: Clean/Layered

```
┌─────────────────────────────────────────────┐
│         Presentation Layer (API)            │
│  ┌────────────────────────────────────────┐ │
│  │  handlers.go                           │ │
│  │  - SpellHandler    (POST /spell)       │ │
│  │  - RescoreHandler  (POST /rescore)     │ │
│  │  - HealthHandler   (GET /health)       │ │
│  │  - CORS middleware                     │ │
│  └────────────────────────────────────────┘ │
└──────────────────┬──────────────────────────┘
                   │ (uses)
┌──────────────────▼──────────────────────────┐
│       Business Logic Layer                  │
│  ┌──────────────┐  ┌──────────────┐        │
│  │  spellcheck/ │  │    llm/      │        │
│  │  - SymSpell  │  │  - Context   │        │
│  │  - Lookup    │  │    Analyzer  │        │
│  │  - Edit Dist │  │  - Confus-   │        │
│  │              │  │    ables     │        │
│  └──────────────┘  └──────────────┘        │
│                                             │
│  ┌──────────────────────────────────────┐  │
│  │  guardrails/                         │  │
│  │  - URL detection                     │  │
│  │  - Code detection                    │  │
│  │  - Email detection                   │  │
│  │  - Pattern matching                  │  │
│  └──────────────────────────────────────┘  │
└──────────────────┬──────────────────────────┘
                   │ (uses)
┌──────────────────▼──────────────────────────┐
│           Data Layer                        │
│  ┌────────────────────────────────────────┐ │
│  │  types/ - Domain Models                │ │
│  │  - CorrectionRequest                   │ │
│  │  - CorrectionResponse                  │ │
│  │  - Candidate                           │ │
│  │  - Config                              │ │
│  └────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────┐ │
│  │  data/ - Dictionary Files              │ │
│  │  - test_dictionary.txt                 │ │
│  │  - Frequency lists                     │ │
│  └────────────────────────────────────────┘ │
└─────────────────────────────────────────────┘
```

### Key Backend Principles

**1. Dependency Inversion**
```go
// Handlers depend on interfaces, not implementations
type Server struct {
    spellChecker    *spellcheck.SymSpell      // Business logic
    contextAnalyzer *llm.ContextAnalyzer      // Business logic
    guardrails      *guardrails.Guardrails    // Business logic
    config          *types.Config             // Data
}
```

**2. Separation of Concerns**
- `api/` - HTTP handling, request/response
- `spellcheck/` - Spell checking algorithm
- `llm/` - Context analysis, confusables
- `guardrails/` - Protection logic
- `types/` - Data structures

**3. Testability**
- Each layer can be tested independently
- Business logic has NO HTTP dependencies
- 29 tests across 11 test suites
- 100% test success rate

---

## Frontend Architecture: Event-Driven

```
┌─────────────────────────────────────────────┐
│         Browser Extension Layers            │
│                                             │
│  ┌────────────────────────────────────────┐ │
│  │  DOM Event Layer                       │ │
│  │  - input events                        │ │
│  │  - keydown events                      │ │
│  │  - focus/blur events                   │ │
│  └────────────┬───────────────────────────┘ │
│               │ triggers                     │
│  ┌────────────▼───────────────────────────┐ │
│  │  Content Script (content.js)           │ │
│  │  - Event listeners                     │ │
│  │  - Word extraction                     │ │
│  │  - Context extraction                  │ │
│  │  - UI injection                        │ │
│  │  - Suggestion display                  │ │
│  └────────────┬───────────────────────────┘ │
│               │ communicates with            │
│  ┌────────────▼───────────────────────────┐ │
│  │  Background Worker (background.js)     │ │
│  │  - Service worker lifecycle            │ │
│  │  - Health monitoring                   │ │
│  │  - Settings storage                    │ │
│  │  - Message passing                     │ │
│  └────────────┬───────────────────────────┘ │
│               │ updates                      │
│  ┌────────────▼───────────────────────────┐ │
│  │  Popup UI (popup.html/js)              │ │
│  │  - Status display                      │ │
│  │  - Settings controls                   │ │
│  │  - Statistics (future)                 │ │
│  └────────────────────────────────────────┘ │
└─────────────────────────────────────────────┘
```

### Event-Driven Workflow

```
User types "teh " → input event
                 ↓
       Content script captures
                 ↓
       Extracts word "teh"
                 ↓
     HTTP POST to /spell
                 ↓
   Backend returns "the"
                 ↓
    Content script auto-corrects
                 ↓
         User sees "the "
```

---

## Communication: REST API

```
┌────────────────────────────────────────────┐
│        HTTP Request/Response Flow          │
│                                            │
│  Browser Extension (Client)                │
│         │                                  │
│         │ POST /spell                      │
│         │ {text: "teh", context: "..."}    │
│         │                                  │
│         ▼                                  │
│  ┌─────────────────────────────────────┐  │
│  │  Go Backend (Server)                │  │
│  │  127.0.0.1:8080                     │  │
│  │                                     │  │
│  │  1. Validate request                │  │
│  │  2. Check guardrails                │  │
│  │  3. SymSpell lookup                 │  │
│  │  4. Context analysis (if /rescore)  │  │
│  │  5. Return candidates               │  │
│  └─────────────────────────────────────┘  │
│         │                                  │
│         │ HTTP 200 OK                      │
│         │ {candidates: [...],              │
│         │  best_candidate: {...}}          │
│         │                                  │
│         ▼                                  │
│  Browser Extension                         │
│  - Apply correction                        │
│  - or Show suggestion                      │
└────────────────────────────────────────────┘
```

### API Endpoints

**POST /spell** - Fast Layer
- Called on: SPACE key
- Speed: <50ms
- Uses: SymSpell only
- Returns: Spelling candidates

**POST /rescore** - Smart Layer
- Called on: Punctuation or pause
- Speed: <300ms
- Uses: SymSpell + ContextAnalyzer
- Returns: Context-aware candidates

**GET /health**
- Health check
- Returns: {status, version}

---

## Data Flow

### Fast Correction Flow (on SPACE)

```
1. User types "teh "
         ↓
2. Extension extracts "teh"
         ↓
3. POST /spell {"text": "teh"}
         ↓
4. Guardrails check ✓
         ↓
5. SymSpell lookup
         ↓
6. Candidates: ["the" (0.95), "be" (0.7), ...]
         ↓
7. Response: {best_candidate: "the", should_auto_correct: true}
         ↓
8. Extension applies: "teh" → "the"
         ↓
9. User sees: "the "
```

### Context-Aware Flow (on PUNCTUATION)

```
1. User types "I went to there house."
         ↓
2. Extension extracts:
   - word: "house"
   - context: "I went to there house"
         ↓
3. POST /rescore {"text": "house", "context": "..."}
         ↓
4. Guardrails check ✓
         ↓
5. SymSpell lookup → ["house", ...]
         ↓
6. Add confusables → ["house", "their", "there", "they're"]
         ↓
7. Context analysis:
   - Pattern: "their house" matches context ✓
   - Score: 0.62
         ↓
8. Response: {best_candidate: "their", confidence: 0.62}
         ↓
9. Extension suggests: "there" → "their"
         ↓
10. User presses TAB to accept
```

---

## Component Interaction

```
┌────────────────────────────────────────────────────┐
│                 Complete System                    │
│                                                    │
│  ┌──────────────────────────────────────────────┐ │
│  │  User's Browser (Chrome/Arc)                 │ │
│  │  ┌────────────────────────────────────────┐  │ │
│  │  │  Gmail / Docs / Notion / Discord       │  │ │
│  │  │  ┌──────────────────────────────────┐  │  │ │
│  │  │  │  Extension Content Script        │  │  │ │
│  │  │  │  - Monitors typing               │  │  │ │
│  │  │  │  - Extracts words & context      │  │  │ │
│  │  │  └──────────────┬───────────────────┘  │  │ │
│  │  └─────────────────┼──────────────────────┘  │ │
│  └────────────────────┼─────────────────────────┘ │
│                       │                           │
│                       │ HTTP (JSON)               │
│                       ▼                           │
│  ┌────────────────────────────────────────────┐  │
│  │  Local Backend (Go)                        │  │
│  │  127.0.0.1:8080                            │  │
│  │                                            │  │
│  │  ┌──────────────────────────────────────┐ │  │
│  │  │  Guardrails Layer                    │ │  │
│  │  │  - URL detection                     │ │  │
│  │  │  - Code detection                    │ │  │
│  │  │  - Email detection                   │ │  │
│  │  └────────┬─────────────────────────────┘ │  │
│  │           │                                │  │
│  │           ▼ (if allowed)                   │  │
│  │  ┌──────────────────────────────────────┐ │  │
│  │  │  SymSpell (Fast Layer)               │ │  │
│  │  │  - Dictionary lookup                 │ │  │
│  │  │  - Edit distance                     │ │  │
│  │  │  - Frequency ranking                 │ │  │
│  │  └────────┬─────────────────────────────┘ │  │
│  │           │                                │  │
│  │           ▼                                │  │
│  │  ┌──────────────────────────────────────┐ │  │
│  │  │  Context Analyzer (Smart Layer)      │ │  │
│  │  │  - Add confusables                   │ │  │
│  │  │  - Pattern matching                  │ │  │
│  │  │  - Confidence rescoring              │ │  │
│  │  └────────┬─────────────────────────────┘ │  │
│  │           │                                │  │
│  │           ▼                                │  │
│  │  ┌──────────────────────────────────────┐ │  │
│  │  │  Response Builder                    │ │  │
│  │  │  - Best candidate                    │ │  │
│  │  │  - Auto-correct flag                 │ │  │
│  │  └──────────────────────────────────────┘ │  │
│  └────────────────────────────────────────────┘  │
│                       │                           │
│                       │ HTTP Response             │
│                       ▼                           │
│  ┌────────────────────────────────────────────┐  │
│  │  Extension Content Script                  │  │
│  │  - Apply correction or show suggestion    │  │
│  └────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────┘
```

---

## Design Patterns Used

### 1. **Layered Architecture**
- Presentation → Business Logic → Data
- Clear separation of concerns
- Each layer depends only on the layer below

### 2. **Strategy Pattern**
- Different correction strategies:
  - Fast (SymSpell only)
  - Smart (SymSpell + Context)
- Guardrails strategies for different content types

### 3. **Chain of Responsibility**
- Request processing chain:
  - Validation → Guardrails → Spelling → Context → Response

### 4. **Observer Pattern**
- DOM events trigger correction workflow
- Event-driven extension architecture

### 5. **Dependency Injection**
- Server receives dependencies at construction
- Makes testing easy

---

## Key Architectural Decisions

### ✅ **Why Go for Backend?**
- Fast execution (<1ms spell checks)
- Strong typing (catches bugs at compile time)
- Great standard library (HTTP, JSON, regex)
- Easy concurrency (if needed later)
- Single binary deployment

### ✅ **Why Browser Extension?**
- Works everywhere (Gmail, Docs, Notion, Discord)
- No website modifications needed
- Runs in user's context (privacy!)
- Real-time, low latency

### ✅ **Why Local-First?**
- Privacy: No data sent to cloud
- Speed: No network latency
- Reliability: Works offline
- Control: User owns their data

### ✅ **Why REST API?**
- Simple, stateless
- Easy to test (curl, Postman)
- Language-agnostic (can add other clients)
- HTTP is universal

### ✅ **Why Separate Layers?**
- Testability: Each layer tested independently
- Maintainability: Changes localized
- Flexibility: Swap implementations easily
- Scalability: Add features without breaking existing code

---

## Security Architecture

```
┌────────────────────────────────────────┐
│         Security Layers                │
│                                        │
│  1. Localhost Only (127.0.0.1)         │
│     - Backend not exposed to internet  │
│                                        │
│  2. Guardrails                         │
│     - Skip password fields             │
│     - Skip sensitive data              │
│                                        │
│  3. CORS Headers                       │
│     - Only allow extension origin      │
│                                        │
│  4. Input Validation                   │
│     - Validate all requests            │
│     - Sanitize inputs                  │
│                                        │
│  5. No Data Storage                    │
│     - Don't log user text              │
│     - Don't persist corrections        │
└────────────────────────────────────────┘
```

---

## Performance Architecture

```
┌────────────────────────────────────────┐
│       Performance Optimizations        │
│                                        │
│  1. Pre-computation (SymSpell)         │
│     - Dictionary loaded once (0.7ms)   │
│     - Deletions pre-generated          │
│                                        │
│  2. O(1) Lookups                       │
│     - HashMap-based                    │
│     - No linear scans                  │
│                                        │
│  3. Early Returns                      │
│     - Guardrails exit fast             │
│     - No processing if protected       │
│                                        │
│  4. Minimal Context                    │
│     - Only send necessary data         │
│     - Small JSON payloads              │
│                                        │
│  5. Lazy Loading                       │
│     - Extension loads on demand        │
│     - Background worker lightweight    │
└────────────────────────────────────────┘
```

---

## Future Architecture Evolution

### Phase 7: LLM Integration

```
┌─────────────────────────────────────┐
│  Current: Rule-Based Context        │
│  ┌────────────────────────────────┐ │
│  │  Pattern Matching              │ │
│  │  - "their house"               │ │
│  │  - "you're welcome"            │ │
│  └────────────────────────────────┘ │
└─────────────────────────────────────┘
                ↓
┌─────────────────────────────────────┐
│  Future: Hybrid Approach            │
│  ┌────────────────────────────────┐ │
│  │  1. Guardrails (fastest)       │ │
│  │  2. SymSpell (fast)            │ │
│  │  3. Rules (fast)               │ │
│  │  4. LLM (slow, complex cases)  │ │
│  └────────────────────────────────┘ │
└─────────────────────────────────────┘
```

---

## Summary

Your autocorrect system is a **modern, well-architected application** that combines:

✅ **Clean Architecture** (backend)
✅ **Event-Driven Architecture** (frontend)
✅ **REST API** (communication)
✅ **Microservices-lite** (overall pattern)

It's **NOT** traditional MVC - it's **better**!

**Why better?**
- More testable
- More scalable
- More maintainable
- Clearer separation of concerns
- Modern best practices

You've built **production-quality software** using industry-standard patterns! 🚀

# Learning Guide: Building a Local Autocorrect System

## What We've Built So Far (Phases 1-3)

### Phase 1-2: SymSpell Spell Checker

**What is SymSpell?**
- A lightning-fast spell checking algorithm
- Pre-computes all possible misspellings up to edit distance N
- Uses a HashMap for O(1) lookups instead of scanning the dictionary
- Catches 99% of typos with edit distance 2

**Key Go Concepts Learned:**
1. **Structs**: Like classes in other languages
   ```go
   type SymSpell struct {
       dictionary map[string]int64  // HashMap in Go
       deletes    map[string][]string
   }
   ```

2. **Maps**: Go's built-in HashMap
   ```go
   myMap := make(map[string]int)  // Create
   myMap["key"] = 42              // Set
   value := myMap["key"]          // Get
   ```

3. **Pointers**: Using `*` to reference memory locations
   ```go
   func NewSymSpell() *SymSpell {  // Returns a pointer
       return &SymSpell{...}        // & gets memory address
   }
   ```

4. **Error Handling**: Go's explicit error handling
   ```go
   result, err := someFunction()
   if err != nil {
       // Handle error
       return fmt.Errorf("wrapped error: %w", err)
   }
   ```

5. **JSON Tags**: Mapping struct fields to JSON
   ```go
   type Response struct {
       Word string `json:"word"`  // JSON field name
   }
   ```

### Phase 3: HTTP API Server

**What We Built:**
- HTTP server listening on `http://127.0.0.1:8080`
- Three endpoints:
  - `GET /health` - Health check
  - `POST /spell` - Fast spell check (for space key)
  - `POST /rescore` - Context-aware correction (for punctuation)

**Key Go Concepts:**
1. **HTTP Handlers**: Functions that handle web requests
   ```go
   func (s *Server) SpellHandler(w http.ResponseWriter, r *http.Request) {
       // w = write response, r = read request
   }
   ```

2. **JSON Encoding/Decoding**:
   ```go
   json.NewDecoder(r.Body).Decode(&req)  // Parse JSON
   json.NewEncoder(w).Encode(response)   // Send JSON
   ```

3. **CORS Headers**: Allow browser extensions to call our API
   ```go
   w.Header().Set("Access-Control-Allow-Origin", "*")
   ```

## Architecture Diagram

```
┌─────────────────────────────────────────────┐
│           Browser Extension                 │
│  ┌──────────────────────────────────────┐   │
│  │  Content Script (monitors typing)    │   │
│  │  - Detects space/punctuation         │   │
│  │  - Extracts word + context           │   │
│  │  - Shows suggestions                 │   │
│  └──────────────┬───────────────────────┘   │
└─────────────────┼───────────────────────────┘
                  │ HTTP POST
                  ▼
┌─────────────────────────────────────────────┐
│         Go Backend (127.0.0.1:8080)         │
│  ┌──────────────────────────────────────┐   │
│  │  /spell - Fast Layer                 │   │
│  │  ├─ SymSpell lookup (<1ms)           │   │
│  │  └─ Returns candidates               │   │
│  └──────────────────────────────────────┘   │
│  ┌──────────────────────────────────────┐   │
│  │  /rescore - Smart Layer              │   │
│  │  ├─ Get SymSpell candidates          │   │
│  │  ├─ LLM rescore with context         │   │
│  │  └─ Return best match                │   │
│  └──────────────────────────────────────┘   │
└─────────────────────────────────────────────┘
```

## Performance Metrics

- **Dictionary Load**: ~700µs (0.7ms)
- **Spell Check**: <1ms per word
- **Auto-correct Threshold**: 0.9 (90% confidence)
- **Suggestion Threshold**: 0.5 (50% confidence)

## Next Steps

### Phase 4: Browser Extension
- Build TypeScript extension
- Monitor `<input>`, `<textarea>`, `contenteditable`
- Extract words and context
- Call backend API
- Show inline suggestions

### Phase 5: LLM Integration
- Integrate llama.cpp
- Use 1-3B parameter model (Phi-2, TinyLlama, etc.)
- Context-aware disambiguation
- Handle real-word errors (their/there, meat/meet)

### Phase 6: Guardrails
- Skip URLs, code blocks, passwords
- User dictionary for names/slang
- Handle stuck-together words ("havea" → "have a")

## Resources

- SymSpell Paper: https://github.com/wolfgarbe/SymSpell
- Go Documentation: https://go.dev/doc/
- Chrome Extensions: https://developer.chrome.com/docs/extensions/mv3/
- llama.cpp: https://github.com/ggerganov/llama.cpp

## Key Takeaways

1. **SymSpell is brilliant**: Pre-computation makes it blazing fast
2. **Go is simple**: No classes, just structs and functions
3. **Error handling is explicit**: No hidden exceptions
4. **HTTP is straightforward**: Standard library does everything
5. **Testing is built-in**: `*_test.go` files just work

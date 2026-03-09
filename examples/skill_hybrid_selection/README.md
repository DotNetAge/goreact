# Skill Hybrid Selection Example

This example demonstrates the hybrid skill selection mode that combines keyword matching and semantic matching.

## Selection Modes

### 1. Keyword Only (Fast)
- Uses predefined keyword mappings
- Fast but may miss semantic relationships
- No LLM required

### 2. Semantic Only (Accurate)
- Uses LLM to understand task semantics
- Most accurate but slower
- Requires LLM client

### 3. Hybrid (Recommended) ⭐
- First: Fast keyword filtering to get top N candidates
- Then: Semantic matching to select the best one
- Balances speed and accuracy
- Falls back to keyword-only if LLM unavailable

## How It Works

```
Task: "Calculate 25 * 4 + 10"
    ↓
[Step 1: Keyword Filtering]
    → Candidate 1: math-wizard (score: 15.5)
    → Candidate 2: data-analyzer (score: 3.2)
    → Candidate 3: text-processor (score: 1.8)
    ↓
[Step 2: Semantic Selection via LLM]
    → Analyzes task semantics
    → Compares with candidate descriptions
    → Selects: math-wizard ✓
```

## Running the Example

```bash
cd examples/skill_hybrid_selection
go run main.go
```

## Expected Output

- Comparison of all three selection modes
- Performance metrics (speed, accuracy)
- Selected skill for each mode
- Execution results

package reactor

// Session management for Reactor is now handled by LLMCaller.
// This file is retained for future session-related utilities.
//
// All previous methods have been migrated:
//   - ensureContextWindow  → LLMCaller (internal)
//   - persistMessage       → Reactor.persistStepToStore (uses LLMCaller.SessionStore)
//   - checkSlide           → LLMCaller.doSlide (internal, invoked during Call/CallStream)

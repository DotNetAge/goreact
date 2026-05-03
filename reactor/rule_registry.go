package reactor

// DefaultRuleRegistry has been removed. RuleRegistry has no default implementation —
// it is injected by external callers via WithRuleRegistry.
// Users must provide their own core.RuleRegistry implementation when needed.

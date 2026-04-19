package core

type MemoryType int

const (
	MemoryTypeShortTerms MemoryType = iota
	MemoryTypeLongTerms
	MemoryTypeRefactive
)

type Memory interface {
	Search(query string, memType MemoryType) ([]string, error)
	Save(value string, memType MemoryType) error
}

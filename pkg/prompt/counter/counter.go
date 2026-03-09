package counter

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// TokenCounter Token 计数器接口
type TokenCounter interface {
	Count(text string) int
}

// SimpleEstimator 简单估算器（1 token ≈ 4 chars）
type SimpleEstimator struct{}

func NewSimpleEstimator() *SimpleEstimator {
	return &SimpleEstimator{}
}

func (e *SimpleEstimator) Count(text string) int {
	return len(text) / 4
}

// UniversalEstimator 通用估算器（基于语言特征）
type UniversalEstimator struct {
	language string // "en", "zh", "mixed"
}

func NewUniversalEstimator(language string) *UniversalEstimator {
	if language == "" {
		language = "mixed"
	}
	return &UniversalEstimator{language: language}
}

func (e *UniversalEstimator) Count(text string) int {
	switch e.language {
	case "en":
		return e.countEnglish(text)
	case "zh":
		return e.countChinese(text)
	default:
		return e.countMixed(text)
	}
}

// countEnglish 英文 token 估算
func (e *UniversalEstimator) countEnglish(text string) int {
	// 英文：按空格分词，平均每个词 1.3 tokens
	words := strings.Fields(text)
	wordTokens := int(float64(len(words)) * 1.3)

	// 标点符号和特殊字符
	specialChars := regexp.MustCompile(`[^\w\s]`).FindAllString(text, -1)
	specialTokens := len(specialChars)

	return wordTokens + specialTokens
}

// countChinese 中文 token 估算
func (e *UniversalEstimator) countChinese(text string) int {
	// 中文：每个汉字约 1.5-2 tokens
	chineseChars := 0
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF {
			chineseChars++
		}
	}

	// 其他字符（英文、数字、标点）
	otherChars := utf8.RuneCountInString(text) - chineseChars

	return int(float64(chineseChars)*1.8) + otherChars/4
}

// countMixed 混合语言 token 估算
func (e *UniversalEstimator) countMixed(text string) int {
	chineseTokens := 0
	englishChars := 0
	otherChars := 0

	inWord := false
	wordStart := 0

	runes := []rune(text)
	for i, r := range runes {
		if r >= 0x4E00 && r <= 0x9FFF {
			// 中文字符
			if inWord {
				// 结束英文单词
				word := string(runes[wordStart:i])
				englishChars += len(strings.Fields(word))
				inWord = false
			}
			chineseTokens += 2 // 每个中文字符约 2 tokens
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			// 英文字符或数字
			if !inWord {
				inWord = true
				wordStart = i
			}
		} else {
			// 其他字符（空格、标点等）
			if inWord {
				word := string(runes[wordStart:i])
				englishChars += len(strings.Fields(word))
				inWord = false
			}
			otherChars++
		}
	}

	// 处理最后一个单词
	if inWord {
		word := string(runes[wordStart:])
		englishChars += len(strings.Fields(word))
	}

	// 英文单词约 1.3 tokens/word，其他字符约 0.5 tokens/char
	return chineseTokens + int(float64(englishChars)*1.3) + otherChars/2
}

// CachedTokenCounter 带缓存的 Token 计数器
type CachedTokenCounter struct {
	counter TokenCounter
	cache   map[string]int
	maxSize int
}

func NewCachedTokenCounter(counter TokenCounter, maxSize int) *CachedTokenCounter {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &CachedTokenCounter{
		counter: counter,
		cache:   make(map[string]int),
		maxSize: maxSize,
	}
}

func (c *CachedTokenCounter) Count(text string) int {
	// 检查缓存
	if count, ok := c.cache[text]; ok {
		return count
	}

	// 计算
	count := c.counter.Count(text)

	// 缓存（如果未满）
	if len(c.cache) < c.maxSize {
		c.cache[text] = count
	}

	return count
}

// Clear 清空缓存
func (c *CachedTokenCounter) Clear() {
	c.cache = make(map[string]int)
}

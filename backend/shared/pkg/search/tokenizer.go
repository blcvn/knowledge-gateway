package search

import (
    "strings"
    "unicode"
)

var stopwords = map[string]bool{
    "the": true, "is": true, "at": true, "which": true, "on": true,
    "a": true, "an": true, "and": true, "or": true, "but": true,
    "in": true, "of": true, "to": true, "for": true, "with": true,
    "this": true, "that": true, "have": true, "from": true, "be": true,
}

// Tokenize splits text into normalized tokens for BM25 indexing
// Supports: ASCII word splitting + CJK bigrams
func Tokenize(text string) []string {
    text = strings.ToLower(text)
    var tokens []string

    // ASCII word splitting + simple stemming
    words := strings.FieldsFunc(text, func(r rune) bool {
        return !unicode.IsLetter(r) && !unicode.IsDigit(r)
    })
    for _, w := range words {
        stemmed := porterStem(w)
        if len(stemmed) >= 2 && !stopwords[stemmed] {
            tokens = append(tokens, stemmed)
        }
    }

    // CJK bigram tokenization
    tokens = append(tokens, cjkBigrams(text)...)
    return tokens
}

// cjkBigrams generates bigrams from CJK unicode characters
func cjkBigrams(text string) []string {
    runes := []rune(text)
    var bigrams []string
    for i := 0; i < len(runes)-1; i++ {
        if isCJK(runes[i]) && isCJK(runes[i+1]) {
            bigrams = append(bigrams, string(runes[i:i+2]))
        }
    }
    return bigrams
}

func isCJK(r rune) bool {
    return (r >= 0x4E00 && r <= 0x9FFF) ||  // CJK Unified
           (r >= 0x3400 && r <= 0x4DBF) ||  // Extension A
           (r >= 0xAC00 && r <= 0xD7AF)     // Hangul
}

// Simple Porter stemmer (first 3 rules only for brevity)
func porterStem(word string) string {
    if len(word) <= 3 { return word }
    if strings.HasSuffix(word, "ing") && len(word) > 5 { return word[:len(word)-3] }
    if strings.HasSuffix(word, "ed") && len(word) > 4 { return word[:len(word)-2] }
    if strings.HasSuffix(word, "es") && len(word) > 3 { return word[:len(word)-2] }
    if strings.HasSuffix(word, "s") && len(word) > 3 { return word[:len(word)-1] }
    return word
}

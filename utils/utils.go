package utils

import (
	"strings"
)

// UniqueWords - делаем массив с уникальными строками
func UniqueWords(words []string) []string {
	uniqWords := make(map[string]struct{})
	for _, word := range words {
		uniqWords[strings.TrimSpace(word)] = struct{}{}
	}
	n := len(uniqWords)
	words = words[:n:n]
	i := 0
	for word := range uniqWords {
		words[i] = word
		i++
	}
	return words
}

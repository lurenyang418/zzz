package main

import (
	"fmt"
	"regexp"
	"strings"
)

type Filter struct {
	ignore  []*regexp.Regexp
	include []*regexp.Regexp
}

func defaultFilter() (*Filter, error) {
	return newFilter(
		[]string{"node_modules/", ".git/", "dist/", "build/", ".DS_Store", ".env", ".*ignore"},
		nil,
	)
}

func newFilter(ignorePatterns, includePatterns []string) (*Filter, error) {
	ignore, err := compilePatterns(ignorePatterns)
	if err != nil {
		return nil, err
	}
	include, err := compilePatterns(includePatterns)
	if err != nil {
		return nil, err
	}
	return &Filter{ignore: ignore, include: include}, nil
}

func (f *Filter) shouldIgnore(path string) bool {
	if len(f.include) > 0 && !matchesAny(f.include, path) {
		return true
	}
	return matchesAny(f.ignore, path)
}

func (f *Filter) shouldPruneDirectory(path string) bool {
	return matchesAny(f.ignore, path) || matchesAny(f.ignore, path+"/placeholder")
}

func matchesAny(patterns []*regexp.Regexp, value string) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	return false
}

func compilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, raw := range patterns {
		pattern := strings.TrimSpace(raw)
		if pattern == "" {
			continue
		}
		normalized := strings.TrimSuffix(pattern, "/")
		if strings.HasSuffix(pattern, "/") || (!strings.Contains(normalized, "/") && !strings.Contains(normalized, ".")) {
			normalized += "/**"
		}
		expression, err := globRegexp(normalized)
		if err != nil {
			return nil, &APIError{Status: 400, Message: fmt.Sprintf("invalid filter rule %q: %v", pattern, err)}
		}
		compiled = append(compiled, expression)
	}
	return compiled, nil
}

func globRegexp(pattern string) (*regexp.Regexp, error) {
	var builder strings.Builder
	builder.WriteString("^")
	for index := 0; index < len(pattern); index++ {
		switch pattern[index] {
		case '*':
			builder.WriteString(".*")
		case '?':
			builder.WriteByte('.')
		case '[':
			end := strings.IndexByte(pattern[index+1:], ']')
			if end < 0 {
				return nil, fmt.Errorf("unterminated character class")
			}
			end += index + 1
			class := pattern[index : end+1]
			if _, err := regexp.Compile("^(?:" + class + ")$"); err != nil {
				return nil, fmt.Errorf("invalid character class")
			}
			builder.WriteString(class)
			index = end
		default:
			builder.WriteString(regexp.QuoteMeta(string(pattern[index])))
		}
	}
	builder.WriteString("$")
	return regexp.Compile(builder.String())
}

func splitPatterns(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			result = append(result, value)
		}
	}
	return result
}

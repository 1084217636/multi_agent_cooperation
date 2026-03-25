package rag

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var wordRegexp = regexp.MustCompile(`[A-Za-z0-9_./:-]+`)

// Options 控制索引范围。
type Options struct {
	IncludePaths     []string
	ExcludeNames     []string
	MaxFileSizeBytes int64
	ChunkSize        int
}

// Result 表示一次检索命中。
type Result struct {
	Path    string  `json:"path"`
	Title   string  `json:"title"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

type chunk struct {
	path    string
	title   string
	content string
	terms   map[string]int
}

// Index 是一个轻量级、无需 embedding 的本地 RAG 索引。
type Index struct {
	root   string
	chunks []chunk
	files  int
}

// Build 在给定目录构建检索索引。
func Build(root string, options Options) (*Index, error) {
	idx := &Index{root: root}
	if options.MaxFileSizeBytes <= 0 {
		options.MaxFileSizeBytes = 512 * 1024
	}
	if options.ChunkSize <= 0 {
		options.ChunkSize = 700
	}

	for _, include := range options.IncludePaths {
		absPath := include
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(root, include)
		}

		info, err := os.Stat(absPath)
		if err != nil {
			continue
		}

		if info.IsDir() {
			if err := filepath.Walk(absPath, func(path string, fileInfo os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return nil
				}
				if fileInfo.IsDir() {
					if shouldSkip(fileInfo.Name(), options.ExcludeNames) {
						return filepath.SkipDir
					}
					return nil
				}
				idx.addFile(path, fileInfo, options)
				return nil
			}); err != nil {
				return nil, err
			}
			continue
		}

		idx.addFile(absPath, info, options)
	}

	return idx, nil
}

// FileCount 返回被索引的文件数。
func (i *Index) FileCount() int {
	return i.files
}

// ChunkCount 返回被索引的切片数。
func (i *Index) ChunkCount() int {
	return len(i.chunks)
}

// Search 检索与 query 最相关的若干切片。
func (i *Index) Search(query string, limit int) []Result {
	if limit <= 0 {
		limit = 5
	}

	queryTerms := tokenFrequency(query)
	if len(queryTerms) == 0 {
		return nil
	}

	type scored struct {
		Result
	}

	var results []scored
	for _, chunk := range i.chunks {
		score := scoreChunk(queryTerms, chunk)
		if score <= 0 {
			continue
		}

		results = append(results, scored{
			Result: Result{
				Path:    chunk.path,
				Title:   chunk.title,
				Content: chunk.content,
				Score:   score,
			},
		})
	}

	sort.Slice(results, func(a, b int) bool {
		if results[a].Score == results[b].Score {
			return results[a].Path < results[b].Path
		}
		return results[a].Score > results[b].Score
	})

	if len(results) > limit {
		results = results[:limit]
	}

	final := make([]Result, 0, len(results))
	for _, item := range results {
		final = append(final, item.Result)
	}
	return final
}

func (i *Index) addFile(path string, info os.FileInfo, options Options) {
	if !isTextLike(path) || info.Size() > options.MaxFileSizeBytes {
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return
	}

	relPath, err := filepath.Rel(i.root, path)
	if err != nil {
		relPath = path
	}

	i.files++
	for _, part := range splitContent(content, options.ChunkSize) {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		i.chunks = append(i.chunks, chunk{
			path:    relPath,
			title:   filepath.Base(path),
			content: trimmed,
			terms:   tokenFrequency(trimmed + " " + relPath),
		})
	}
}

func splitContent(content string, chunkSize int) []string {
	if len([]rune(content)) <= chunkSize {
		return []string{content}
	}

	blocks := strings.Split(content, "\n\n")
	var chunks []string
	var current strings.Builder

	for _, block := range blocks {
		trimmed := strings.TrimSpace(block)
		if trimmed == "" {
			continue
		}
		next := current.String()
		if len([]rune(next))+len([]rune(trimmed)) > chunkSize && current.Len() > 0 {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			current.Reset()
		}
		current.WriteString(trimmed)
		current.WriteString("\n\n")
	}

	if current.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}

	if len(chunks) == 0 {
		return []string{content}
	}
	return chunks
}

func scoreChunk(queryTerms map[string]int, candidate chunk) float64 {
	var score float64
	for term, weight := range queryTerms {
		freq := candidate.terms[term]
		if freq == 0 {
			continue
		}
		score += float64(freq * weight)
	}

	if score == 0 {
		return 0
	}

	if strings.Contains(strings.ToLower(candidate.path), "readme") || strings.Contains(strings.ToLower(candidate.path), "doc") {
		score *= 1.1
	}

	return score / float64(len([]rune(candidate.content))+40)
}

func tokenFrequency(input string) map[string]int {
	freq := map[string]int{}
	for _, token := range tokenize(input) {
		freq[token]++
	}
	return freq
}

func tokenize(input string) []string {
	lower := strings.ToLower(input)
	seen := []string{}

	for _, token := range wordRegexp.FindAllString(lower, -1) {
		if len(token) > 1 {
			seen = append(seen, token)
		}
	}

	var hanRunes []rune
	for _, r := range []rune(lower) {
		if unicode.Is(unicode.Han, r) {
			hanRunes = append(hanRunes, r)
			seen = append(seen, string(r))
		}
	}

	for idx := 0; idx < len(hanRunes)-1; idx++ {
		seen = append(seen, string([]rune{hanRunes[idx], hanRunes[idx+1]}))
	}

	return seen
}

func shouldSkip(name string, excludes []string) bool {
	for _, exclude := range excludes {
		if name == exclude {
			return true
		}
	}
	return false
}

func isTextLike(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".md", ".txt", ".yaml", ".yml", ".json", ".toml":
		return true
	default:
		return false
	}
}

package companion

import (
	"net/url"
	"path/filepath"
	"sort"
	"strings"
)

func (s *Service) projectRoot() string {
	return s.root
}

func (s *Service) generatedRoot() string {
	return s.config.App.GeneratedRoot
}

func (s *Service) analysisScopeEntries() []string {
	items := make([]string, 0, len(s.config.Knowledge.IncludePaths))
	for _, path := range s.config.Knowledge.IncludePaths {
		items = append(items, displayPathFromRoot(s.root, path))
	}
	sort.Strings(items)
	return uniqueScopeStrings(items)
}

func (s *Service) projectExcludeNames() []string {
	items := append([]string{}, s.config.Knowledge.ExcludeNames...)
	items = append(items, firstScopedComponent(s.root, s.config.App.DataDir))
	items = append(items, firstScopedComponent(s.root, s.config.App.GeneratedRoot))
	items = append(items, ".git", ".vscode", "bin", "node_modules", "accounts")
	sort.Strings(items)
	return uniqueScopeStrings(items)
}

func (s *Service) generatedFileURL(targetMode, outputDir, relativePath string) string {
	relativePath, ok := sanitizeRelativePath(relativePath)
	if !ok {
		return ""
	}

	if targetMode == "current_repo_patch" {
		return joinURLPath("/project-files", relativePath)
	}

	fullPath := filepath.Join(outputDir, filepath.FromSlash(relativePath))
	rel, err := filepath.Rel(s.generatedRoot(), fullPath)
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || rel == "" || strings.HasPrefix(rel, "../") || rel == ".." {
		return ""
	}
	return joinURLPath("/generated-files", rel)
}

func displayPathFromRoot(root, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if rel, err := filepath.Rel(root, path); err == nil {
		rel = filepath.ToSlash(rel)
		switch {
		case rel == ".":
			return "."
		case rel != ".." && !strings.HasPrefix(rel, "../"):
			return rel
		}
	}
	return path
}

func firstScopedComponent(root, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || rel == "" || rel == ".." || strings.HasPrefix(rel, "../") {
		return ""
	}
	parts := strings.Split(rel, "/")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func joinURLPath(base, rel string) string {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" {
		return base
	}
	parts := strings.Split(rel, "/")
	for idx, part := range parts {
		parts[idx] = url.PathEscape(part)
	}
	return strings.TrimRight(base, "/") + "/" + strings.Join(parts, "/")
}

func uniqueScopeStrings(items []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

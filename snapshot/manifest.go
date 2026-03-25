package snapshot

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FileRecord 描述快照中的单个文件。
type FileRecord struct {
	Path   string    `json:"path"`
	SHA256 string    `json:"sha256"`
	Size   int64     `json:"size"`
	Mode   string    `json:"mode"`
	ModAt  time.Time `json:"mod_at"`
	Data   string    `json:"data,omitempty"`
}

// Manifest 是一次工作区快照。
type Manifest struct {
	ID        string       `json:"id"`
	Root      string       `json:"root"`
	CreatedAt time.Time    `json:"created_at"`
	Files     []FileRecord `json:"files"`
}

// Diff 表示两次快照间的差异。
type Diff struct {
	Added    []string `json:"added"`
	Modified []string `json:"modified"`
	Deleted  []string `json:"deleted"`
}

// Capture 对工作区创建快照。
func Capture(root string, excludeNames []string) (*Manifest, error) {
	manifest := &Manifest{
		ID:        time.Now().Format("20060102150405"),
		Root:      root,
		CreatedAt: time.Now(),
		Files:     []FileRecord{},
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		name := info.Name()
		if info.IsDir() {
			if shouldSkipDir(name, excludeNames) {
				return filepath.SkipDir
			}
			return nil
		}

		if !isSnapshotCandidate(path) {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		hash := sha256.Sum256(data)
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			relPath = path
		}

		manifest.Files = append(manifest.Files, FileRecord{
			Path:   filepath.ToSlash(relPath),
			SHA256: hex.EncodeToString(hash[:]),
			Size:   info.Size(),
			Mode:   info.Mode().String(),
			ModAt:  info.ModTime(),
			Data:   base64.StdEncoding.EncodeToString(data),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(manifest.Files, func(i, j int) bool {
		return manifest.Files[i].Path < manifest.Files[j].Path
	})

	return manifest, nil
}

// Save 将快照写入 JSON 文件。
func Save(manifest *Manifest, path string) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Load 从 JSON 文件读取快照。
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// Compare 计算两次快照差异。
func Compare(before, after *Manifest) Diff {
	if before == nil || after == nil {
		return Diff{}
	}

	beforeMap := map[string]FileRecord{}
	afterMap := map[string]FileRecord{}
	for _, file := range before.Files {
		beforeMap[file.Path] = file
	}
	for _, file := range after.Files {
		afterMap[file.Path] = file
	}

	diff := Diff{}
	for path, beforeFile := range beforeMap {
		afterFile, ok := afterMap[path]
		if !ok {
			diff.Deleted = append(diff.Deleted, path)
			continue
		}
		if beforeFile.SHA256 != afterFile.SHA256 {
			diff.Modified = append(diff.Modified, path)
		}
	}

	for path := range afterMap {
		if _, ok := beforeMap[path]; !ok {
			diff.Added = append(diff.Added, path)
		}
	}

	sort.Strings(diff.Added)
	sort.Strings(diff.Modified)
	sort.Strings(diff.Deleted)
	return diff
}

// Restore 使用 before 快照恢复工作区。
func Restore(root string, before *Manifest, diff Diff) error {
	if before == nil {
		return nil
	}

	for _, path := range diff.Added {
		_ = os.Remove(filepath.Join(root, filepath.FromSlash(path)))
	}

	for _, file := range before.Files {
		decoded, err := base64.StdEncoding.DecodeString(file.Data)
		if err != nil {
			return err
		}

		fullPath := filepath.Join(root, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(fullPath, decoded, 0o644); err != nil {
			return err
		}
	}

	return nil
}

func shouldSkipDir(name string, excludes []string) bool {
	for _, exclude := range excludes {
		if name == exclude {
			return true
		}
	}
	return false
}

func isSnapshotCandidate(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".md", ".yaml", ".yml", ".json", ".toml", ".txt", ".html", ".js", ".css":
		return true
	default:
		return false
	}
}

package companion

import (
	"path/filepath"
	"sort"
	"strings"
)

// resolveValidationTargets 根据代码生成结果推导本轮应该验证的包范围。
func resolveValidationTargets(root, executionRoot string, report CodeGenerationReport) []string {
	targetSet := map[string]struct{}{}

	addFromRelativePath := func(path string) {
		path = strings.TrimSpace(path)
		if !strings.EqualFold(filepath.Ext(path), ".go") {
			return
		}
		dir := filepath.ToSlash(filepath.Dir(path))
		switch dir {
		case ".", "":
			targetSet["."] = struct{}{}
		default:
			targetSet["./"+dir] = struct{}{}
		}
	}

	for _, file := range report.Files {
		addFromRelativePath(file.Path)
	}

	// 当前仓库 patch 若没有成功写出 Go 文件，则回退到候选文件所在包。
	if len(targetSet) == 0 && report.TargetMode == "current_repo_patch" && executionRoot == root {
		for _, candidate := range report.PatchCandidates {
			addFromRelativePath(candidate)
		}
	}

	if len(targetSet) == 0 {
		return nil
	}

	targets := make([]string, 0, len(targetSet))
	for target := range targetSet {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	return targets
}

func validationScopeLabel(targets []string) string {
	switch len(targets) {
	case 0:
		return "./..."
	case 1:
		return targets[0]
	case 2:
		return targets[0] + ", " + targets[1]
	default:
		return targets[0] + ", " + targets[1] + " ..."
	}
}

func validationArgs(targets []string) []string {
	if len(targets) == 0 {
		return []string{"./..."}
	}
	return append([]string{}, targets...)
}

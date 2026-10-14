package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo"
)

// textExtensions are the files Batch File Cleaning picks up by default.
var textExtensions = []string{"txt", "text", "md", "markdown", "csv", "tsv", "html", "htm", "xml", "json", "yaml", "yml",
	"log", "srt", "vtt", "ini", "conf", "tex", "rst", "org", "css", "js", "ts", "go", "py", "rb", "sh", "sql", "plist"}

// Batch cleans files on disk (TextSoap's Batch File Cleaning).
type Batch struct{}

// BatchItem is a file or folder to clean.
type BatchItem struct {
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
}

// BatchProgress reports one processed file.
type BatchProgress struct {
	Done   int    `json:"done"`
	Total  int    `json:"total"`
	Source string `json:"source"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

// BatchSummary is the outcome of a run.
type BatchSummary struct {
	Cleaned int `json:"cleaned"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// Choose asks for files and folders to clean.
func (Batch) Choose(ctx context.Context, folders bool) ([]BatchItem, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent: mygo.CallerWindow(ctx), Title: "选择要处理的文件", Multiple: true, Directory: folders,
	})
	if err != nil {
		return nil, err
	}
	return Batch{}.Describe(paths), nil
}

// Describe tells files from folders, for dropped paths.
func (Batch) Describe(paths []string) []BatchItem {
	out := []BatchItem{}
	for _, p := range paths {
		if st, err := os.Stat(p); err == nil {
			out = append(out, BatchItem{Path: p, IsDir: st.IsDir()})
		}
	}
	return out
}

// ChooseDestination asks for the output folder.
func (Batch) ChooseDestination(ctx context.Context) (string, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent: mygo.CallerWindow(ctx), Title: "选择输出文件夹", Directory: true, CreateDirectories: true,
	})
	if err != nil || len(paths) == 0 {
		return "", err
	}
	return paths[0], nil
}

// Reveal shows a file in the file manager.
func (Batch) Reveal(path string) { mygo.Shell.ShowItemInFolder(path) }

func (s engineBatch) wants(path string) bool {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	var extra []string
	for _, e := range strings.FieldsFunc(s.Extensions, func(r rune) bool { return r == ',' || r == ' ' || r == ';' }) {
		extra = append(extra, strings.ToLower(strings.TrimPrefix(e, ".")))
	}
	if s.ExtMode == "only" {
		return slices.Contains(extra, ext)
	}
	return slices.Contains(textExtensions, ext) || slices.Contains(extra, ext)
}

type engineBatch engine.BatchSettings

// output is where a cleaned copy of path goes. A relative destination is
// a folder next to each file.
func (s engineBatch) output(path string) string {
	dir := filepath.Dir(path)
	switch {
	case s.DestFolder == "":
	case filepath.IsAbs(s.DestFolder):
		dir = s.DestFolder
	default:
		dir = filepath.Join(dir, s.DestFolder)
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(filepath.Base(path), ext)
	if s.NewExt != "" {
		ext = "." + strings.TrimPrefix(s.NewExt, ".")
	}
	return filepath.Join(dir, s.Prefix+base+s.Suffix+ext)
}

// Run cleans every file, writing cleaned copies, and streams progress.
func (Batch) Run(ctx context.Context, settings engine.BatchSettings, items []BatchItem, progress *mygo.Channel[BatchProgress]) (BatchSummary, error) {
	s := engineBatch(settings)
	if settings.Cleaner == "" {
		return BatchSummary{}, errors.New("请选择一个清理器")
	}
	if settings.DestFolder == "" && settings.Prefix == "" && settings.Suffix == "" && settings.NewExt == "" {
		return BatchSummary{}, errors.New("输出会覆盖原文件：请设置输出文件夹、前缀、后缀或新扩展名之一")
	}
	LibraryService{}.SaveBatch(settings)
	var sum BatchSummary
	var files []string
	for _, it := range items {
		if !it.IsDir {
			files = append(files, it.Path)
			continue
		}
		root := it.Path
		_ = filepath.WalkDir(root, func(p string, de os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if de.IsDir() {
				if p != root && (!settings.IncludeSubs || strings.HasPrefix(de.Name(), ".") ||
					(!filepath.IsAbs(settings.DestFolder) && de.Name() == settings.DestFolder)) {
					return filepath.SkipDir
				}
				return nil
			}
			if s.wants(p) && !strings.HasPrefix(de.Name(), ".") {
				files = append(files, p)
			} else {
				sum.Skipped++
			}
			return nil
		})
	}
	for i, f := range files {
		if ctx.Err() != nil {
			return sum, ctx.Err()
		}
		pr := BatchProgress{Done: i + 1, Total: len(files), Source: f}
		data, err := os.ReadFile(f)
		if err == nil {
			out := Soap{}.Apply(settings.Cleaner, decodeText(data)).Text
			dst := s.output(f)
			if filepath.Clean(dst) == filepath.Clean(f) {
				err = errors.New("输出路径与原文件相同，已跳过")
			} else if err = os.MkdirAll(filepath.Dir(dst), 0o755); err == nil {
				err = os.WriteFile(dst, []byte(out), 0o644)
			}
			pr.Output = dst
		}
		if err != nil {
			pr.Error = err.Error()
			sum.Failed++
		} else {
			sum.Cleaned++
		}
		if progress != nil {
			if progress.Send(pr) != nil {
				return sum, nil
			}
		}
	}
	return sum, nil
}

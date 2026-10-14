package engine

import (
	"os"
	"path/filepath"
	"strings"
)

// FileResult is the outcome of processing one file in batch mode.
type FileResult struct {
	Path    string
	Changed bool
	Before  int // rune counts, for the summary
	After   int
	Err     error
}

// ProcessFiles runs fn over each text file. With write false it only
// reports what would change. With backup true, "<name>.bak" is written
// before the file is overwritten.
func ProcessFiles(paths []string, fn func(string) string, write, backup bool) []FileResult {
	var out []FileResult
	for _, p := range paths {
		r := FileResult{Path: p}
		data, err := os.ReadFile(p)
		if err != nil {
			r.Err = err
			out = append(out, r)
			continue
		}
		in := string(data)
		res := fn(in)
		r.Before, r.After, r.Changed = RuneLen(in), RuneLen(res), res != in
		if write && r.Changed {
			if backup {
				r.Err = os.WriteFile(p+".bak", data, 0o644)
			}
			if r.Err == nil {
				r.Err = os.WriteFile(p, []byte(res), 0o644)
			}
		}
		out = append(out, r)
	}
	return out
}

// ExpandPaths turns folders into the text files inside them (recursively),
// skipping hidden files and common binary extensions.
func ExpandPaths(paths []string) []string {
	var out []string
	skip := map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".pdf": true, ".zip": true, ".dmg": true,
		".app": true, ".bak": true, ".exe": true, ".so": true, ".dylib": true, ".heic": true, ".mov": true, ".mp4": true, ".mp3": true}
	for _, p := range paths {
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		if !st.IsDir() {
			out = append(out, p)
			continue
		}
		_ = filepath.WalkDir(p, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			name := d.Name()
			if strings.HasPrefix(name, ".") && path != p {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if !d.IsDir() && !skip[strings.ToLower(filepath.Ext(name))] {
				out = append(out, path)
			}
			return nil
		})
	}
	return out
}

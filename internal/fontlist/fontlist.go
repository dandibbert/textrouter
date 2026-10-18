// Package fontlist finds the font families installed on the system by
// reading the name, OS/2 and post tables of the font files in the system's
// font folders. It needs no Core Text or fontconfig.
package fontlist

import (
	"encoding/binary"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"unicode/utf16"
)

// Family is an installed font family.
type Family struct {
	Name      string // the family name, as Core Text and other shapers know it
	LocalName string // the Chinese name, when the font has one, as "苹方-简"
	Mono      bool   // fixed pitch
	CJK       bool   // covers Chinese or Japanese (by its OS/2 code pages)
	Files     int
}

// Dirs are the folders scanned on this system.
func Dirs() []string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return []string{"/System/Library/Fonts", "/Library/Fonts", filepath.Join(home, "Library/Fonts")}
	case "windows":
		return []string{filepath.Join(os.Getenv("WINDIR"), "Fonts"), filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft/Windows/Fonts")}
	default:
		return []string{"/usr/share/fonts", "/usr/local/share/fonts", filepath.Join(home, ".local/share/fonts"), filepath.Join(home, ".fonts")}
	}
}

var (
	once   sync.Once
	cached []Family
)

// System returns the installed families, sorted by name; scanned once.
func System() []Family {
	once.Do(func() { cached = Scan(Dirs()...) })
	return cached
}

// Scan reads the fonts in dirs (recursively) and returns their families.
func Scan(dirs ...string) []Family {
	byName := map[string]*Family{}
	for _, d := range dirs {
		filepath.WalkDir(d, func(p string, e fs.DirEntry, err error) error {
			if err != nil || e.IsDir() {
				return nil
			}
			switch strings.ToLower(filepath.Ext(p)) {
			case ".ttf", ".otf", ".ttc", ".otc":
			default:
				return nil
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			faces, _ := Parse(data)
			for _, f := range faces {
				if f.Name == "" || strings.HasPrefix(f.Name, ".") {
					continue // hidden system families, as .SF NS
				}
				g := byName[f.Name]
				if g == nil {
					g = &Family{Name: f.Name}
					byName[f.Name] = g
				}
				g.Files++
				g.Mono = g.Mono || f.Mono
				g.CJK = g.CJK || f.CJK
				if g.LocalName == "" {
					g.LocalName = f.LocalName
				}
			}
			return nil
		})
	}
	out := make([]Family, 0, len(byName))
	for _, f := range byName {
		out = append(out, *f)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out
}

// Parse reads the faces of a TrueType/OpenType font or collection.
func Parse(data []byte) ([]Family, error) {
	if len(data) < 12 {
		return nil, errors.New("short font")
	}
	if string(data[:4]) == "ttcf" {
		n := int(binary.BigEndian.Uint32(data[8:]))
		if n > 256 || 12+4*n > len(data) {
			return nil, errors.New("bad collection")
		}
		var out []Family
		for i := 0; i < n; i++ {
			off := int(binary.BigEndian.Uint32(data[12+4*i:]))
			if f, err := parseFace(data, off); err == nil {
				out = append(out, f)
			}
		}
		return out, nil
	}
	f, err := parseFace(data, 0)
	if err != nil {
		return nil, err
	}
	return []Family{f}, nil
}

func parseFace(data []byte, off int) (Family, error) {
	var f Family
	if off+12 > len(data) {
		return f, errors.New("bad offset")
	}
	n := int(binary.BigEndian.Uint16(data[off+4:]))
	tables := map[string][]byte{}
	for i := 0; i < n; i++ {
		r := off + 12 + 16*i
		if r+16 > len(data) {
			break
		}
		tag := string(data[r : r+4])
		o, l := int(binary.BigEndian.Uint32(data[r+8:])), int(binary.BigEndian.Uint32(data[r+12:]))
		if o >= 0 && l >= 0 && o+l <= len(data) {
			tables[tag] = data[o : o+l]
		}
	}
	name := tables["name"]
	if name == nil {
		return f, errors.New("no name table")
	}
	f.Name, f.LocalName = names(name)
	if p := tables["post"]; len(p) >= 16 {
		f.Mono = binary.BigEndian.Uint32(p[12:]) != 0
	}
	if o := tables["OS/2"]; len(o) >= 86 {
		cp1 := binary.BigEndian.Uint32(o[78:])
		// Code page bits: 17 JIS, 18 PRC (GB 2312), 19 Korean, 20 Big5.
		f.CJK = cp1&(1<<17|1<<18|1<<20) != 0
	}
	return f, nil
}

// names returns the English family name (typographic family, ID 16, else
// ID 1) and a Chinese one when present.
func names(t []byte) (en, zh string) {
	if len(t) < 6 {
		return
	}
	count, strOff := int(binary.BigEndian.Uint16(t[2:])), int(binary.BigEndian.Uint16(t[4:]))
	best, bestZh := map[int]string{}, map[int]string{}
	for i := 0; i < count; i++ {
		r := 6 + 12*i
		if r+12 > len(t) {
			break
		}
		plat := binary.BigEndian.Uint16(t[r:])
		enc := binary.BigEndian.Uint16(t[r+2:])
		lang := binary.BigEndian.Uint16(t[r+4:])
		id := int(binary.BigEndian.Uint16(t[r+6:]))
		l, o := int(binary.BigEndian.Uint16(t[r+8:])), int(binary.BigEndian.Uint16(t[r+10:]))
		if id != 1 && id != 16 {
			continue
		}
		s0 := strOff + o
		if s0+l > len(t) {
			continue
		}
		raw := t[s0 : s0+l]
		var s string
		switch {
		case plat == 3 || plat == 0:
			u := make([]uint16, len(raw)/2)
			for j := range u {
				u[j] = binary.BigEndian.Uint16(raw[2*j:])
			}
			s = string(utf16.Decode(u))
		case plat == 1 && enc == 0 && lang == 0:
			s = string(raw) // Mac Roman; family names are ASCII in practice
		default:
			continue
		}
		s = strings.TrimSpace(s)
		switch {
		case (plat == 3 && lang == 0x0409) || (plat == 1 && lang == 0) || plat == 0:
			if _, ok := best[id]; !ok || plat == 3 {
				best[id] = s
			}
		case plat == 3 && (lang == 0x0804 || lang == 0x0404 || lang == 0x0C04 || lang == 0x1004):
			if _, ok := bestZh[id]; !ok {
				bestZh[id] = s
			}
		}
	}
	en = best[16]
	if en == "" {
		en = best[1]
	}
	zh = bestZh[16]
	if zh == "" {
		zh = bestZh[1]
	}
	if zh == en {
		zh = ""
	}
	return
}

var (
	loadMu  sync.Mutex
	loading bool
	ready   bool
)

// Load scans the system's fonts in the background, once, and calls done
// (from that goroutine) when they are ready. Ready reports the result.
func Load(done func()) {
	loadMu.Lock()
	if ready || loading {
		loadMu.Unlock()
		return
	}
	loading = true
	loadMu.Unlock()
	go func() {
		System()
		loadMu.Lock()
		ready, loading = true, false
		loadMu.Unlock()
		if done != nil {
			done()
		}
	}()
}

// Ready returns the installed families once Load finished.
func Ready() ([]Family, bool) {
	loadMu.Lock()
	defer loadMu.Unlock()
	if !ready {
		return nil, false
	}
	return cached, true
}

// SetForTest replaces the families, for tests of interfaces.
func SetForTest(fs []Family) {
	once.Do(func() {})
	loadMu.Lock()
	cached, ready = fs, true
	loadMu.Unlock()
}

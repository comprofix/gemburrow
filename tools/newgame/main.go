// Command newgame turns a fresh copy of the skeleton into a new game.
// It does the steps that must be exact:
//
//	go run ./tools/newgame -module github.com/comprofix/<game> -name <game> -title "<Game Title>" [-style pixel|smooth|both] [-keep-demo]
//
// It changes the module path in go.mod and every import, and the skeleton's
// name in code, in AGENTS.md and in the release templates; renames
// cmd/gameskeleton to cmd/<game>; sets config.Title; for the smooth art
// style sets config.Project to the Smooth preset (pixel art keeps Modern,
// to be changed by hand for another pixel preset); for both looks sets
// config.Project to Smooth and config.Second to Modern; removes the demo
// unless -keep-demo is given, and always for smooth or both, as the demo is
// pixel art;
// writes game-specific AGENTS.md guidance and a new README saying which skeleton version it
// came from; removes itself; and runs go build, go vet and go test to check
// everything still works.
//
// It refuses to run where DESIGN.md exists, so it can't damage the skeleton
// itself, whose working documents live beside it.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// SkeletonVersion is the skeleton release this copy is. Keep it in step with
// the release tag.
const SkeletonVersion = "0.5.1"

const (
	oldModule = "github.com/comprofix/gameskeleton"
	oldName   = "gameskeleton"
)

// The demo's files and folders, relative to the root.
var demoPaths = []string{
	"internal/demo",
	"internal/game/demo.go",
	"assets/sprites/demo.txt",
	"assets/gen/demo.png",
	"assets/gen/demo.json",
}

// Folders the rewrite never enters.
var skipDirs = map[string]bool{".git": true, ".claude": true, "bin": true, "dist": true, "ref": true, "preview": true}

type options struct {
	dir, module, name, title, style string
	keepDemo, check                 bool
}

// The art styles, as -style takes them, and the preset each sets: for both
// looks the first, with secondLook as the second.
var stylePresets = map[string]string{"pixel": "Modern", "smooth": "Smooth", "both": "Smooth"}

// secondLook is the second look -style both sets.
const secondLook = "Modern"

func main() {
	var o options
	flag.StringVar(&o.dir, "dir", ".", "the copy of the skeleton to change")
	flag.StringVar(&o.module, "module", "", "the new module path, e.g. github.com/comprofix/mygame")
	flag.StringVar(&o.name, "name", "", "the command's name, e.g. mygame (lower case letters, digits and -)")
	flag.StringVar(&o.title, "title", "", `the game's title, e.g. "My Game"`)
	flag.StringVar(&o.style, "style", "pixel", "the art style: pixel, smooth, or both (Smooth first, Modern second, switched in Options)")
	flag.BoolVar(&o.keepDemo, "keep-demo", false, "keep the Dot Dash demo (pixel art only)")
	flag.BoolVar(&o.check, "check", true, "run go build and go test afterwards")
	flag.Parse()
	if err := run(o, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "newgame:", err)
		os.Exit(1)
	}
}

var validName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// validModulePath checks the syntax required for a Go module import path.
// Module paths are slash-separated import paths with a dotted first element.
func validModulePath(path string) bool {
	if path == "" || strings.ContainsAny(path, " \t\r\n\\\"") {
		return false
	}
	parts := strings.Split(path, "/")
	if parts[0] == "" || parts[0][0] == '-' || !strings.Contains(parts[0], ".") {
		return false
	}
	// The first element is a domain name, so it has stricter character rules.
	for _, r := range parts[0] {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '-') {
			return false
		}
	}
	for _, part := range parts {
		if part == "" || part[0] == '.' || strings.HasSuffix(part, ".") || strings.Contains(part, "..") {
			return false
		}
		for _, r := range part {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') || strings.ContainsRune("-._~", r)) {
				return false
			}
		}
		// Module paths must be portable to Windows filesystems.
		device := part
		if dot := strings.IndexByte(device, '.'); dot >= 0 {
			device = device[:dot]
		}
		switch strings.ToUpper(device) {
		case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
			"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
			return false
		}
		if tilde := strings.LastIndexByte(part, '~'); tilde >= 0 && tilde < len(part)-1 {
			shortName := true
			for _, r := range part[tilde+1:] {
				if r < '0' || r > '9' {
					shortName = false
					break
				}
			}
			if shortName {
				return false
			}
		}
	}
	last := parts[len(parts)-1]
	if strings.HasPrefix(last, "v") {
		numeric := len(last) > 1
		for _, r := range last[1:] {
			if (r < '0' || r > '9') && r != '.' {
				numeric = false
				break
			}
		}
		if numeric && (last == "v1" || strings.Contains(last, ".") || last[1] == '0') {
			return false
		}
	}
	return true
}

func preflight(root string, o options) error {
	switch {
	case !validModulePath(o.module) || o.module == oldModule:
		return fmt.Errorf("-module %q: invalid module path; use a dotted host and valid slash-separated path elements", o.module)
	case !validName.MatchString(o.name) || o.name == oldName:
		return fmt.Errorf("-name %q: use lower case letters, digits and -, starting with a letter", o.name)
	case strings.TrimSpace(o.title) == "":
		return errors.New("-title: give the game's title")
	case stylePresets[o.style] == "":
		return fmt.Errorf("-style %q: use pixel, smooth or both", o.style)
	case o.style != "pixel" && o.keepDemo:
		return errors.New("-keep-demo: the demo is pixel art, so only a pixel game can keep it")
	}

	if _, err := os.Stat(filepath.Join(root, "DESIGN.md")); err == nil {
		return errors.New("DESIGN.md exists here: newgame runs on a fresh copy before DESIGN.md is written (move it aside to go on)")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := rejectSymlinks(root); err != nil {
		return err
	}

	for _, p := range []string{
		"go.mod", "AGENTS.md", "README.md", "CLAUDE.md",
		"cmd/gameskeleton/main.go", "internal/config/project.go",
	} {
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(p)))
		if err != nil {
			return fmt.Errorf("preflight: required file %s: %w", p, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("preflight: %s is not a regular file", p)
		}
	}
	for _, p := range []string{"tools/newgame", "cmd", "internal/config"} {
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(p)))
		if err != nil {
			return fmt.Errorf("preflight: required directory %s: %w", p, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("preflight: %s is not a directory", p)
		}
	}

	gomod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return err
	}
	if !strings.HasPrefix(string(gomod), "module "+oldModule+"\n") {
		return errors.New("go.mod is not the skeleton's: has this copy already been set up?")
	}

	agents, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		return err
	}
	a := string(agents)
	start := strings.Index(a, "## 3. Phases and decisions")
	end := strings.Index(a, "## 11. Rules for the skeleton itself")
	if start < 0 || end <= start {
		return errors.New("AGENTS.md: cannot find ordered shared instruction sections 3-10")
	}

	project, err := os.ReadFile(filepath.Join(root, "internal/config/project.go"))
	if err != nil {
		return err
	}
	if !titleLine.Match(project) || !presetLine.Match(project) {
		return errors.New("internal/config/project.go: missing title or project preset declaration")
	}
	if o.style == "both" && !secondLine.Match(project) {
		return errors.New("internal/config/project.go: missing second preset declaration")
	}
	if _, err := os.Lstat(filepath.Join(root, "cmd", o.name)); err == nil {
		return fmt.Errorf("cmd/%s already exists", o.name)
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func rejectSymlinks(root string) error {
	info, err := os.Lstat(root)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing symlinked project root %s", root)
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing symlink in project: %s", rel)
		}
		return nil
	})
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path != src && d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := dst
		if rel != "." {
			target = filepath.Join(dst, rel)
		}
		if d.Type()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			if err := os.MkdirAll(target, info.Mode().Perm()); err != nil {
				return err
			}
			return os.Chmod(target, info.Mode().Perm())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(target, data, info.Mode().Perm()); err != nil {
			return err
		}
		return os.Chmod(target, info.Mode().Perm())
	})
}

func restoreTree(backup, root string) error {
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() == ".git" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(root, entry.Name())); err != nil {
			return err
		}
	}
	return copyTree(backup, root)
}

func run(o options, log io.Writer) error {
	root := o.dir
	if err := preflight(root, o); err != nil {
		return err
	}
	backup, err := os.MkdirTemp("", "newgame-backup-")
	if err != nil {
		return fmt.Errorf("create rollback snapshot: %w", err)
	}
	defer os.RemoveAll(backup)
	if err := copyTree(root, backup); err != nil {
		return fmt.Errorf("snapshot project: %w", err)
	}
	if err := runInPlace(o, log); err != nil {
		if restoreErr := restoreTree(backup, root); restoreErr != nil {
			return errors.Join(err, fmt.Errorf("restore original project: %w", restoreErr))
		}
		return err
	}
	return nil
}

func runInPlace(o options, log io.Writer) error {
	switch {
	case !validModulePath(o.module) || o.module == oldModule:
		return fmt.Errorf("-module %q: give the new game's module path", o.module)
	case !validName.MatchString(o.name) || o.name == oldName:
		return fmt.Errorf("-name %q: use lower case letters, digits and -, starting with a letter", o.name)
	case strings.TrimSpace(o.title) == "":
		return errors.New("-title: give the game's title")
	case stylePresets[o.style] == "":
		return fmt.Errorf("-style %q: use pixel, smooth or both", o.style)
	case o.style != "pixel" && o.keepDemo:
		return errors.New("-keep-demo: the demo is pixel art, so only a pixel game can keep it")
	}
	root := o.dir
	if _, err := os.Stat(filepath.Join(root, "DESIGN.md")); err == nil {
		return errors.New("DESIGN.md exists here: newgame runs on a fresh copy before DESIGN.md is written (move it aside to go on)")
	}
	gomod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return err
	}
	if !strings.HasPrefix(string(gomod), "module "+oldModule+"\n") {
		return errors.New("go.mod is not the skeleton's: has this copy already been set up?")
	}

	n, err := rewrite(root, o)
	if err != nil {
		return err
	}
	fmt.Fprintf(log, "renamed the module and %s in %d files\n", oldName, n)

	if err := writeGameInstructions(root, o); err != nil {
		return err
	}
	fmt.Fprintln(log, "wrote game-specific AGENTS.md")

	if err := os.Rename(filepath.Join(root, "cmd", oldName), filepath.Join(root, "cmd", o.name)); err != nil {
		return err
	}
	fmt.Fprintf(log, "moved cmd/%s to cmd/%s\n", oldName, o.name)

	project := filepath.Join(root, "internal", "config", "project.go")
	if err := setTitle(project, o.title); err != nil {
		return err
	}
	fmt.Fprintf(log, "set the title to %q\n", o.title)
	if err := setPreset(project, stylePresets[o.style]); err != nil {
		return err
	}
	if o.style == "both" {
		if err := setSecond(project, secondLook); err != nil {
			return err
		}
		fmt.Fprintf(log, "set the looks to %s and %s\n", stylePresets[o.style], secondLook)
	} else {
		fmt.Fprintf(log, "set the preset to %s for %s art\n", stylePresets[o.style], o.style)
	}

	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(readme(o)), 0o644); err != nil {
		return err
	}
	fmt.Fprintln(log, "wrote README.md")

	if o.check {
		for _, args := range [][]string{{"build", "./..."}, {"vet", "./..."}} {
			fmt.Fprintln(log, "go", strings.Join(args, " "))
			cmd := exec.Command("go", args...)
			cmd.Dir, cmd.Stdout, cmd.Stderr = root, log, log
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("go %s: %w", args[0], err)
			}
		}
	}

	// Remove the generator before go test: its own integration test runs the
	// generator and would recursively invoke go test otherwise. The enclosing
	// snapshot restores these paths if the final test fails.
	remove := []string{"tools/newgame"}
	if !o.keepDemo {
		remove = append(remove, demoPaths...)
	}
	for _, p := range remove {
		if err := os.RemoveAll(filepath.Join(root, filepath.FromSlash(p))); err != nil {
			return err
		}
		fmt.Fprintln(log, "removed", p)
	}

	if o.check {
		args := []string{"test", "./..."}
		fmt.Fprintln(log, "go", strings.Join(args, " "))
		cmd := exec.Command("go", args...)
		cmd.Dir, cmd.Stdout, cmd.Stderr = root, log, log
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("go %s: %w", args[0], err)
		}
	}
	fmt.Fprintln(log, "done")
	return nil
}

// rewrite changes the module path, then the skeleton's name, in every Go
// file, go.mod and Markdown file, and returns how many it changed.
func rewrite(root string, o options) (int, error) {
	changed := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		switch filepath.Ext(path) {
		case ".go", ".mod", ".md", ".yml", ".toml":
		default:
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		original := string(data)
		// Protect the requested module path while replacing the skeleton name:
		// a valid destination may itself contain "gameskeleton".
		placeholder := "__NEWGAME_MODULE_PATH_PLACEHOLDER__"
		for strings.Contains(original, placeholder) {
			placeholder += "_"
		}
		s := strings.ReplaceAll(original, oldModule, placeholder)
		s = strings.ReplaceAll(s, oldName, o.name)
		s = strings.ReplaceAll(s, placeholder, o.module)
		if s == string(data) {
			return nil
		}
		changed++
		return os.WriteFile(path, []byte(s), 0o644)
	})
	return changed, err
}

var titleLine = regexp.MustCompile(`(?m)^const Title = ".*"$`)

// setTitle sets config.Title in project.go.
func setTitle(path, title string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !titleLine.Match(data) {
		return fmt.Errorf("%s: no `const Title = \"...\"` line", path)
	}
	out := titleLine.ReplaceAllLiteral(data, []byte("const Title = "+strconv.Quote(title)))
	return os.WriteFile(path, out, 0o644)
}

var presetLine = regexp.MustCompile(`(?m)^var Project = \w+$`)

// setPreset sets config.Project in project.go to the named preset.
func setPreset(path, preset string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !presetLine.Match(data) {
		return fmt.Errorf("%s: no `var Project = ...` line", path)
	}
	out := presetLine.ReplaceAllLiteral(data, []byte("var Project = "+preset))
	return os.WriteFile(path, out, 0o644)
}

var secondLine = regexp.MustCompile(`(?m)^var Second \*Preset$`)

// setSecond sets config.Second in project.go to the named preset.
func setSecond(path, preset string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !secondLine.Match(data) {
		return fmt.Errorf("%s: no `var Second *Preset` line", path)
	}
	out := secondLine.ReplaceAllLiteral(data, []byte("var Second = &"+preset))
	return os.WriteFile(path, out, 0o644)
}

// writeGameInstructions turns the skeleton's setup and maintenance guidance
// into the shared instructions for a game created from it. The common
// development rules (sections 3-10) stay sourced from the skeleton AGENTS.md.
func writeGameInstructions(root string, o options) error {
	path := filepath.Join(root, "AGENTS.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	source := string(data)
	start := strings.Index(source, "## 3. Phases and decisions")
	end := strings.Index(source, "## 11. Rules for the skeleton itself")
	if start < 0 || end < 0 || end <= start {
		return fmt.Errorf("%s: cannot find shared instruction sections 3-10", path)
	}
	project := fmt.Sprintf(`## 1. Project

**%s**: <one-line description of the game and its fixed decisions>.

Replace this section with the game's description and fixed decisions once
DESIGN.md and PLAN.md have been prepared.

## 2. Working documents

DESIGN.md and PLAN.md are personal working documents and are not tracked in
git. If either is missing at the start of a session, ask the user for their
copies before doing anything else. Do not create them unless asked. This
AGENTS.md is tracked and contains shared project guidance; CLAUDE.md is a
compatibility hand-off to it. Keep personal instructions in the local,
untracked configuration supported by the chosen coding agent.

`, o.title)
	out := "# AGENTS.md\n\nShared, agent-neutral instructions for this game. This file is tracked in git. Agent-specific entry points should hand off to this file; personal instructions belong in the chosen agent's local, untracked configuration.\n\n" + project + source[start:end]
	return os.WriteFile(path, []byte(out), 0o644)
}

// readme is the new game's starting README.
func readme(o options) string {
	return fmt.Sprintf(`# %s

Created from [gameskeleton](https://github.com/comprofix/gameskeleton) v%s.

## Requirements

- Go 1.27 or later
- **Linux only:** cgo plus the X11/GL/ALSA development packages:
  `+"`libgl1-mesa-dev libxcursor-dev libxi-dev libxinerama-dev libxrandr-dev libxxf86vm-dev libasound2-dev pkg-config`"+`

## Build and run

`+"```sh"+`
go run ./cmd/%[3]s                 # run the game
go run -tags debug ./cmd/%[3]s     # with the debug overlay
go generate ./assets               # rebuild sprite sheets
go vet ./...
go test ./...
go build -o bin/ ./cmd/%[3]s
`+"```"+`

## Personal documents

`+"`DESIGN.md` and `PLAN.md`"+` are personal working documents. `+"`.gitignore`"+` keeps
them out of git, so back them up somewhere else. `+"`AGENTS.md`"+` is tracked and contains the shared project guidance. `+"`CLAUDE.md`"+` is a compatibility hand-off for Claude Code. Keep personal instructions in the local, untracked configuration supported by your coding agent.

## License

MIT. See [LICENSE](LICENSE).
`, o.title, SkeletonVersion, o.name)
}

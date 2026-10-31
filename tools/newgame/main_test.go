package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// skeleton is the skeleton's root, two folders up.
const skeleton = "../.."

// copySkeleton copies what a fresh clone of the skeleton has into dst:
// everything but git's folder, local documents and settings, and build
// output.
func copySkeleton(t *testing.T, dst string) {
	t.Helper()
	skipFiles := map[string]bool{"DESIGN.md": true, "PLAN.md": true, "settings.json": true, "scores.json": true}
	err := filepath.WalkDir(skeleton, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(skeleton, path)
		if d.IsDir() {
			if rel != "." && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		if skipFiles[rel] || filepath.Ext(rel) == ".exe" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func exists(root, rel string) bool {
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil
}

// leftovers lists every line still naming the skeleton, apart from the
// README's note of where the game came from.
func leftovers(t *testing.T, root string) []string {
	t.Helper()
	var found []string
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		for i, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, oldName) && !(rel == "README.md" && strings.HasPrefix(line, "Created from")) {
				found = append(found, fmt.Sprintf("%s:%d: %s", filepath.ToSlash(rel), i+1, line))
			}
		}
		return nil
	})
	return found
}

func TestRewritePreservesModuleContainingSkeletonName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+oldModule+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "import.go"), []byte("package example\nimport _ \""+oldModule+"/internal/config\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	module := "example.com/me/mygameskeleton-game"
	if _, err := rewrite(dir, options{module: module, name: "arcade"}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"go.mod", "import.go"} {
		got, err := os.ReadFile(filepath.Join(dir, path))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(got), module) {
			t.Errorf("%s does not contain requested module path %q: %s", path, module, got)
		}
		if strings.Contains(string(got), "myarcade-game") {
			t.Errorf("%s had the requested module path rewritten: %s", path, got)
		}
	}
}

func TestValidModulePath(t *testing.T) {
	for _, tc := range []struct {
		path string
		want bool
	}{
		{"example.com", true},
		{"example.com/me/game", true},
		{"github.com/comprofix/mygameskeleton-game", true},
		{"example.com/game.v2", true},
		{"example.com/.hidden", false},
		{"example.com/game.", false},
		{"example.com/foo..bar", false},
		{"-example.com/game", false},
		{"Example.com/game", false},
		{"example.com/con", false},
		{"example.com/pkg~1", false},
		{"example.com/game/v1", false},
		{"example.com/game/v02", false},
		{"example.com/game/v2.0", false},
		{"", false},
		{"/example.com/game", false},
		{"example.com/game/", false},
		{"example.com//game", false},
		{"example.com/./game", false},
		{"example.com/../game", false},
		{"example.com/..", false},
		{"example.com/game\nother", false},
		{"example.com/game\rother", false},
		{"example.com/game\tother", false},
		{"example.com/game\\other", false},
		{"example.com/game name", false},
		{"example.com/game:tag", false},
		{"localhost/game", false},
	} {
		t.Run(fmt.Sprintf("%q", tc.path), func(t *testing.T) {
			if got := validModulePath(tc.path); got != tc.want {
				t.Errorf("validModulePath(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestNewGame(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and tests whole games")
	}
	for _, c := range []struct {
		name, style, preset, second string
		keep                        bool
	}{
		{"pixel without the demo", "pixel", "Modern", "*Preset", false},
		{"pixel with the demo", "pixel", "Modern", "*Preset", true},
		{"smooth", "smooth", "Smooth", "*Preset", false},
		{"both looks", "both", "Smooth", "= &Modern", false},
	} {
		keep := c.keep
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			copySkeleton(t, dir)
			var log bytes.Buffer
			o := options{dir: dir, module: "example.com/me/rocket-rumble", name: "rocket-rumble", title: `Rocket "RR" Rumble`, style: c.style, keepDemo: keep, check: true}
			if err := run(o, &log); err != nil {
				t.Fatalf("%v\n%s", err, log.String())
			}
			t.Log(log.String())

			for _, rel := range []string{"cmd/rocket-rumble/main.go", "internal/scene/title.go", "assets/gen/README.md"} {
				if !exists(dir, rel) {
					t.Errorf("%s is missing", rel)
				}
			}
			for _, rel := range []string{"cmd/gameskeleton", "tools/newgame"} {
				if exists(dir, rel) {
					t.Errorf("%s is still there", rel)
				}
			}
			for _, rel := range demoPaths {
				if exists(dir, rel) != keep {
					t.Errorf("%s: there %v, want %v", rel, !keep, keep)
				}
			}
			if gomod, _ := os.ReadFile(filepath.Join(dir, "go.mod")); !strings.HasPrefix(string(gomod), "module example.com/me/rocket-rumble\n") {
				t.Errorf("go.mod starts %q", strings.SplitN(string(gomod), "\n", 2)[0])
			}
			project, _ := os.ReadFile(filepath.Join(dir, "internal", "config", "project.go"))
			if !strings.Contains(string(project), `const Title = "Rocket \"RR\" Rumble"`) {
				t.Error("the title was not set in project.go")
			}
			if !strings.Contains(string(project), "\nvar Project = "+c.preset+"\n") {
				t.Errorf("the preset was not set to %s in project.go", c.preset)
			}
			if !strings.Contains(string(project), "\nvar Second "+c.second+"\n") {
				t.Errorf("the second look is not `var Second %s` in project.go", c.second)
			}
			readme, _ := os.ReadFile(filepath.Join(dir, "README.md"))
			if !strings.Contains(string(readme), "Created from [gameskeleton](https://github.com/comprofix/gameskeleton) v"+SkeletonVersion) {
				t.Errorf("README does not say where it came from:\n%s", readme)
			}

			agents, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
			if err != nil {
				t.Fatalf("generated AGENTS.md: %v", err)
			}
			for _, want := range []string{"# AGENTS.md", "Rocket", "## 2. Working documents", "DESIGN.md", "## 3. Phases and decisions", "recorded in DESIGN.md"} {
				if !strings.Contains(string(agents), want) {
					t.Errorf("generated AGENTS.md does not contain %q", want)
				}
			}
			for _, unwanted := range []string{"New game setup", "Rules for the skeleton itself", "setup (§2)", "Dot Dash", oldName} {
				if strings.Contains(string(agents), unwanted) {
					t.Errorf("generated AGENTS.md still contains skeleton-only text %q", unwanted)
				}
			}
			claude, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
			if err != nil || string(claude) != "@AGENTS.md\n" {
				t.Errorf("generated CLAUDE.md = %q, err %v; want @AGENTS.md hand-off", claude, err)
			}
			ignore, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
			if err != nil {
				t.Fatalf("generated .gitignore: %v", err)
			}
			if strings.Contains(string(ignore), "\nAGENTS.md\n") {
				t.Error("generated .gitignore still ignores AGENTS.md")
			}
			if !keep {
				// A game with no sprites yet can still regenerate its assets.
				cmd := exec.Command("go", "generate", "./assets")
				cmd.Dir = dir
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Errorf("go generate with no sprites: %v\n%s", err, out)
				}
			}
			if left := leftovers(t, dir); len(left) > 0 {
				t.Errorf("still naming the skeleton:\n%s", strings.Join(left, "\n"))
			}
		})
	}
}

func TestRejectsSymlinkWithoutTouchingTarget(t *testing.T) {
	dir := t.TempDir()
	copySkeleton(t, dir)
	external := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(external, []byte("external data"), 0o644); err != nil {
		t.Fatal(err)
	}
	readme := filepath.Join(dir, "README.md")
	if err := os.Remove(readme); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, readme); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	o := options{dir: dir, module: "example.com/me/game", name: "game", title: "Game", style: "pixel", check: false}
	if err := run(o, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("run error = %v, want symlink refusal", err)
	}
	got, err := os.ReadFile(external)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "external data" {
		t.Fatalf("external target changed: %q", got)
	}
	if !exists(dir, "tools/newgame/main.go") {
		t.Fatal("generator was removed after preflight refusal")
	}
}

func TestRestoresProjectWhenValidationFails(t *testing.T) {
	dir := t.TempDir()
	copySkeleton(t, dir)
	fakeBin := t.TempDir()
	countFile := filepath.Join(t.TempDir(), "go-count")
	fakeGo := filepath.Join(fakeBin, "go")
	script := "#!/bin/sh\ncount=0\nif [ -f " + countFile + " ]; then count=$(cat " + countFile + "); fi\ncount=$((count + 1))\necho \"$count\" > " + countFile + "\nif [ \"$count\" -eq 3 ]; then exit 17; fi\nexit 0\n"
	if err := os.WriteFile(fakeGo, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	o := options{dir: dir, module: "example.com/me/game", name: "game", title: "Game", style: "pixel", check: true}
	if err := run(o, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "go test") {
		t.Fatalf("run error = %v, want go test failure after cleanup", err)
	}
	if !exists(dir, "tools/newgame/main.go") || !exists(dir, "cmd/gameskeleton/main.go") || !exists(dir, "internal/demo") {
		t.Fatal("failed transformation did not restore generator, demo and original command")
	}
	if exists(dir, "cmd/game") {
		t.Fatal("failed transformation left renamed command")
	}
	gomod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(gomod), "module "+oldModule+"\n") {
		t.Fatal("failed transformation did not restore go.mod")
	}
}

func TestRejectsSymlinkedDirectory(t *testing.T) {
	dir := t.TempDir()
	copySkeleton(t, dir)
	external := t.TempDir()
	marker := filepath.Join(external, "project.go")
	if err := os.WriteFile(marker, []byte("external data"), 0o644); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(dir, "internal", "config")
	if err := os.RemoveAll(config); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, config); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	o := options{dir: dir, module: "example.com/me/game", name: "game", title: "Game", style: "pixel", check: false}
	if err := run(o, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("run error = %v, want symlink refusal", err)
	}
	got, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "external data" {
		t.Fatalf("external directory target changed: %q", got)
	}
}
func TestRefuses(t *testing.T) {
	good := options{module: "example.com/me/game", name: "game", title: "Game", style: "pixel"}
	for name, o := range map[string]options{
		"no module":     {name: "game", title: "Game"},
		"same module":   {module: oldModule, name: "game", title: "Game"},
		"bad name":      {module: good.module, name: "My Game", title: "Game"},
		"same name":     {module: good.module, name: oldName, title: "Game"},
		"no title":      {module: good.module, name: "game", title: "  ", style: "pixel"},
		"bad style":     {module: good.module, name: "game", title: "Game", style: "vector"},
		"smooth demo":   {module: good.module, name: "game", title: "Game", style: "smooth", keepDemo: true},
		"both demo":     {module: good.module, name: "game", title: "Game", style: "both", keepDemo: true},
		"no skeleton":   good,
		"design exists": good,
	} {
		dir := t.TempDir()
		o.dir = dir
		switch name {
		case "no skeleton":
			os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/other\n"), 0o644)
		case "design exists":
			os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module "+oldModule+"\n"), 0o644)
			os.WriteFile(filepath.Join(dir, "DESIGN.md"), []byte("# mine\n"), 0o644)
		}
		if err := run(o, &bytes.Buffer{}); err == nil {
			t.Errorf("%s: no error", name)
		}
		if name == "design exists" && !exists(dir, "DESIGN.md") {
			t.Error("removed DESIGN.md")
		}
	}
}

//go:build mage
// +build mage

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var (
	binDir  = "bin"
	tmpDir  = "tmp"
	appName = "pehlione-web"
)

var Default = Dev

// Dev: Gerekli ön adımları çalıştırır (templ generate, tidy) sonra air/go run
func Dev() error {
	mg.Deps(PreDev)

	// air varsa onu kullan
	if _, err := exec.LookPath("air"); err == nil {
		fmt.Println("Starting hot-reload with air ...")
		return sh.RunV("air")
	}

	fmt.Println("air not found. Falling back to `go run ./cmd/web`.")
	fmt.Println("Install with: mage Tools")
	return Run()
}

// PreDev: Dev öncesi gerekli işler (gerektikçe buraya ekleyeceğiz)
func PreDev() error {
	mg.Deps(Tidy, Gen)
	return nil
}

// Gen: Codegen (şu an templ). İleride sqlc/mockery vs eklenebilir.
func Gen() error {
	// templ binary var mı?
	if _, err := exec.LookPath("templ"); err != nil {
		return fmt.Errorf("templ not found. Install with: mage Tools")
	}
	fmt.Println("Generating templ components...")
	return sh.RunV("templ", "generate")
}

func Run() error {
	mg.Deps(Gen) // go run öncesi de garanti olsun
	fmt.Println("Running (go run) on :8080 ...")
	return sh.RunV("go", "run", "./cmd/web")
}

func Build() error {
	mg.Deps(Tidy, Gen)

	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}

	out := filepath.Join(binDir, appName+exeSuffix())
	fmt.Println("Building:", out)

	env := map[string]string{"CGO_ENABLED": "0"}
	return sh.RunWithV(env, "go", "build", "-trimpath", "-o", out, "./cmd/web")
}

func Test() error {
	fmt.Println("Testing...")
	return sh.RunV("go", "test", "./...", "-count=1")
}

func TestRace() error {
	fmt.Println("Testing with -race...")
	if runtime.GOOS == "windows" {
		fmt.Println("Note: -race on Windows may be unsupported/unstable depending on your Go toolchain.")
	}
	return sh.RunV("go", "test", "./...", "-race", "-count=1")
}

func Fmt() error {
	fmt.Println("Formatting...")
	return sh.RunV("gofmt", "-w", "./cmd", "./internal", "./magefile.go")
}

func Lint() error {
	fmt.Println("Linting (golangci-lint)...")
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		return fmt.Errorf("golangci-lint not found. Install with: mage Tools")
	}
	return sh.RunV("golangci-lint", "run", "--timeout=3m", "./...")
}

func Check() error {
	mg.Deps(Fmt, Lint, Test)
	fmt.Println("Check OK.")
	return nil
}

func Tidy() error {
	fmt.Println("Tidying go.mod/go.sum...")
	return sh.RunV("go", "mod", "tidy")
}

func Clean() error {
	fmt.Println("Cleaning...")
	_ = os.RemoveAll(binDir)
	_ = os.RemoveAll(tmpDir)
	return nil
}

// Tools: air + templ + golangci-lint (v2 önerilir) kur
func Tools() error {
	fmt.Println("Installing tools (air, templ, golangci-lint)...")

	if err := sh.RunV("go", "install", "github.com/air-verse/air@latest"); err != nil {
		return err
	}
	if err := sh.RunV("go", "install", "github.com/a-h/templ/cmd/templ@latest"); err != nil {
		return err
	}

	// Not: golangci-lint v2 için v2 path kullanın (config v2 istiyorsanız)
	// Eğer sizde v1 kullanacaksanız bu satırı eski haline bırakın.
	if err := sh.RunV("go", "install", "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest"); err != nil {
		return err
	}

	// PATH kontrolü
	if _, err := exec.LookPath("air"); err != nil && !errors.Is(err, exec.ErrNotFound) {
		return err
	}
	if _, err := exec.LookPath("templ"); err != nil && !errors.Is(err, exec.ErrNotFound) {
		return err
	}
	if _, err := exec.LookPath("golangci-lint"); err != nil && !errors.Is(err, exec.ErrNotFound) {
		return err
	}

	fmt.Println("Tools installed. Ensure GOBIN/GOPATH/bin is in PATH.")
	return nil
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}
func MigrateUp() error {
	return sh.RunV("goose", "-dir", "./migrations", "mysql", os.Getenv("DB_DSN"), "up")
}

func MigrateDown() error {
	return sh.RunV("goose", "-dir", "./migrations", "mysql", os.Getenv("DB_DSN"), "down")
}

// CSS: Build Tailwind CSS via PostCSS (Tailwind v4)
func CSS() error {
	fmt.Println("Building CSS (tailwind via postcss)...")
	npm := "npm"
	if runtime.GOOS == "windows" {
		npm = "npm.cmd"
	}
	if err := sh.RunV(npm, "run", "build:css"); err != nil {
		return err
	}
	return nil
}

// CSSWatch: Watch and rebuild Tailwind CSS on changes
func CSSWatch() error {
	fmt.Println("Watching CSS (tailwind via postcss --watch)...")
	npm := "npm"
	if runtime.GOOS == "windows" {
		npm = "npm.cmd"
	}
	return sh.RunV(npm, "run", "dev:css")
}

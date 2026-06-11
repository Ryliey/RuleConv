// Package compiler shells out to the official mihomo and sing-box binaries.
// Shelling out to pinned binaries (rather than importing the cores' internal
// packages) guarantees byte-correct output and avoids their unstable APIs.
package compiler

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Core binary paths. Default to PATH lookup; override via MIHOMO_BIN /
// SINGBOX_BIN or SetBinaries.
var (
	mihomoBin  = envOr("MIHOMO_BIN", "mihomo")
	singboxBin = envOr("SINGBOX_BIN", "sing-box")
)

// SetBinaries overrides the core binary paths. Empty values are ignored.
func SetBinaries(mihomo, singbox string) {
	if mihomo != "" {
		mihomoBin = mihomo
	}
	if singbox != "" {
		singboxBin = singbox
	}
}

// MihomoConvert runs `mihomo convert-ruleset <behavior> <format> <src> <out>`.
// behavior must be "domain" or "ipcidr"; format must be "yaml" or "text".
func MihomoConvert(behavior, format, src, out string) error {
	if behavior != "domain" && behavior != "ipcidr" {
		return fmt.Errorf("mihomo: unsupported behavior %q (mrs supports only domain/ipcidr)", behavior)
	}
	return run(mihomoBin, "convert-ruleset", behavior, format, src, out)
}

// SingBoxCompile runs `sing-box rule-set compile --output <out> <src>`.
func SingBoxCompile(src, out string) error {
	return run(singboxBin, "rule-set", "compile", "--output", out, src)
}

// CheckAvailable returns an error naming any core binary not found.
func CheckAvailable(needMihomo, needSingbox bool) error {
	var missing []string
	if needMihomo {
		if _, err := exec.LookPath(mihomoBin); err != nil {
			missing = append(missing, fmt.Sprintf("mihomo (%q)", mihomoBin))
		}
	}
	if needSingbox {
		if _, err := exec.LookPath(singboxBin); err != nil {
			missing = append(missing, fmt.Sprintf("sing-box (%q)", singboxBin))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing core binaries: %s — install them or set MIHOMO_BIN/SINGBOX_BIN, or pass --skip-binary", strings.Join(missing, ", "))
	}
	return nil
}

func run(bin string, args ...string) error {
	cmd := exec.Command(bin, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		var execErr *exec.Error
		if errors.As(err, &execErr) {
			return fmt.Errorf("%s: %w (is it installed and on PATH?)", bin, err)
		}
		return fmt.Errorf("%s %s: %w: %s", bin, strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

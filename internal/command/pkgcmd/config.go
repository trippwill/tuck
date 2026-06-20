package pkgcmd

import (
	"fmt"
	"strings"

	"github.com/trippwill/tuck/internal/apperr"
	"github.com/trippwill/tuck/internal/command"
	"github.com/trippwill/tuck/internal/domain"
	"github.com/trippwill/tuck/internal/output"
	"github.com/trippwill/tuck/internal/packages"
	"github.com/trippwill/tuck/internal/pkgref"
)

const KindConfig output.Kind = "config"

type ConfigShowRequest struct {
	Ref      string
	Path     string
	SourceID string
	Context  string
}

type ConfigSetRequest struct {
	Ref       string
	Path      string
	Deploy    string
	Mode      string
	SetDeploy bool
	SetMode   bool
	SourceID  string
	Context   string
}

type ConfigUnsetRequest struct {
	Ref         string
	Path        string
	UnsetDeploy bool
	UnsetMode   bool
	SourceID    string
	Context     string
}

type ConfigPayload struct {
	Source       string       `json:"source"`
	Package      string       `json:"package"`
	ManifestPath string       `json:"manifestPath,omitempty"`
	Files        []ConfigFile `json:"files"`
}

type ConfigFile struct {
	Path       string `json:"path"`
	Deploy     string `json:"deploy"`
	Mode       string `json:"mode,omitempty"`
	Configured bool   `json:"configured"`
}

func ConfigShow(req ConfigShowRequest) output.Outcome {
	payload, err := configPayload(req.SourceID, req.Context, req.Ref, req.Path)
	if err != nil {
		return command.ErrorOutcome(err)
	}
	return output.OK(configResult(payload))
}

func ConfigSet(req ConfigSetRequest) output.Outcome {
	if !req.SetDeploy && !req.SetMode {
		return output.OK(output.InvalidArgs("package config set requires --deploy or --mode", "pass --deploy symlink|copy, --mode <octal>, or both"))
	}
	resolved, config, rel, err := loadConfigTarget(req.SourceID, req.Context, req.Ref, req.Path)
	if err != nil {
		return command.ErrorOutcome(err)
	}
	file, ok := packages.ConfiguredFile(config, rel)
	if !ok {
		file = packages.FileConfig{Path: rel, Deploy: packages.DeploySymlink}
	}
	if req.SetDeploy {
		deploy := packages.Deploy(req.Deploy)
		if deploy != packages.DeploySymlink && deploy != packages.DeployCopy {
			return output.OK(output.InvalidArgs("deploy must be symlink or copy", "pass --deploy symlink or --deploy copy"))
		}
		file.Deploy = deploy
	}
	if req.SetMode {
		baseMode, err := configModeBase(resolved, file, rel)
		if err != nil {
			return command.ErrorOutcome(err)
		}
		mode, err := packages.NormalizeModeFlag(req.Mode, baseMode)
		if err != nil {
			return output.OK(output.InvalidArgs("mode must be octal or chmod-style rwx expression", "pass a mode like 0600 or u=rw,go="))
		}
		file.Mode = mode
	}
	config = packages.SetFileConfig(config, file)
	if err := packages.SaveConfig(resolved.Identity.Root, config); err != nil {
		return command.ErrorOutcome(err)
	}
	payload, err := configPayload(req.SourceID, req.Context, req.Ref, rel)
	if err != nil {
		return command.ErrorOutcome(err)
	}
	return output.OK(configResult(payload))
}

func ConfigUnset(req ConfigUnsetRequest) output.Outcome {
	resolved, config, rel, err := loadConfigTarget(req.SourceID, req.Context, req.Ref, req.Path)
	if err != nil {
		return command.ErrorOutcome(err)
	}
	file, ok := packages.ConfiguredFile(config, rel)
	if !ok {
		payload, err := configPayload(req.SourceID, req.Context, req.Ref, rel)
		if err != nil {
			return command.ErrorOutcome(err)
		}
		return output.OK(configResult(payload))
	}
	if !req.UnsetDeploy && !req.UnsetMode {
		config = packages.RemoveFileConfig(config, rel)
	} else {
		if req.UnsetDeploy {
			file.Deploy = packages.DeploySymlink
		}
		if req.UnsetMode {
			file.Mode = ""
		}
		if file.Deploy == packages.DeploySymlink && file.Mode == "" {
			config = packages.RemoveFileConfig(config, rel)
		} else {
			config = packages.SetFileConfig(config, file)
		}
	}
	if err := packages.SaveConfig(resolved.Identity.Root, config); err != nil {
		return command.ErrorOutcome(err)
	}
	payload, err := configPayload(req.SourceID, req.Context, req.Ref, rel)
	if err != nil {
		return command.ErrorOutcome(err)
	}
	return output.OK(configResult(payload))
}

func configPayload(sourceID, contextName, rawRef, rawPath string) (ConfigPayload, error) {
	resolved, config, rel, err := loadConfigTarget(sourceID, contextName, rawRef, rawPath)
	if err != nil {
		return ConfigPayload{}, err
	}
	payload := ConfigPayload{
		Source:       resolved.Identity.Source,
		Package:      resolved.Identity.String(),
		ManifestPath: config.ManifestPath,
	}
	if rel != "" {
		payload.Files = append(payload.Files, effectiveConfig(config, rel))
		return payload, nil
	}
	for _, file := range config.Files {
		payload.Files = append(payload.Files, ConfigFile{Path: file.Path, Deploy: string(file.Deploy), Mode: file.Mode, Configured: true})
	}
	return payload, nil
}

func loadConfigTarget(sourceID, contextName, rawRef, rawPath string) (packages.Resolved, packages.PackageConfig, string, error) {
	selection, err := domain.SelectActive(domain.SelectionOptions{
		SourceID:    sourceID,
		Context:     contextName,
		RequireHome: false,
	})
	if err != nil {
		return packages.Resolved{}, packages.PackageConfig{}, "", err
	}
	ref, err := pkgref.Parse(rawRef)
	if err != nil {
		return packages.Resolved{}, packages.PackageConfig{}, "", err
	}
	resolved, err := packages.ResolveOne(selection.Source, selection.Scope.Context, ref.Name)
	if err != nil {
		return packages.Resolved{}, packages.PackageConfig{}, "", err
	}
	config, err := packages.LoadConfig(resolved.Identity.Root, resolved.Entries)
	if err != nil {
		return packages.Resolved{}, packages.PackageConfig{}, "", err
	}
	if rawPath == "" {
		return resolved, config, "", nil
	}
	rel, err := cleanRequestedLeaf(resolved, rawPath)
	if err != nil {
		return packages.Resolved{}, packages.PackageConfig{}, "", err
	}
	return resolved, config, rel, nil
}

func cleanRequestedLeaf(resolved packages.Resolved, rawPath string) (string, error) {
	for _, entry := range packages.Leaves(resolved.Entries) {
		if entry.Rel == rawPath {
			return rawPath, nil
		}
	}
	return "", apperr.AppErrMsgf(packages.ErrConfigInvalid, "package path %q is not a package leaf", rawPath)
}

func configModeBase(resolved packages.Resolved, file packages.FileConfig, rel string) (string, error) {
	if file.Mode != "" {
		return file.Mode, nil
	}
	for _, entry := range packages.Leaves(resolved.Entries) {
		if entry.Rel == rel {
			return packages.ModeFromFile(entry.Path)
		}
	}
	return "", apperr.AppErrMsgf(packages.ErrConfigInvalid, "package path %q is not a package leaf", rel)
}

func effectiveConfig(config packages.PackageConfig, rel string) ConfigFile {
	if file, ok := packages.ConfiguredFile(config, rel); ok {
		return ConfigFile{Path: file.Path, Deploy: string(file.Deploy), Mode: file.Mode, Configured: true}
	}
	return ConfigFile{Path: rel, Deploy: string(packages.DeploySymlink), Configured: false}
}

func configResult(payload ConfigPayload) output.Result {
	return output.Result{
		Kind:          KindConfig,
		Data:          payload,
		ExitCode:      output.ExitOK,
		ConsoleString: renderConfig,
	}
}

func renderConfig(console output.Console, data any) (string, error) {
	p, ok := data.(ConfigPayload)
	if !ok {
		return "", fmt.Errorf("package config console renderer received %T", data)
	}
	inv := console.Invocation
	var b strings.Builder
	fmt.Fprintf(&b, "tuck %s   (context: %s, source: %s)\n\n", inv.Command, inv.Context, p.Source)
	fmt.Fprintf(&b, "%s %s\n", console.Style(output.StyleAccent, "package:"), console.Style(output.StylePackage, p.Package))
	if p.ManifestPath != "" {
		fmt.Fprintf(&b, "%s %s\n", console.Style(output.StyleAccent, "manifest:"), console.Style(output.StylePath, p.ManifestPath))
	}
	if len(p.Files) > 0 {
		fmt.Fprintln(&b)
	}
	for _, file := range p.Files {
		configured := "default"
		if file.Configured {
			configured = "configured"
		}
		mode := ""
		if file.Mode != "" {
			mode = " mode=" + file.Mode
		}
		fmt.Fprintf(&b, "%s deploy=%s%s %s\n", file.Path, file.Deploy, mode, console.Style(output.StyleMuted, configured))
	}
	fmt.Fprintf(&b, "\n%s\n", console.Style(output.StyleMuted, fmt.Sprintf("%d %s", len(p.Files), configEntryNoun(len(p.Files)))))
	return b.String(), nil
}

func configEntryNoun(count int) string {
	if count == 1 {
		return "entry"
	}
	return "entries"
}

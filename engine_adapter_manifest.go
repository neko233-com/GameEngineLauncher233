package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/egoist/mygo/ui"
)

// engineAdapterManifest is the on-disk format, versioned separately from the
// app. It is the declarative half of the adapter spec; process is the
// optional LSP-like half.
type engineAdapterManifest struct {
	SchemaVersion int                   `json:"schemaVersion"`
	ID            string                `json:"id"`
	Name          string                `json:"name"`
	Version       string                `json:"version"`
	Description   string                `json:"description,omitempty"`
	Homepage      string                `json:"homepage,omitempty"`
	IconSvg       string                `json:"iconSvg,omitempty"`
	BrandIconSvg  string                `json:"brandIconSvg,omitempty"`
	Features      engineAdapterFeatures `json:"features,omitempty"`
	URLs          engineAdapterURLs     `json:"urls,omitempty"`
	Pages         []enginePageManifest  `json:"pages"`
	Process       *engineAdapterProcess `json:"process,omitempty"`
	Source        string                `json:"source,omitempty"`
}

type enginePageManifest struct {
	ID     string            `json:"id"`
	Title  string            `json:"title"`
	Kind   string            `json:"kind"`
	Icon   string            `json:"icon,omitempty"`
	Groups []engineLinkGroup `json:"groups,omitempty"`
	URL    string            `json:"url,omitempty"`
}

var (
	engineAdapterIDPattern   = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,63}$`)
	engineAdapterPagePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)
	engineAdapterReserved    = map[string]bool{"unity": true, "tuanjie": true, "godot": true, "unreal": true, "system": true, "settings": true}
)

const (
	maxAdapterManifestBytes = 1 << 20
	maxAdapterIconBytes     = 128 << 10
)

// parseEngineAdapterManifest validates a declarative adapter and turns it into
// the runtime adapter. It never executes anything.
func parseEngineAdapterManifest(data []byte, source, dir string) (*engineAdapter, error) {
	if len(data) == 0 || len(data) > maxAdapterManifestBytes {
		return nil, fmt.Errorf("adapter manifest has an invalid size")
	}
	var m engineAdapterManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("adapter manifest: %w", err)
	}
	if m.SchemaVersion != engineAdapterSchemaVersion {
		return nil, fmt.Errorf("unsupported adapter schemaVersion %d (want %d)", m.SchemaVersion, engineAdapterSchemaVersion)
	}
	m.ID = strings.TrimSpace(m.ID)
	m.Name = strings.TrimSpace(m.Name)
	if !engineAdapterIDPattern.MatchString(m.ID) {
		return nil, fmt.Errorf("adapter id %q must be a lowercase slug", m.ID)
	}
	if engineAdapterReserved[m.ID] {
		return nil, fmt.Errorf("adapter id %q is reserved", m.ID)
	}
	if m.Name == "" || len(m.Name) > 80 {
		return nil, fmt.Errorf("adapter name must be 1-80 characters")
	}
	if m.Version == "" {
		m.Version = "0.1.0"
	}
	if len(m.Pages) == 0 || len(m.Pages) > 50 {
		return nil, fmt.Errorf("adapter needs 1-50 pages")
	}
	if m.IconSvg != "" && len(m.IconSvg) > maxAdapterIconBytes {
		return nil, fmt.Errorf("adapter icon is too large")
	}
	if m.BrandIconSvg != "" && len(m.BrandIconSvg) > maxAdapterIconBytes {
		return nil, fmt.Errorf("adapter brand icon is too large")
	}
	icon, err := parseAdapterIcon(m.IconSvg)
	if err != nil {
		return nil, fmt.Errorf("adapter icon: %w", err)
	}
	brand, err := parseAdapterIcon(m.BrandIconSvg)
	if err != nil {
		return nil, fmt.Errorf("adapter brand icon: %w", err)
	}
	if icon == nil {
		icon = iconEngine
	}
	if err := validateAdapterURLs(m.URLs); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	pages := make([]enginePage, 0, len(m.Pages))
	for i, p := range m.Pages {
		p.ID = strings.TrimSpace(p.ID)
		p.Title = strings.TrimSpace(p.Title)
		p.Kind = strings.TrimSpace(p.Kind)
		if !engineAdapterPagePattern.MatchString(p.ID) {
			return nil, fmt.Errorf("page %d has invalid id %q", i+1, p.ID)
		}
		if seen[p.ID] {
			return nil, fmt.Errorf("duplicate page id %q", p.ID)
		}
		seen[p.ID] = true
		if p.Title == "" || len(p.Title) > 80 {
			return nil, fmt.Errorf("page %q needs a 1-80 character title", p.ID)
		}
		if !enginePageKinds[p.Kind] {
			return nil, fmt.Errorf("page %q has unknown kind %q", p.ID, p.Kind)
		}
		if p.URL != "" {
			if err := validateAdapterURL(p.URL); err != nil {
				return nil, fmt.Errorf("page %q: %w", p.ID, err)
			}
		}
		if len(p.Groups) > 20 {
			return nil, fmt.Errorf("page %q has too many link groups", p.ID)
		}
		for gi := range p.Groups {
			if p.Groups[gi].Title == "" {
				return nil, fmt.Errorf("page %q group %d has no title", p.ID, gi+1)
			}
			if len(p.Groups[gi].Links) > 50 {
				return nil, fmt.Errorf("page %q group %q has too many links", p.ID, p.Groups[gi].Title)
			}
			for li := range p.Groups[gi].Links {
				l := &p.Groups[gi].Links[li]
				l.Title = strings.TrimSpace(l.Title)
				if l.Title == "" {
					return nil, fmt.Errorf("page %q group %q link %d has no title", p.ID, p.Groups[gi].Title, li+1)
				}
				if err := validateAdapterURL(l.URL); err != nil {
					return nil, fmt.Errorf("page %q link %q: %w", p.ID, l.Title, err)
				}
			}
		}
		pages = append(pages, enginePage{
			Key:    p.ID,
			Route:  externalPageRoute(m.ID, p.ID),
			Kind:   p.Kind,
			Icon:   adapterIcon(p.Icon),
			Groups: p.Groups,
			URL:    p.URL,
			title:  func(title string) func() string { return func() string { return title } }(p.Title),
		})
	}
	if m.Process != nil {
		m.Process.Command = strings.TrimSpace(m.Process.Command)
		if m.Process.Command == "" {
			return nil, fmt.Errorf("process command is empty")
		}
		if len(m.Process.Args) > 20 {
			return nil, fmt.Errorf("process has too many arguments")
		}
		for _, arg := range m.Process.Args {
			if len(arg) > 1024 {
				return nil, fmt.Errorf("process argument is too long")
			}
		}
		if dir != "" && !filepath.IsAbs(m.Process.Command) {
			root, err := filepath.Abs(dir)
			if err != nil {
				return nil, err
			}
			target, err := filepath.Abs(filepath.Join(root, filepath.FromSlash(m.Process.Command)))
			if err != nil || !strings.HasPrefix(target, root+string(os.PathSeparator)) {
				return nil, fmt.Errorf("process command must stay inside the adapter folder")
			}
			m.Process.Command = target
		}
	}
	return &engineAdapter{
		ID:          m.ID,
		Version:     m.Version,
		Description: strings.TrimSpace(m.Description),
		Homepage:    strings.TrimSpace(m.Homepage),
		Source:      strings.TrimSpace(source),
		Icon:        icon,
		BrandIcon:   brand,
		Pages:       pages,
		Features:    m.Features,
		URLs:        m.URLs,
		Process:     m.Process,
		Dir:         dir,
		name:        func() string { return m.Name },
	}, nil
}

func parseAdapterIcon(raw string) (*ui.SVG, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if !strings.HasPrefix(raw, "<svg") {
		return nil, fmt.Errorf("iconSvg must be an SVG document")
	}
	icon, err := ui.ParseSVG([]byte(raw))
	if err != nil {
		return nil, err
	}
	if w, h := icon.Size(); w <= 0 || h <= 0 {
		return nil, fmt.Errorf("iconSvg has no drawable size")
	}
	return icon, nil
}

func validateAdapterURLs(u engineAdapterURLs) error {
	for name, raw := range map[string]string{
		"assetStore": u.AssetStore, "docs": u.Docs, "license": u.License,
		"learn": u.Learn, "community": u.Community,
	} {
		if raw == "" {
			continue
		}
		if err := validateAdapterURL(raw); err != nil {
			return fmt.Errorf("urls.%s: %w", name, err)
		}
	}
	return nil
}

func validateAdapterURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("URL is empty")
	}
	if len(raw) > 2048 {
		return fmt.Errorf("URL is too long")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("URL must use http or https")
	}
	if u.Host == "" {
		return fmt.Errorf("URL has no host")
	}
	return nil
}

// adapterManifestBytes marshals a manifest for an imported adapter so the
// source and normalized fields survive a reload.
func adapterManifestBytes(data []byte) []byte {
	var m engineAdapterManifest
	if json.Unmarshal(data, &m) != nil {
		return data
	}
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return data
	}
	return append(out, '\n')
}

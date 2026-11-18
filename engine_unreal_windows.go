//go:build windows

package main

import (
	"golang.org/x/sys/windows/registry"
)

// unrealRegistryDirs reads the Epic Games Launcher's per-engine registry keys.
// The LauncherInstalled.dat manifest is the primary source; the registry is a
// fallback for machines where the manifest was cleared or moved.
func unrealRegistryDirs() []string {
	var out []string
	for _, path := range []string{
		`SOFTWARE\EpicGames\Unreal Engine`,
		`SOFTWARE\WOW6432Node\EpicGames\Unreal Engine`,
	} {
		parent, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.READ)
		if err != nil {
			continue
		}
		names, err := parent.ReadSubKeyNames(-1)
		parent.Close()
		if err != nil {
			continue
		}
		for _, name := range names {
			sub, err := registry.OpenKey(registry.LOCAL_MACHINE, path+`\`+name, registry.READ)
			if err != nil {
				continue
			}
			dir, _, err := sub.GetStringValue("InstalledDirectory")
			sub.Close()
			if err == nil && dir != "" {
				out = append(out, dir)
			}
		}
	}
	return dedupe(out)
}

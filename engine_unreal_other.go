//go:build !windows

package main

// unrealRegistryDirs has no registry source outside Windows. The launcher
// manifest and the usual install folders are still scanned.
func unrealRegistryDirs() []string { return nil }

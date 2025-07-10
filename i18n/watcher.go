package i18n

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// WatchLocales watches the locale directory for changes and reloads changed files.
// This should be called once, typically in development mode.
func (m *Manager) WatchLocales(dir string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	// Watch the directory
	if err := watcher.Add(dir); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					filename := filepath.Base(event.Name)
					if strings.HasSuffix(filename, ".json") || strings.HasSuffix(filename, ".toml") || strings.HasSuffix(filename, ".yaml") {
						localeCode := strings.TrimSuffix(filename, filepath.Ext(filename))
						log.Printf("[i18n] Reloading locale: %s", localeCode)
						m.mu.Lock()
						_ = m.loadLocaleFile(localeCode, event.Name)
						m.mu.Unlock()
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("[i18n] Watcher error: %v", err)
			}
		}
	}()

	return nil
}

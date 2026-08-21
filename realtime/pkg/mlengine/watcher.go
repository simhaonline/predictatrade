// Package mlengine — Model file watcher for hot-reload.
//
// Uses fsnotify to watch the models/ directory. When ONNX model files
// are written or created (e.g., by the training pipeline), the watcher
// triggers an atomic session swap via Reload().
package mlengine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ModelWatcher watches the models directory for changes and triggers
// hot-reload of ONNX sessions in a thread-safe manner.
type ModelWatcher struct {
	engine   *MLEngine
	watcher  *fsnotify.Watcher
	modelsDir string
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewModelWatcher creates a new file watcher for the models directory.
// Returns nil if the directory doesn't exist or fsnotify fails.
func NewModelWatcher(engine *MLEngine, modelsDir string) *ModelWatcher {
	if engine == nil || !engine.IsEnabled() {
		return nil
	}
	if _, err := os.Stat(modelsDir); os.IsNotExist(err) {
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("[mlengine] fsnotify watcher creation failed: %v\n", err)
		return nil
	}

	w := &ModelWatcher{
		engine:    engine,
		watcher:   watcher,
		modelsDir: modelsDir,
		stopCh:    make(chan struct{}),
	}

	w.wg.Add(1)
	go w.loop()

	return w
}

// loop runs the watcher event loop.
func (w *ModelWatcher) loop() {
	defer w.wg.Done()

	// Add directory to watcher
	if err := w.watcher.Add(w.modelsDir); err != nil {
		fmt.Printf("[mlengine] fsnotify add watch failed: %v\n", err)
		return
	}

	// Debounce timer — avoid reloading on rapid file writes
	var debounceTimer *time.Timer
	debounceDelay := 500 * time.Millisecond

	for {
		select {
		case <-w.stopCh:
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			// Only handle .onnx file writes/creates
			if !strings.HasSuffix(event.Name, ".onnx") {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			// Debounce: reset timer on each event
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(debounceDelay, func() {
				fmt.Printf("[mlengine] Model file changed: %s — reloading\n", filepath.Base(event.Name))
				w.engine.Reload()
			})

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("[mlengine] fsnotify error: %v\n", err)
		}
	}
}

// Close stops the watcher and releases resources.
func (w *ModelWatcher) Close() {
	if w == nil {
		return
	}
	close(w.stopCh)
	w.watcher.Close()
	w.wg.Wait()
}

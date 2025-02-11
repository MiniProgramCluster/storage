package filecache

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"storage/log"

	"github.com/fsnotify/fsnotify"
)

// FileCache is a simple in-memory cache for file contents
type FileCache struct {
	mu      sync.RWMutex
	cache   map[string][]byte
	watcher *fsnotify.Watcher
}

// NewFileCache creates a new FileCache
func New() *FileCache {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Logger().Error().Err(err).Msgf("create watcher failed")
		return nil
	}
	fc := &FileCache{
		cache:   make(map[string][]byte),
		watcher: watcher,
	}
	go func() {
		for event := range watcher.Events {
			filePath := strings.TrimPrefix(event.Name, "./")
			fc.mu.Lock()
			fc.cache[filePath] = nil
			log.Logger().Debug().Msgf("event:%s, filePath:%s clear it", event.Name, filePath)
			fc.mu.Unlock()
		}
	}()
	return fc
}

// Get retrieves a file from the cache, loading it from disk if necessary
func (fc *FileCache) Get(filePath string) ([]byte, error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	data, found := fc.cache[filePath]
	if found {
		return data, nil
	}
	// Load file from disk
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Logger().Error().Err(err).Msgf("filePath:%s not found", filePath)
		return nil, err
	}
	filePath = strings.TrimPrefix(filePath, "./")
	fc.cache[filePath] = data
	fc.watcher.Add(filePath)
	return data, nil
}

func (fc *FileCache) Put(filePath string, data []byte) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.cache[filePath] = data
	fc.watcher.Add(filePath)
	dir := filepath.Dir(filePath)
	os.MkdirAll(dir, 0o755)
	return os.WriteFile(filePath, data, 0o644)
}

func (fc *FileCache) Delete(filePath string) error {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	delete(fc.cache, filePath)
	fc.watcher.Remove(filePath)
	return os.Remove(filePath)
}

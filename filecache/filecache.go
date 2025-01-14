package filecache

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"sync"

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
		log.Fatal(err)
		return nil
	}
	fc := &FileCache{
		cache:   make(map[string][]byte),
		watcher: watcher,
	}
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				filePath := strings.TrimPrefix(event.Name, "./")
				fc.mu.Lock()
				fc.cache[filePath] = nil
				fmt.Printf("event:%s, filePath:%s clear it\n", event.Name, filePath)
				fc.mu.Unlock()
			}
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
		fmt.Printf("filePath:%s not found\n", filePath)
		return nil, err
	}
	filePath = strings.TrimPrefix(filePath, "./")
	fc.cache[filePath] = data
	fc.watcher.Add(filePath)
	return data, nil
}

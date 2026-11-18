package tracking

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"
)

// FilePersistence periodically writes usage data to a JSON file.
type FilePersistence struct {
	store    *UsageStore
	filePath string
	interval time.Duration
	done     chan struct{}
	wg       sync.WaitGroup
}

// NewFilePersistence creates a new file persistence layer.
func NewFilePersistence(store *UsageStore, filePath string, interval time.Duration) *FilePersistence {
	return &FilePersistence{
		store:    store,
		filePath: filePath,
		interval: interval,
		done:     make(chan struct{}),
	}
}

// Start begins the periodic persistence loop.
func (fp *FilePersistence) Start() {
	fp.wg.Add(1)
	go func() {
		defer fp.wg.Done()
		ticker := time.NewTicker(fp.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				fp.save()
			case <-fp.done:
				fp.save() // Save on shutdown
				return
			}
		}
	}()
}

// Stop stops the persistence loop.
func (fp *FilePersistence) Stop() {
	close(fp.done)
	fp.wg.Wait()
}

// Load loads usage data from file on startup.
func (fp *FilePersistence) Load() error {
	data, err := os.ReadFile(fp.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // First run, no data yet
		}
		return err
	}

	var storeData StoreData
	if err := json.Unmarshal(data, &storeData); err != nil {
		return err
	}

	// Replay records into the store
	for _, record := range storeData.Records {
		fp.store.Tracker.Record(record)
	}

	return nil
}

func (fp *FilePersistence) save() {
	data := fp.store.GetData()
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}

	// Ensure directory exists
	dir := ""
	if idx := strings.LastIndex(fp.filePath, string(os.PathSeparator)); idx >= 0 {
		dir = fp.filePath[:idx]
	}
	if dir != "" {
		os.MkdirAll(dir, 0755)
	}

	os.WriteFile(fp.filePath, jsonData, 0644)
}

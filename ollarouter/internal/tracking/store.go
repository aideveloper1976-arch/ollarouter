package tracking

// UsageStore wraps the Tracker for in-memory usage data storage.
type UsageStore struct {
	*Tracker
}

// NewUsageStore creates a new in-memory usage store.
func NewUsageStore() *UsageStore {
	return &UsageStore{
		Tracker: NewTracker(),
	}
}

// GetData returns the complete data set for persistence.
func (s *UsageStore) GetData() *StoreData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &StoreData{
		Records:   s.records,
		Providers: s.providers,
		Models:    s.models,
	}
}

// StoreData is the serializable format for persistence.
type StoreData struct {
	Records   []UsageRecord             `json:"records"`
	Providers map[string]*providerStats `json:"providers"`
	Models    map[string]*modelStats    `json:"models"`
}

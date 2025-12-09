package proto

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/heytom-labs/heytom-gateway/internal/config"
)

// HotReloadManager manages hot reload of protosets
type HotReloadManager struct {
	loader        *DescriptorLoader
	config        *config.ProtoHotReloadConfig
	protosets     map[string]*config.ProtoSetInfo
	ticker        *time.Ticker
	stopCh        chan struct{}
	wg            sync.WaitGroup
	httpClient    *http.Client
	msgCacheClear func() // Callback to clear message cache
	mu            sync.RWMutex
}

// NewHotReloadManager creates a new hot reload manager
func NewHotReloadManager(
	loader *DescriptorLoader,
	cfg *config.ProtoHotReloadConfig,
	protosets []config.ProtoSetInfo,
) *HotReloadManager {
	protosetMap := make(map[string]*config.ProtoSetInfo)
	for i := range protosets {
		protosetMap[protosets[i].ServiceName] = &protosets[i]
	}

	return &HotReloadManager{
		loader:    loader,
		config:    cfg,
		protosets: protosetMap,
		stopCh:    make(chan struct{}),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetMessageCacheClearFunc sets the callback to clear message cache when proto is updated
func (m *HotReloadManager) SetMessageCacheClearFunc(fn func()) {
	m.msgCacheClear = fn
}

// Start starts the hot reload process
func (m *HotReloadManager) Start(ctx context.Context) error {
	if !m.config.Enabled {
		return nil
	}

	if m.config.CheckPeriod <= 0 {
		return fmt.Errorf("check period must be greater than 0")
	}

	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.ticker = time.NewTicker(time.Duration(m.config.CheckPeriod) * time.Second)
		defer m.ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-m.stopCh:
				return
			case <-m.ticker.C:
				m.checkAndReload()
			}
		}
	}()

	return nil
}

// Stop stops the hot reload process
func (m *HotReloadManager) Stop() {
	close(m.stopCh)
	m.wg.Wait()
}

// checkAndReload checks for updates and reloads protosets if necessary
func (m *HotReloadManager) checkAndReload() {
	m.mu.RLock()
	protosets := make([]config.ProtoSetInfo, 0, len(m.protosets))
	for _, ps := range m.protosets {
		protosets = append(protosets, *ps)
	}
	m.mu.RUnlock()

	for _, ps := range protosets {
		if err := m.reloadProtoset(&ps); err != nil {
			fmt.Printf("Failed to reload protoset for service %s: %v\n", ps.ServiceName, err)
		}
	}
}

// reloadProtoset reloads a single protoset
func (m *HotReloadManager) reloadProtoset(info *config.ProtoSetInfo) error {
	// If URL is provided, download from artifact repository
	if info.URL != "" {
		tempFile, err := m.downloadProtoset(info.URL)
		if err != nil {
			return fmt.Errorf("failed to download protoset from %s: %w", info.URL, err)
		}
		defer os.Remove(tempFile)

		// Load the downloaded file
		data, err := os.ReadFile(tempFile)
		if err != nil {
			return fmt.Errorf("failed to read downloaded protoset: %w", err)
		}

		if err := m.loader.LoadProtosetData(data); err != nil {
			return fmt.Errorf("failed to load protoset data: %w", err)
		}
	} else if info.Path != "" {
		// Load from local file
		if err := m.loader.LoadProtoset(info.Path); err != nil {
			return fmt.Errorf("failed to load protoset from %s: %w", info.Path, err)
		}
	}

	// Clear message cache after loading new protosets
	if m.msgCacheClear != nil {
		m.msgCacheClear()
	}

	fmt.Printf("Successfully reloaded protoset for service: %s\n", info.ServiceName)
	return nil
}

// downloadProtoset downloads a protoset file from the artifact repository
func (m *HotReloadManager) downloadProtoset(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add auth token if configured
	if m.config.AuthToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.config.AuthToken))
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download protoset: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status code %d", resp.StatusCode)
	}

	// Create temporary file
	tempFile, err := os.CreateTemp("", "protoset-*.pb")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tempFile.Close()

	// Download file content
	if _, err := io.Copy(tempFile, resp.Body); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	return tempFile.Name(), nil
}

// ReloadServiceProtoset manually reloads a specific service's protoset
func (m *HotReloadManager) ReloadServiceProtoset(serviceName string) error {
	m.mu.RLock()
	ps, ok := m.protosets[serviceName]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("protoset not found for service: %s", serviceName)
	}

	return m.reloadProtoset(ps)
}

// RegisterProtoset registers a new protoset for hot reload
func (m *HotReloadManager) RegisterProtoset(info config.ProtoSetInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.protosets[info.ServiceName] = &info
}

// UnregisterProtoset unregisters a protoset from hot reload
func (m *HotReloadManager) UnregisterProtoset(serviceName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.protosets, serviceName)
}

// GetRegisteredProtosets returns all registered protosets
func (m *HotReloadManager) GetRegisteredProtosets() []config.ProtoSetInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	protosets := make([]config.ProtoSetInfo, 0, len(m.protosets))
	for _, ps := range m.protosets {
		protosets = append(protosets, *ps)
	}
	return protosets
}

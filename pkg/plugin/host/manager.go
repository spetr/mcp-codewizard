// Package host provides the plugin host for loading external plugins.
package host

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"

	"github.com/spetr/mcp-codewizard/pkg/plugin/shared"
)

// Manager manages external plugins.
type Manager struct {
	pluginsDir string
	plugins    map[string]*LoadedPlugin
	mu         sync.RWMutex
	logger     hclog.Logger
}

// LoadedPlugin represents a loaded plugin.
type LoadedPlugin struct {
	Name       string
	Type       shared.PluginType
	Path       string
	Client     *plugin.Client
	Embedding  shared.EmbeddingProvider
	Reranker   shared.RerankerProvider
}

// NewManager creates a new plugin manager.
func NewManager(pluginsDir string) *Manager {
	// Create a hashicorp logger that wraps slog
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "plugins",
		Level:  hclog.Warn,
		Output: os.Stderr,
	})

	return &Manager{
		pluginsDir: pluginsDir,
		plugins:    make(map[string]*LoadedPlugin),
		logger:     logger,
	}
}

// DiscoverPlugins discovers available plugins in the plugins directory.
func (m *Manager) DiscoverPlugins() ([]string, error) {
	if _, err := os.Stat(m.pluginsDir); os.IsNotExist(err) {
		return nil, nil // No plugins directory
	}

	entries, err := os.ReadDir(m.pluginsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugins directory: %w", err)
	}

	var plugins []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(m.pluginsDir, entry.Name())
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		// Check if executable
		if info.Mode()&0111 != 0 {
			plugins = append(plugins, entry.Name())
		}
	}

	return plugins, nil
}

// LoadPlugin loads a plugin by name.
func (m *Manager) LoadPlugin(name string, pluginType shared.PluginType) (*LoadedPlugin, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already loaded
	if p, exists := m.plugins[name]; exists {
		return p, nil
	}

	pluginPath := filepath.Join(m.pluginsDir, name)
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}

	slog.Info("loading plugin", "name", name, "type", pluginType, "path", pluginPath)

	// Create the plugin client
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: shared.Handshake,
		Plugins:         shared.PluginMap,
		Cmd:             exec.Command(pluginPath),
		Logger:          m.logger,
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC,
		},
	})

	// Connect to the plugin
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to connect to plugin: %w", err)
	}

	// Request the plugin
	raw, err := rpcClient.Dispense(string(pluginType))
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to dispense plugin: %w", err)
	}

	loaded := &LoadedPlugin{
		Name:   name,
		Type:   pluginType,
		Path:   pluginPath,
		Client: client,
	}

	// Cast to the appropriate type
	switch pluginType {
	case shared.PluginTypeEmbedding:
		provider, ok := raw.(shared.EmbeddingProvider)
		if !ok {
			client.Kill()
			return nil, fmt.Errorf("plugin does not implement EmbeddingProvider")
		}
		loaded.Embedding = provider
	case shared.PluginTypeReranker:
		provider, ok := raw.(shared.RerankerProvider)
		if !ok {
			client.Kill()
			return nil, fmt.Errorf("plugin does not implement RerankerProvider")
		}
		loaded.Reranker = provider
	default:
		client.Kill()
		return nil, fmt.Errorf("unsupported plugin type: %s", pluginType)
	}

	m.plugins[name] = loaded
	slog.Info("plugin loaded", "name", name, "type", pluginType)

	return loaded, nil
}

// GetEmbeddingPlugin returns a loaded embedding plugin.
func (m *Manager) GetEmbeddingPlugin(name string) (shared.EmbeddingProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, exists := m.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin not loaded: %s", name)
	}

	if p.Embedding == nil {
		return nil, fmt.Errorf("plugin is not an embedding provider: %s", name)
	}

	return p.Embedding, nil
}

// GetRerankerPlugin returns a loaded reranker plugin.
func (m *Manager) GetRerankerPlugin(name string) (shared.RerankerProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, exists := m.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin not loaded: %s", name)
	}

	if p.Reranker == nil {
		return nil, fmt.Errorf("plugin is not a reranker provider: %s", name)
	}

	return p.Reranker, nil
}

// UnloadPlugin unloads a plugin.
func (m *Manager) UnloadPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, exists := m.plugins[name]
	if !exists {
		return nil
	}

	// Close the provider first
	if p.Embedding != nil {
		p.Embedding.Close()
	}
	if p.Reranker != nil {
		p.Reranker.Close()
	}

	// Kill the plugin process
	p.Client.Kill()

	delete(m.plugins, name)
	slog.Info("plugin unloaded", "name", name)

	return nil
}

// UnloadAll unloads all plugins.
func (m *Manager) UnloadAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, p := range m.plugins {
		if p.Embedding != nil {
			p.Embedding.Close()
		}
		if p.Reranker != nil {
			p.Reranker.Close()
		}
		p.Client.Kill()
		slog.Debug("plugin unloaded", "name", name)
	}

	m.plugins = make(map[string]*LoadedPlugin)
}

// ListLoaded returns a list of loaded plugins.
func (m *Manager) ListLoaded() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var names []string
	for name := range m.plugins {
		names = append(names, name)
	}
	return names
}

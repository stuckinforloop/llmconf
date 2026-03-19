package providers

// Provider defines the interface for LLM provider implementations
type Provider interface {
	Name() string
	DisplayName() string
	Description() string
	DocumentationURL() string
	RequiredEnvVars() []EnvVarSpec
	OptionalEnvVars() []EnvVarSpec
	SupportsModelPinning() bool
	DefaultModels() map[string]string
	ModelSuggestions() map[string][]string

	// GenerateEnv generates environment variables for this provider
	GenerateEnv(config ProviderConfig) (map[string]string, error)

	// Validate validates provider configuration
	Validate(config ProviderConfig) error

	// ValidateModel validates a model ID for this provider
	ValidateModel(modelType string, modelID string) error
}

// EnvVarSpec defines an environment variable specification
type EnvVarSpec struct {
	Name        string
	Description string
	Sensitive   bool
	Validate    func(value string) error
}

// ProviderConfig holds configuration for a provider
type ProviderConfig struct {
	Credentials  map[string]string
	Models       map[string]string
	ExtraEnv     map[string]string
	AuthMethod   string
}

// ProviderRegistry manages available providers
type ProviderRegistry struct {
	providers map[string]Provider
}

// NewRegistry creates a new provider registry with all providers
func NewRegistry() *ProviderRegistry {
	registry := &ProviderRegistry{
		providers: make(map[string]Provider),
	}

	// Register all providers
	registry.Register(NewAnthropic())
	registry.Register(NewBedrock())
	registry.Register(NewVertex())
	registry.Register(NewFoundry())
	registry.Register(NewFireworks())
	registry.Register(NewLiteLLM())

	return registry
}

// Register adds a provider to the registry
func (r *ProviderRegistry) Register(provider Provider) {
	r.providers[provider.Name()] = provider
}

// Get retrieves a provider by name
func (r *ProviderRegistry) Get(name string) (Provider, bool) {
	provider, ok := r.providers[name]
	return provider, ok
}

// List returns all registered providers
func (r *ProviderRegistry) List() []Provider {
	result := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		result = append(result, p)
	}
	return result
}

// Names returns all provider names
func (r *ProviderRegistry) Names() []string {
	result := make([]string, 0, len(r.providers))
	for name := range r.providers {
		result = append(result, name)
	}
	return result
}

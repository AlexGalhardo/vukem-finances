package ai

import "fmt"

// NewProvider returns the production Factory. allowPrivate is
// AI_ALLOW_PRIVATE_HOSTS in the personal edition and false in saas; it reaches
// both the base URL check and the connect-time address check.
func NewProvider(allowPrivate bool) Factory {
	h := NewHTTP(HTTPOptions{AllowPrivate: allowPrivate})

	return func(cfg ProviderConfig) (Provider, error) {
		switch cfg.Kind {
		case KindAnthropic:
			return NewAnthropic(cfg.Key, cfg.Model, h), nil
		case KindOpenAI:
			return NewOpenAI(cfg.Key, cfg.Model, h), nil
		case KindOpenAICompatible:
			if err := CheckBaseURL(cfg.Kind, cfg.BaseURL, allowPrivate); err != nil {
				return nil, err
			}

			return NewCompatible(cfg.Key, cfg.BaseURL, cfg.Model, h), nil
		default:
			return nil, fmt.Errorf("ai: unknown provider %q", cfg.Kind)
		}
	}
}

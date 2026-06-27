package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				Database: DatabaseConfig{Host: "localhost", Name: "test"},
				Brands:   []BrandConfig{{Name: "Brand1"}},
			},
			wantErr: false,
		},
		{
			name: "missing database host",
			cfg: Config{
				Database: DatabaseConfig{Name: "test"},
				Brands:   []BrandConfig{{Name: "Brand1"}},
			},
			wantErr: true,
		},
		{
			name: "missing brands",
			cfg: Config{
				Database: DatabaseConfig{Host: "localhost", Name: "test"},
				Brands:   []BrandConfig{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

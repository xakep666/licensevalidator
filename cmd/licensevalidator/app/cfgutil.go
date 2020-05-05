package app

import (
	"fmt"
	"net/url"
	"os"

	"github.com/pelletier/go-toml"
)

// MaskedString is a string that will be always represented as "****" in json
type MaskedString string

func (m MaskedString) MarshalJSON() ([]byte, error) {
	return []byte(`"****"`), nil
}

// MaskedURL is an url that will be represented with masked password in json
type MaskedURL string

func (m MaskedURL) MarshalJSON() ([]byte, error) {
	u, err := url.Parse(string(m))
	if err != nil {
		return nil, err
	}

	if _, hasPass := u.User.Password(); hasPass {
		u.User = url.UserPassword(u.User.Username(), "****")
	}

	return []byte(fmt.Sprintf(`"%s"`, u.String())), nil
}

func ConfigFromFile(cfgFilePath string) (Config, error) {
	var cfg Config

	cfgFile, err := os.OpenFile(cfgFilePath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return cfg, fmt.Errorf("config open failed: %w", err)
	}

	defer cfgFile.Close()

	if err := toml.NewDecoder(cfgFile).Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("config parsee failed: %w", err)
	}

	return cfg, nil
}

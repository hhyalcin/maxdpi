package engine

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
)

// Config: kalici ayar. Yalnizca profil ADI saklanir; motor argumanlari asla dosyadan okunmaz.
type Config struct {
	Profile string `json:"profile"`
}

func configPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "discord-unblock", "config.json"), nil
}

// LoadConfig: dosya yok/bozuk/asiri buyukse bos Config doner (hata kullaniciya yansimaz).
func LoadConfig() Config {
	p, err := configPath()
	if err != nil {
		return Config{}
	}
	return loadConfigFrom(p)
}

func loadConfigFrom(path string) Config {
	// #nosec G304 -- yol UserConfigDir + sabit ad (testte gecici dizin)
	f, err := os.Open(path)
	if err != nil {
		return Config{}
	}
	defer func() { _ = f.Close() }()
	var c Config
	if err := json.NewDecoder(io.LimitReader(f, 4<<10)).Decode(&c); err != nil {
		return Config{}
	}
	return c
}

func SaveConfig(c Config) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	return saveConfigTo(p, c)
}

// saveConfigTo: gecici dosyaya yazip yeniden adlandirir (yarim yazilmis dosya kalmaz).
func saveConfigTo(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// OrderProfiles: kayitli profil (varsa ve listede taniniyorsa) one alinir, kalanlar mevcut sirada izler.
func OrderProfiles(all []Profile, preferred string) []Profile {
	out := make([]Profile, 0, len(all))
	for _, p := range all {
		if p.Name == preferred {
			out = append(out, p)
		}
	}
	for _, p := range all {
		if p.Name != preferred {
			out = append(out, p)
		}
	}
	return out
}

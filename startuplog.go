package main

import (
	"os"
	"path/filepath"
	"time"
)

// logStartupError: pencere acilamazsa nedeni <UserCacheDir>/maxdpi/startup.log dosyasina yazar.
func logStartupError(err error) {
	base, derr := os.UserCacheDir()
	if derr != nil {
		return
	}
	dir := filepath.Join(base, "maxdpi")
	if os.MkdirAll(dir, 0o700) != nil {
		return
	}
	// #nosec G304 -- yol UserCacheDir + sabit ad
	f, ferr := os.OpenFile(filepath.Join(dir, "startup.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if ferr != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = f.WriteString(time.Now().Format(time.RFC3339) + " " + err.Error() + "\n")
}

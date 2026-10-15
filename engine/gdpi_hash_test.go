package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// THIRD_PARTY.md ile ayni: GoodbyeDPI-Turkey release-0.2.3rc3-turkey, x86_64/. Gomulu ikililer bu hash'lerden
// sapmamali (yonetici yetkisiyle calisirlar); degistirirken THIRD_PARTY.md'yi de guncelle.
var gdpiHashes = map[string]string{
	"goodbyedpi.exe":  "8d412b094bb9c137ff25ba9a794d1122ecc84bb776debff6c249723a13cc31cd",
	"WinDivert.dll":   "6110bfa44667405179c3e15e12af1b62037e447ed59b054b19042032995e6c7e",
	"WinDivert64.sys": "e69b5ba3f0cd6cfb2983e442636e7f0b342b61b15264b0328317d4559c82cf50",
}

func TestGDPIEmbeddedHashes(t *testing.T) {
	for name, want := range gdpiHashes {
		b, err := os.ReadFile(filepath.Join("gdpi_windows", name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		sum := sha256.Sum256(b)
		if got := hex.EncodeToString(sum[:]); got != want {
			t.Errorf("%s: sha256 %s, beklenen %s", name, got, want)
		}
	}
}

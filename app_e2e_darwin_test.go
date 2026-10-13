//go:build darwin

package main

import (
	"context"
	"os"
	"testing"

	"maxdpi/engine"
)

// Gercek engelli hatta: MAXDPI_E2E=1 go test . -run E2E -v
// HOME gecici dizine alinir; kullanicinin gercek ayarlarina ve onbellegine dokunulmaz.
func TestE2EProfilePersistence(t *testing.T) {
	if os.Getenv("MAXDPI_E2E") != "1" {
		t.Skip("MAXDPI_E2E=1 degil")
	}
	t.Setenv("HOME", t.TempDir())

	// Kayitli profil yokken: ilk calisan profil secilmeli ve kaydedilmeli.
	a := NewApp()
	a.ctx = context.Background()
	st, err := a.Connect()
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Logf("ilk baglanti profili: %s", st.Profile)
	if got := engine.LoadConfig().Profile; got != st.Profile {
		t.Fatalf("profil kaydedilmedi: kayit=%q, secilen=%q", got, st.Profile)
	}
	a.Disconnect()

	// Varsayilanda ilk olmayan bir profil kaydet: yeni oturum onunla baglanmali.
	want := engine.DefaultProfiles[len(engine.DefaultProfiles)-1].Name
	if err := engine.SaveConfig(engine.Config{Profile: want}); err != nil {
		t.Fatal(err)
	}
	b := NewApp()
	b.ctx = context.Background()
	st2, err := b.Connect()
	if err != nil {
		t.Fatalf("2. Connect: %v", err)
	}
	defer b.Disconnect()
	if st2.Profile != want {
		t.Errorf("kayitli profil once denenmedi: secilen=%q, beklenen=%q", st2.Profile, want)
	}

	// Bozuk/taninmayan kayit: bagimsiz calismaya devam etmeli.
	b.Disconnect()
	if err := engine.SaveConfig(engine.Config{Profile: "yok-boyle-profil"}); err != nil {
		t.Fatal(err)
	}
	c := NewApp()
	c.ctx = context.Background()
	st3, err := c.Connect()
	if err != nil {
		t.Fatalf("taninmayan kayitla Connect: %v", err)
	}
	defer c.Disconnect()
	if st3.Profile != engine.DefaultProfiles[0].Name {
		t.Logf("not: taninmayan kayitta secilen=%q (varsayilan ilk=%q)", st3.Profile, engine.DefaultProfiles[0].Name)
	}
}

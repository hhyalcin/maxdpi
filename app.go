package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"maxdpi/engine"

	"github.com/wailsapp/wails/v2/pkg/options"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App: arayuzun cagirdigi Baglan/Kes mantigi.
type App struct {
	ctx context.Context

	mu       sync.Mutex
	eng      engine.Engine
	front    *engine.Frontend
	resolver *engine.Resolver
	profile  string
	sys      bool // sistem geneli motor (GoodbyeDPI) bagli; yerel onyuz yok
}

func NewApp() *App {
	return &App{eng: newEngine(), resolver: engine.NewResolver()}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	engine.RecoverSystemProxy() // onceki oturum cokmusse kalan proxy ayarini temizle
}

// secondInstance: ikinci kez acilirsa yeni ornek acilmaz (yetim motor temizligi calisan motoru oldurmesin); mevcut pencere one gelir.
func (a *App) secondInstance(options.SecondInstanceData) {
	wruntime.WindowUnminimise(a.ctx)
	wruntime.Show(a.ctx)
}

// shutdown: pencere kapaninca motoru ve onyuzu temizler.
func (a *App) shutdown(ctx context.Context) { a.Disconnect() }

// Status: arayuz icin durum.
type Status struct {
	Connected bool   `json:"connected"`
	Profile   string `json:"profile"`
}

func (a *App) Status() Status {
	a.mu.Lock()
	defer a.mu.Unlock()
	return Status{Connected: a.front != nil || a.sys, Profile: a.profile}
}

// Connect: profilleri sirayla dener, ilk calisanla onyuzu baslatir.
func (a *App) Connect() (Status, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.front != nil || a.sys {
		return Status{Connected: true, Profile: a.profile}, nil
	}

	sys := engine.IsSystemWide(a.eng)
	if !sys {
		// DoH profilden bagimsizdir: erisilemiyorsa strateji denemenin anlami yok (sonuc onbellege girer, Probe yeniden kullanir).
		dctx, dcancel := context.WithTimeout(a.ctx, 14*time.Second) // 4 DoH sunucusu x ~3 sn
		_, derr := a.resolver.Lookup(dctx, "discord.com")
		dcancel()
		if derr != nil {
			return Status{}, errors.New("DNS-over-HTTPS erisilemiyor: internet baglantisini kontrol edin")
		}
	}

	cfg := engine.LoadConfig()
	var lastErr error // gercek neden (Defender, port, yazma hatasi) kullaniciya ulassin
	for _, p := range engine.OrderProfiles(engine.DefaultProfiles, cfg.Profile) {
		addr, err := a.eng.Start(p)
		if err != nil {
			lastErr = err
			continue
		}
		ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
		if sys {
			time.Sleep(time.Second) // DNS onbellegi temizlensin, filtre otursun
			err = engine.ProbeDirect(ctx)
		} else {
			err = engine.Probe(ctx, addr, a.resolver)
		}
		cancel()
		if err != nil {
			lastErr = err
			_ = a.eng.Stop()
			continue
		}
		if sys {
			a.sys, a.profile = true, p.Name
			if cfg.Profile != p.Name {
				_ = engine.SaveConfig(engine.Config{Profile: p.Name})
			}
			return Status{Connected: true, Profile: p.Name}, nil
		}
		f, err := engine.StartFrontend(addr, a.resolver)
		if err != nil {
			_ = a.eng.Stop()
			return Status{}, errors.New("yerel proxy baslatilamadi")
		}
		a.front, a.profile = f, p.Name
		if cfg.Profile != p.Name {
			_ = engine.SaveConfig(engine.Config{Profile: p.Name}) // kayit basarisizsa bir sonraki acilista yeniden denenir
		}
		return Status{Connected: true, Profile: p.Name}, nil
	}
	if lastErr != nil {
		return Status{}, fmt.Errorf("hicbir strateji calismadi (son hata: %v)", lastErr)
	}
	return Status{}, errors.New("hicbir strateji calismadi")
}

// Disconnect: onyuzu ve motoru kapatir.
func (a *App) Disconnect() Status {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.front != nil {
		a.front.Close()
		a.front = nil
	}
	_ = a.eng.Stop()
	a.sys = false
	engine.RestoreSystemProxy()
	engine.RestoreDiscordSettings()
	a.profile = ""
	return Status{}
}

// OpenDiscord: Discord'u onyuz proxy'si ile baslatir (bagli olmalidir).
func (a *App) OpenDiscord() error {
	a.mu.Lock()
	f, sys := a.front, a.sys
	a.mu.Unlock()
	if sys {
		return engine.LaunchDiscordDirect()
	}
	if f == nil {
		return errors.New("once baglanin")
	}
	return engine.LaunchDiscord(f.Addr())
}

// links: arayuzden acilabilen harici adresler. Arayuz yalnizca kimlik gonderir, keyfi URL kabul edilmez.
var links = map[string]string{
	"site":      "https://maxicode.com.tr/",
	"instagram": "https://www.instagram.com/hh_yalcin/",
}

// OpenLink: sabit tablodaki adresi varsayilan tarayicida acar.
func (a *App) OpenLink(id string) error {
	u, ok := links[id]
	if !ok {
		return errors.New("bilinmeyen baglanti")
	}
	wruntime.BrowserOpenURL(a.ctx, u)
	return nil
}

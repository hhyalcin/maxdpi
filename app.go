package main

import (
	"context"
	"errors"
	"sync"
	"time"

	"discord-unblock/engine"
)

// App: arayuzun cagirdigi Baglan/Kes mantigi.
type App struct {
	ctx context.Context

	mu       sync.Mutex
	eng      engine.Engine
	front    *engine.Frontend
	resolver *engine.Resolver
	profile  string
}

func NewApp() *App {
	return &App{eng: newEngine(), resolver: engine.NewResolver()}
}

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

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
	return Status{Connected: a.front != nil, Profile: a.profile}
}

// Connect: profilleri sirayla dener, ilk calisanla onyuzu baslatir.
func (a *App) Connect() (Status, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.front != nil {
		return Status{Connected: true, Profile: a.profile}, nil
	}

	// DoH profilden bagimsizdir: erisilemiyorsa strateji denemenin anlami yok (sonuc onbellege girer, Probe yeniden kullanir).
	dctx, dcancel := context.WithTimeout(a.ctx, 8*time.Second)
	_, derr := a.resolver.Lookup(dctx, "discord.com")
	dcancel()
	if derr != nil {
		return Status{}, errors.New("DNS-over-HTTPS erisilemiyor: internet baglantisini kontrol edin")
	}

	for _, p := range engine.DefaultProfiles {
		addr, err := a.eng.Start(p)
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
		err = engine.Probe(ctx, addr, a.resolver)
		cancel()
		if err != nil {
			_ = a.eng.Stop()
			continue
		}
		f, err := engine.StartFrontend(addr, a.resolver)
		if err != nil {
			_ = a.eng.Stop()
			return Status{}, errors.New("yerel proxy baslatilamadi")
		}
		a.front, a.profile = f, p.Name
		return Status{Connected: true, Profile: p.Name}, nil
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
	a.profile = ""
	return Status{}
}

// OpenDiscord: Discord'u onyuz proxy'si ile baslatir (bagli olmalidir).
func (a *App) OpenDiscord() error {
	a.mu.Lock()
	f := a.front
	a.mu.Unlock()
	if f == nil {
		return errors.New("once baglanin")
	}
	return engine.LaunchDiscord(f.Addr())
}

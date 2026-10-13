//go:build darwin

package engine

import (
	_ "embed"
	"os/exec"
)

// ciadpi: scripts/build-engines.sh ile sabit commit'ten kaynaktan derlenir.
//
//go:embed bin_darwin/ciadpi
var ciadpiBin []byte

const ciadpiName = "ciadpi"

// DefaultProfiles: macOS'ta gercek Discord engeline karsi dogrulanan stratejiler (siralama = deneme sirasi).
var DefaultProfiles = []Profile{
	{Name: "disorder+tlsrec", Args: []string{"--disorder", "1", "--tlsrec", "1+s"}},
	{Name: "oob", Args: []string{"--oob", "1"}},
	{Name: "split+tlsrec", Args: []string{"--split", "2", "--tlsrec", "3+s"}},
}

func prepareCmd(cmd *exec.Cmd) { cmd.Env = []string{} } // temiz ortam

func attach(*exec.Cmd) error { return nil }

// killStale: onceki oturumdan kalan yetim motoru (yalnizca bu yoldan calisan) temizler.
func killStale(path string) {
	// #nosec G204 -- sabit komut, yol bizim cache dizinimiz
	_ = exec.Command("/usr/bin/pkill", "-9", "-f", path).Run()
}

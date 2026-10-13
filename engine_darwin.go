//go:build darwin

package main

import "discord-unblock/engine"

func newEngine() engine.Engine { return &engine.ByeDPI{} }

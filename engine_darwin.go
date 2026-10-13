//go:build darwin

package main

import "maxdpi/engine"

func newEngine() engine.Engine { return &engine.ByeDPI{} }

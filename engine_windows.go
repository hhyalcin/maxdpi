//go:build windows

package main

import "maxdpi/engine"

func newEngine() engine.Engine { return &engine.ByeDPI{} }

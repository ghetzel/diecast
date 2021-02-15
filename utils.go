package diecast

import (
	"sync"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/gobwas/glob"
)

var globcache sync.Map

// Return whether the give path matches the given extended glob pattern as described
// at https://pkg.go.dev/github.com/gobwas/glob#Compile
//
func IsGlobMatch(path string, pattern string) bool {
	var globber glob.Glob

	if v, ok := globcache.Load(pattern); ok && v != nil {
		globber = v.(glob.Glob)
	} else if g, err := glob.Compile(pattern); err == nil {
		globber = g
		globcache.Store(pattern, g)
	} else {
		log.Warningf("bad glob %q: %v", pattern, err)
		return false
	}

	return globber.Match(path)
}

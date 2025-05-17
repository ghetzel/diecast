package diecast

import (
	"fmt"
	"time"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/typeutil"
)

// 100ms precision seems generous for this use case, but you *can* change it.
var SharedBindingPollInterval time.Duration = 100 * time.Millisecond

type SharedBindingSet []*Binding

func (set SharedBindingSet) init(server *Server) error {
	go func(s *Server) {
		if SharedBindingPollInterval > 0 {
			for at := range time.NewTicker(SharedBindingPollInterval).C {
				if at.IsZero() {
					break
				}

				for i, binding := range set {
					if ri := typeutil.Duration(binding.Interval); ri > 0 {
						if binding.lastRefreshedAt.IsZero() || time.Since(binding.lastRefreshedAt) >= ri {
							if binding.Name == `` {
								binding.Name = fmt.Sprintf("shared.%d", i)
							}

							go set.refreshAndStore(server, binding)
						}
					}
				}
			}
		}
	}(server)

	return nil
}

func (set SharedBindingSet) refreshAndStore(server *Server, binding *Binding) {
	if binding.syncing {
		return
	} else {
		binding.syncing = true
	}

	defer func() {
		binding.syncing = false
	}()

	binding.server = server

	if data, err := binding.asyncEval(); err == nil {
		if binding.Name != `` {
			server.sharedBindingData.Store(binding.Name, data)
			binding.lastRefreshedAt = time.Now()
		}
	} else {
		log.Warningf("async binding %s: %v", binding.Name, err)
	}
}

func (set SharedBindingSet) perRequestBindings() (bindings []Binding) {
	for _, b := range set {
		if b.Interval == `` {
			bindings = append(bindings, *b)
		}
	}

	return
}

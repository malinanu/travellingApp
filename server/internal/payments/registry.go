package payments

// Registry holds the gateways Spotto can take payments through. The first
// one registered is the default.
type Registry struct {
	order []Gateway
	byKey map[string]Gateway
}

func NewRegistry(gateways ...Gateway) *Registry {
	r := &Registry{byKey: map[string]Gateway{}}
	for _, g := range gateways {
		if g == nil {
			continue
		}
		if _, dup := r.byKey[g.Name()]; dup {
			continue
		}
		r.byKey[g.Name()] = g
		r.order = append(r.order, g)
	}
	return r
}

// Get returns the gateway with this name.
func (r *Registry) Get(name string) (Gateway, bool) {
	if r == nil {
		return nil, false
	}
	g, ok := r.byKey[name]
	return g, ok
}

// Enabled lists the gateways in registration order.
func (r *Registry) Enabled() []Gateway {
	if r == nil {
		return nil
	}
	return append([]Gateway(nil), r.order...)
}

// Default is the first gateway, or nil when there are none.
func (r *Registry) Default() Gateway {
	if r == nil || len(r.order) == 0 {
		return nil
	}
	return r.order[0]
}

// Len is how many gateways there are.
func (r *Registry) Len() int {
	if r == nil {
		return 0
	}
	return len(r.order)
}

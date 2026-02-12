package git

import "context"

// FallbackCloner tries primary Cloner first, falls back to secondary on error.
type FallbackCloner struct {
	primary   Cloner
	secondary Cloner
}

// NewFallbackCloner returns a FallbackCloner.
func NewFallbackCloner(primary, secondary Cloner) *FallbackCloner {
	return &FallbackCloner{primary: primary, secondary: secondary}
}

func (f *FallbackCloner) Clone(ctx context.Context, opts CloneOptions) error {
	err := f.primary.Clone(ctx, opts)
	if err == nil {
		return nil
	}
	return f.secondary.Clone(ctx, opts)
}

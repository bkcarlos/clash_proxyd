//go:build !bundle

package assets

// MihomoBinary and CountryMMDB are nil when built without -tags bundle.
var MihomoBinary []byte
var CountryMMDB []byte

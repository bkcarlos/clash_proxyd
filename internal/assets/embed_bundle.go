//go:build bundle

package assets

import _ "embed"

// MihomoBinary holds the bundled mihomo executable.
//
//go:embed mihomo
var MihomoBinary []byte

// CountryMMDB holds the bundled GeoIP database.
//
//go:embed Country.mmdb
var CountryMMDB []byte

package handler

import (
	"testing"
)

func TestSanitizeKey(t *testing.T) {
	k := sanitizeKey("/someCrazyUrl$%Z/or! even crazier ")
	if k != "_somecrazyurl__z_or__even_crazier_" {
		t.Fatal(k)
	}
}

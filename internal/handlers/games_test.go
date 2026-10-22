package handlers

import "testing"

// Каждое старое название игры обязано вести на существующую игру:
// иначе закэшированная страница игры перестанет получать награду.
func TestLegacyGameTypesPointToKnownGames(t *testing.T) {
	for legacy, canon := range legacyGameTypes {
		if !knownGames[canon] {
			t.Errorf("%q → %q: такой игры нет в knownGames", legacy, canon)
		}
		if knownGames[legacy] {
			t.Errorf("%q одновременно и старое, и действующее название", legacy)
		}
	}
}

// Выдуманный тип игры не должен проходить проверку.
func TestUnknownGameTypeRejected(t *testing.T) {
	for _, gt := range []string{"", "fake_1", "MINEFIELD", "minefield "} {
		if _, legacy := legacyGameTypes[gt]; legacy || knownGames[gt] {
			t.Errorf("%q не должен приниматься", gt)
		}
	}
}

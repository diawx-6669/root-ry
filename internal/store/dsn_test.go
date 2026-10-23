package store

import "testing"

func TestWithBinaryParameters(t *testing.T) {
	cases := []struct{ in, want string }{
		{"postgres://u:p@h:6543/postgres",
			"postgres://u:p@h:6543/postgres?binary_parameters=yes"},
		{"postgresql://u:p@h:6543/postgres?sslmode=require",
			"postgresql://u:p@h:6543/postgres?binary_parameters=yes&sslmode=require"},
		{"host=h dbname=d", "host=h dbname=d binary_parameters=yes"},
		// Уже задано — не трогаем, даже если выключено явно.
		{"postgres://h/d?binary_parameters=no", "postgres://h/d?binary_parameters=no"},
	}
	for _, c := range cases {
		if got := withBinaryParameters(c.in); got != c.want {
			t.Errorf("withBinaryParameters(%q) = %q, хотели %q", c.in, got, c.want)
		}
	}
}

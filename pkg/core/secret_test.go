package core

import (
	"testing"
)

func TestRedactURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"postgres_url",
			"postgres://user:secret@localhost:5432/db",
			"postgres://user:****@localhost:5432/db",
		},
		{
			"mongodb_url",
			"mongodb://admin:password123@host:27017/db",
			"mongodb://admin:****@host:27017/db",
		},
		{
			"empty",
			"",
			"",
		},
		{
			"no_password",
			"postgres://user@localhost:5432/db",
			"postgres://user@localhost:5432/db",
		},
		{
			"special_chars",
			"postgres://user:p%40ssword@host/db",
			"postgres://user:****@host/db",
		},
		{
			"mysql_url",
			"mysql://root:rootpass@tcp(127.0.0.1:3306)/mydb",
			"mysql://root:****@tcp(127.0.0.1:3306)/mydb",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := RedactURL(tc.input); got != tc.want {
				t.Errorf("RedactURL(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestToSecretStrings(t *testing.T) {
	t.Parallel()

	ss := ToSecretStrings([]string{"a", "b", "c"})
	if len(ss) != 3 {
		t.Errorf("length = %d, want 3", len(ss))
	}
	if string(ss[0]) != "a" {
		t.Errorf("ss[0] = %q", ss[0])
	}
}

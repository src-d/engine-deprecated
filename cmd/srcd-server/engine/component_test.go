package engine

import "testing"

func TestJoin(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		expected string
	}{
		{"linux", []string{"/foo/bar", "/baz", "qux/"}, "/foo/bar/baz/qux"},
		{"windows", []string{"C:\\\\Foo\\Bar", "\\Baz", "Qux\\"}, "C:\\\\Foo\\Bar\\Baz\\Qux"},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			result := join(tt.args...)
			if result != tt.expected {
				t.Errorf("expected: %s, got: %s", tt.expected, result)
			}
		})
	}
}

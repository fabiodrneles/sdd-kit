package app

import "testing"

func TestGreeting(t *testing.T) {
	for name, want := range map[string]string{"": "Olá, mundo!", "Ana": "Olá, Ana!"} {
		if got := Greeting(name); got != want {
			t.Errorf("Greeting(%q) = %q, want %q", name, got, want)
		}
	}
}

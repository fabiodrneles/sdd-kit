// Package app é o ponto de partida criado pelo sdd-kit (adopt --skeleton).
package app

// Greeting returns the greeting for name.
func Greeting(name string) string {
	if name == "" {
		name = "mundo"
	}
	return "Olá, " + name + "!"
}

package slug

import "strings"

var accents = strings.NewReplacer("á", "a", "à", "a", "ã", "a", "â", "a", "ç", "c", "é", "e", "ê", "e", "í", "i", "ó", "o", "õ", "o", "ô", "o", "ú", "u")

// Make returns the slug of title.
func Make(title string) string {
	s := accents.Replace(strings.ToLower(title))
	var b strings.Builder
	dash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

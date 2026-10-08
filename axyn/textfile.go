package main

import (
	"os"
	"strings"
)

// The files people open by hand (logs, reports, history) start with the UTF-8 mark (BOM):
// Windows PowerShell 5.1's Get-Content and old editors read a file without it in the
// legacy code page, and the accents and arrows came out broken ("criaÃ§Ã£o", #356).

const utf8BOM = "\ufeff"

// writeText writes a text file for people, with the UTF-8 mark.
func writeText(path, text string) error {
	return os.WriteFile(path, []byte(utf8BOM+strings.TrimPrefix(text, utf8BOM)), 0o644)
}

// openLog opens a log for appending; a new one starts with the UTF-8 mark.
func openLog(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	if st, err := f.Stat(); err == nil && st.Size() == 0 {
		_, _ = f.WriteString(utf8BOM)
	}
	return f, nil
}

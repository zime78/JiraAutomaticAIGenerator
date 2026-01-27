package ui

import (
	"image/color"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// KoreanTheme is a custom theme that supports Korean fonts
type KoreanTheme struct {
	fyne.Theme
	regularFont fyne.Resource
	boldFont    fyne.Resource
}

// NewKoreanTheme creates a theme with Korean font support
func NewKoreanTheme() fyne.Theme {
	kt := &KoreanTheme{
		Theme: theme.DarkTheme(),
	}

	// macOS TTF fonts that support Korean (not TTC)
	fontPaths := []string{
		"/System/Library/Fonts/Supplemental/Arial Unicode.ttf",
		"/System/Library/Fonts/Supplemental/AppleGothic.ttf",
		"/Library/Fonts/Arial Unicode.ttf",
	}

	for _, fontPath := range fontPaths {
		if data, err := os.ReadFile(fontPath); err == nil {
			kt.regularFont = fyne.NewStaticResource("korean-font", data)
			kt.boldFont = kt.regularFont
			break
		}
	}

	return kt
}

func (k *KoreanTheme) Font(style fyne.TextStyle) fyne.Resource {
	if k.regularFont != nil {
		return k.regularFont
	}
	return k.Theme.Font(style)
}

func (k *KoreanTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return k.Theme.Color(name, variant)
}

func (k *KoreanTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return k.Theme.Icon(name)
}

func (k *KoreanTheme) Size(name fyne.ThemeSizeName) float32 {
	return k.Theme.Size(name)
}

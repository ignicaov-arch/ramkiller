package ui

// Tokens de cor ARGB para o tema Dark Pro do RamKiller.
const (
	ColorBgBase      uint32 = 0xFF0D1117
	ColorBgCard      uint32 = 0xFF161B22
	ColorBgElevated  uint32 = 0xFF1F2630
	ColorBorder      uint32 = 0xFF262C36
	ColorTextPrimary uint32 = 0xFFF0F6FC
	ColorTextMuted   uint32 = 0xFF8B949E
	ColorGreen       uint32 = 0xFF10B981
	ColorAmber       uint32 = 0xFFF59E0B
	ColorRed         uint32 = 0xFFEF4444
	ColorBlue        uint32 = 0xFF2563EB

	ColorGreenSoft uint32 = 0x2610B981
	ColorAmberSoft uint32 = 0x26F59E0B
	ColorRedSoft   uint32 = 0x26EF4444
	ColorBlueSoft  uint32 = 0x262563EB
	ColorWhiteSoft uint32 = 0x14F0F6FC
)

// Escala de espacamento (grid 8pt) e raios concentricos padrao.
const (
	GridUnit      = 8
	RadiusCard    = 12
	RadiusButton  = 8
	RadiusControl = 6
	FontUI        = "Segoe UI"
)

// RGBToARGB converte um valor RGB (0x00RRGGBB) ou COLORREF (0x00BBGGRR se bgr=false)
// aplicando o canal alfa especificado para uso em GDI+.
func RGBToARGB(rgb uint32, alpha uint8) uint32 {
	return (uint32(alpha) << 24) | (rgb & 0x00FFFFFF)
}

// COLORREFToARGB converte COLORREF Win32 (0x00BBGGRR) para ARGB GDI+ (0xAARRGGBB).
func COLORREFToARGB(cref uint32, alpha uint8) uint32 {
	r := cref & 0xFF
	g := (cref >> 8) & 0xFF
	b := (cref >> 16) & 0xFF
	return (uint32(alpha) << 24) | (r << 16) | (g << 8) | b
}

// ARGBToCOLORREF converte ARGB GDI+ (0xAARRGGBB) para COLORREF Win32 (0x00BBGGRR).
func ARGBToCOLORREF(argb uint32) uint32 {
	r := (argb >> 16) & 0xFF
	g := (argb >> 8) & 0xFF
	b := argb & 0xFF
	return (b << 16) | (g << 8) | r
}

// ConcentricRadius calcula o raio externo R_ext = R_int + P para cantos concentricos.
func ConcentricRadius(inner, pad int) int {
	if inner <= 0 {
		return 0
	}
	if pad < 0 {
		pad = 0
	}
	return inner + pad
}

// ScaleDPI aplica escala de DPI (96 = 100%) mantendo minimo de 1 px quando v > 0.
func ScaleDPI(v int, dpi uint32) int {
	if v == 0 || dpi == 0 || dpi == 96 {
		return v
	}
	scaled := (v*int(dpi) + 48) / 96
	if v > 0 && scaled < 1 {
		return 1
	}
	return scaled
}

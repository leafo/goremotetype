package main

import _ "embed"

var (
	//go:embed icon_blue.png
	trayIconBlue []byte

	//go:embed icon_green.png
	trayIconGreen []byte

	//go:embed icon_gray.png
	trayIconGray []byte

	//go:embed icon_red.png
	trayIconRed []byte
)

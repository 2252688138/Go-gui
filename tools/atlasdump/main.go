//go:build windows

package main

import (
	"fmt"
	"strings"

	"gogui/internal/text"
)

func dumpGlyph(f *text.Font, r rune) {
	g := f.GlyphFor(r)
	fmt.Printf("\n=== %q valid=%v w=%.0f h=%.0f left=%.0f top=%.0f adv=%.0f uv=(%.5f,%.5f)-(%.5f,%.5f)\n",
		r, g.Valid, g.W, g.H, g.Left, g.Top, g.Advance, g.U0, g.V0, g.U1, g.V1)
	if !g.Valid {
		return
	}
	dim := float32(f.AtlasDimNow())
	x0 := int(g.U0*dim + 0.5)
	x1 := int(g.U1*dim + 0.5)
	// V1 对应图集行索引（已垂直翻转写入）
	topRow := int(g.V1*dim + 0.5)
	botRow := int(g.V0*dim + 0.5)
	if topRow > botRow {
		topRow, botRow = botRow, topRow
	}

	chars := " .:-=+*#%@"
	var sb strings.Builder
	for row := topRow; row <= botRow && row-topRow < 20; row++ {
		if row < 0 || row >= int(dim) {
			continue
		}
		for x := x0; x < x1 && x-x0 < 40; x++ {
			v := int(f.Atlas[row*int(dim)+x])
			sb.WriteByte(chars[v*(len(chars)-1)/255])
		}
		sb.WriteByte('\n')
	}
	fmt.Printf("rows %d..%d\n%s", topRow, botRow, sb.String())
}

func main() {
	f := text.NewFont("Microsoft YaHei UI", 28, 700, 1.0)
	fmt.Printf("Ascent=%.1f LineHeight=%.1f atlas=%d\n", f.Ascent, f.LineHeight, f.AtlasDimNow())
	for _, r := range "A仪B" {
		dumpGlyph(f, r)
	}
}

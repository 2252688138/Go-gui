//go:build windows

package main

import (
	"fmt"

	"gogui/internal/text"
)

func main() {
	f := text.NewFont("Microsoft YaHei UI", 14, 400, 1.0)
	fmt.Printf("Ascent=%.2f Descent=%.2f LineHeight=%.2f scale=%.2f\n", f.Ascent, f.Descent, f.LineHeight, f.Scale())

	for _, s := range []string{"G", "o", "仪", "表", "盘", "GoGUI", "仪表盘"} {
		rs := []rune(s)
		fmt.Printf("\n%q measure=%.2f runes=%d\n", s, f.Measure(s), len(rs))
		for _, r := range rs {
			g := f.GlyphFor(r)
			fmt.Printf("  %q adv=%.2f w=%.1f h=%.1f left=%.1f top=%.1f valid=%v uv=(%.5f,%.5f)-(%.5f,%.5f)\n",
				r, g.Advance, g.W, g.H, g.Left, g.Top, g.Valid, g.U0, g.V0, g.U1, g.V1)
		}
	}

	f2 := text.NewFont("Microsoft YaHei UI", 26, 700, 1.0)
	fmt.Printf("\n[26px bold] Ascent=%.2f LineHeight=%.2f measure(控制台)=%.2f\n", f2.Ascent, f2.LineHeight, f2.Measure("控制台"))
}

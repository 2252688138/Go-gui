// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

package ui

import "strconv"

// 图文卡片 / 社区帖子 / 列表行：内容型列表页的三种典型积木。
// 交互状态由调用方持有，回调里改状态再 RequestRedraw（与其它组件一致）。

// ---- 图文卡片（封面 + 角标 + 评分 + 标题 / 标签 / 操作按钮）----

// MediaItem 一张图文卡片的数据。
type MediaItem struct {
	Key      string
	Cover    string // 封面图路径；为空时用渐变占位
	Title    string
	Sub      string // 副标题（作者 / 来源）
	Badge    string // 左上角角标（视频 / 电影…）
	BadgeBg  string
	Score    string // 右上角评分（如 "8.5"）
	Tags     []string
	Action   string // 底部按钮文字（为空不显示）
	Accent   string // 主题色（渐变占位 / 标签底色）
	ActionBg string // 按钮底色（默认主色渐变）
}

// MediaCardOptions 图文卡片参数。
type MediaCardOptions struct {
	Width    float32 // 卡片宽度，默认 190
	CoverH   float32 // 封面高度，默认 118
	OnPick   func(it MediaItem)
	OnAction func(it MediaItem)
}

// MediaCard 图文卡片：封面（含角标 / 评分）→ 标题 → 标签 → 操作按钮。
func MediaCard(it MediaItem, opts MediaCardOptions) *Node {
	w := opts.Width
	if w <= 0 {
		w = 190
	}
	coverH := opts.CoverH
	if coverH <= 0 {
		coverH = 118
	}
	accent := it.Accent
	if accent == "" {
		accent = ColPrimary
	}
	key := "mediacard:" + it.Key

	var cover *Node
	if it.Cover != "" {
		cover = ImageSized(it.Cover, w-16, coverH).St(St().R(12))
	} else {
		cover = Box().St(St().Wp(100).Hp(100).BgGrad(accent, paintWithAlpha(accent, 0.30), 130))
	}
	coverLayers := []*Node{
		cover,
		// 底部压暗，保证叠字可读
		Box().St(St().AbsFill().BgGrad("rgba(0,0,0,0)", "rgba(0,0,0,0.62)", 180).R(12)),
	}
	if it.Badge != "" {
		bg := it.BadgeBg
		if bg == "" {
			bg = "rgba(99,102,241,0.92)"
		}
		coverLayers = append(coverLayers, Text(it.Badge).St(St().AbsTL(8, 8).
			Text(10.5, 600, "#ffffff").PadXY(7, 3).R(6).BgC(bg)))
	}
	if it.Score != "" {
		coverLayers = append(coverLayers, Row(
			Text("★").St(St().Text(10, 600, "#fbbf24")),
			Text(it.Score).St(St().Text(10.5, 600, "#ffffff")),
		).St(St().AbsTR(8, 8).GapOf(3).PadXY(6, 3).R(6).BgC("rgba(0,0,0,0.55)")).
			AlignTo(AlignCenter))
	}
	coverBox := Stack(coverLayers...).St(St().Wp(100).H(coverH).R(12).Clipped())

	cells := []*Node{coverBox}
	if it.Title != "" {
		cells = append(cells, Text(it.Title).St(St().Text(12.5, 600, ColText).WrapOn().MaxHeight(34)))
	}
	if it.Sub != "" {
		cells = append(cells, Text(it.Sub).St(St().Text(11, 400, ColTextMuted)))
	}
	if len(it.Tags) > 0 {
		tags := make([]*Node, 0, len(it.Tags))
		for _, t := range it.Tags {
			tags = append(tags, Text(t).St(St().Text(10, 500, ColTextDim).PadXY(6, 2).R(5).
				BgC(paintWithAlpha(accent, 0.16))))
		}
		cells = append(cells, Wrap(tags...).St(St().GapOf(5)))
	}
	if it.Action != "" {
		st := St().Text(12, 600, "#ffffff").PadXY(10, 7).R(8).TAlign(AlignCenter)
		if it.ActionBg != "" {
			st.BgC(it.ActionBg)
		} else {
			st.BgGrad(ColPrimary, ColPrimary2, 90)
		}
		btn := Text(it.Action).St(st).K(key + ":action")
		if opts.OnAction != nil {
			act := it
			btn.Tap(func() { opts.OnAction(act); RequestRedraw() })
		}
		cells = append(cells, btn)
	}

	card := Column(cells...).St(St().W(w).Padding(8).GapOf(7).R(14).
		BgC(ColBgSoft).Stroke(1, ColStroke))
	if IsHovered(key) {
		card.St(St().Stroke(1, paintWithAlpha(accent, 0.55)))
	}
	card.K(key)
	if opts.OnPick != nil {
		pick := it
		card.Tap(func() { opts.OnPick(pick); RequestRedraw() })
	}
	return card
}

// MediaCards 批量构建图文卡片（Wrap 流式排列，放不下自动换行）。
func MediaCards(items []MediaItem, opts MediaCardOptions, gap float32) *Node {
	cards := make([]*Node, 0, len(items))
	for _, it := range items {
		cards = append(cards, MediaCard(it, opts))
	}
	if gap <= 0 {
		gap = 12
	}
	return Wrap(cards...).St(St().GapOf(gap))
}

// ---- 列表行（图标 / 头像 + 标题 + 副标题 + 右侧信息）----

// ListRowItem 列表行数据。
type ListRowItem struct {
	Key      string
	Title    string
	Sub      string
	Meta     string // 右侧说明（时间 / 大小）
	Tag      string
	TagColor string
	Icon     string // 头像文字（与 Color 配合）
	Glyph    string // 图标字形（优先于 Icon；如 ✓ ★ ⚑）
	Color    string // 图标 / 头部色块颜色
	Checked  bool   // 多选态（选项框显示）
}

// ListRowsOptions 列表行参数。
type ListRowsOptions struct {
	Key      string
	Selected string
	Dividers bool // 行间分隔线
	CheckBox bool // 显示多选框
	Compact  bool // 紧凑行（高 36）
	OnPick   func(it ListRowItem, i int)
	OnCheck  func(it ListRowItem, i int, on bool)
}

// ListRows 列表组件：左侧图标 / 头像 + 标题与副标题 + 右侧标签 / 说明。
func ListRows(items []ListRowItem, opts ListRowsOptions) *Node {
	rowH := float32(50)
	if opts.Compact {
		rowH = 36
	}
	rows := make([]*Node, 0, len(items)*2)
	for i, it := range items {
		idx := i
		item := it
		key := "listrow:" + opts.Key + ":" + it.Key
		cells := make([]*Node, 0, 6)
		if opts.CheckBox {
			cells = append(cells, Checkbox(key+":cb", "", it.Checked, func(v bool) {
				if opts.OnCheck != nil {
					opts.OnCheck(item, idx, v)
				}
				RequestRedraw()
			}))
		}
		if it.Glyph != "" {
			cells = append(cells, Text(it.Glyph).St(St().W(30).H(30).R(9).
				Text(13, 600, orColor(it.Color)).TAlign(AlignCenter).CenterYOn().
				BgC(paintWithAlpha(orColor(it.Color), 0.16))))
		} else {
			cells = append(cells, Avatar(orAvatar(it.Icon), orColor(it.Color)))
		}
		title := Column(
			Text(it.Title).St(St().Text(12.5, 500, ColText)),
			Text(it.Sub).St(St().Text(11, 400, ColTextMuted).PadOnly(3, 0, 0, 0)),
		).St(St().GapOf(2)).Grow(1)
		cells = append(cells, title)
		if it.Tag != "" {
			cells = append(cells, Tag(it.Tag, orColor(it.TagColor)))
		}
		if it.Meta != "" {
			cells = append(cells, Text(it.Meta).St(St().Text(11, 400, ColTextMuted)))
		}
		row := Row(cells...).St(St().H(rowH).PadXY(12, 0).GapOf(12)).AlignTo(AlignCenter)
		switch {
		case opts.Selected != "" && it.Key == opts.Selected:
			row.St(St().BgC(paintWithAlpha(ColPrimary, 0.14)))
		case IsHovered(key):
			row.St(St().BgC(ColHover))
		}
		row.K(key)
		if opts.OnPick != nil {
			row.Tap(func() { opts.OnPick(item, idx); RequestRedraw() })
		}
		rows = append(rows, row)
		if opts.Dividers && i < len(items)-1 {
			rows = append(rows, Divider())
		}
	}
	return Column(rows...).St(St().R(12).Stroke(1, ColStroke).Clipped())
}

// ---- 社区帖子 ----

// PostItem 一条社区帖子。
type PostItem struct {
	Key      string
	Author   string
	Avatar   string // 头像文字（空则取昵称首字）
	AvatarBg string
	Time     string
	Role     string // 身份标签（如「楼主」「开发者」）
	Body     string
	Images   []string // 配图路径
	Tags     []string
	Likes    int
	Comments int
	Shares   int
	Liked    bool
}

// PostCardOptions 帖子卡片参数。
type PostCardOptions struct {
	Width  float32 // 默认 0 = 填满父容器
	OnLike func(it PostItem)
	OnOpen func(it PostItem)
}

// PostCard 社区帖子卡片：头像 / 昵称 / 时间 → 正文 → 图片 → 点赞评论转发。
func PostCard(it PostItem, opts PostCardOptions) *Node {
	av := it.Avatar
	if av == "" && it.Author != "" {
		av = string([]rune(it.Author)[0])
	}
	head := Row(
		Avatar(av, orColor(it.AvatarBg)),
		Column(
			Row(
				Text(it.Author).St(St().Text(13, 600, ColText)),
				Text(it.Role).St(St().Text(10, 500, ColPrimary).PadXY(6, 2).R(5).
					BgC(paintWithAlpha(ColPrimary, 0.16))),
			).St(St().GapOf(6)).AlignTo(AlignCenter),
			Text(it.Time).St(St().Text(11, 400, ColTextMuted).PadOnly(3, 0, 0, 0)),
		).St(St().GapOf(2)).Grow(1),
		Text("•••").St(St().Text(9, 600, ColPlaceholder)),
	).St(St().GapOf(10)).AlignTo(AlignStart)

	cells := []*Node{head}
	if it.Body != "" {
		cells = append(cells, Text(it.Body).St(St().Text(12.5, 400, ColTextDim).WrapOn().Lh(21)))
	}
	if len(it.Tags) > 0 {
		tags := make([]*Node, 0, len(it.Tags))
		for _, t := range it.Tags {
			tags = append(tags, Text("#"+t).St(St().Text(11, 500, ColInfo)))
		}
		cells = append(cells, Wrap(tags...).St(St().GapOf(8)))
	}
	if len(it.Images) > 0 {
		pics := make([]*Node, 0, len(it.Images))
		for _, p := range it.Images {
			pics = append(pics, ImageSized(p, 118, 88).St(St().R(10)))
		}
		cells = append(cells, Row(pics...).St(St().GapOf(8)))
	}

	likeColor := "#f472b6"
	if !it.Liked {
		likeColor = ColTextMuted
	}
	action := func(glyph, label, color string, fn func()) *Node {
		k := "post:" + it.Key + ":" + label
		n := Row(
			Text(glyph).St(St().Text(12, 500, color)),
			Text(label).St(St().Text(11.5, 500, color)),
		).St(St().GapOf(5).PadXY(8, 5).R(7)).AlignTo(AlignCenter)
		if IsHovered(k) {
			n.St(St().BgC(ColHover))
		}
		n.K(k)
		if fn != nil {
			n.Tap(fn)
		}
		return n
	}
	cells = append(cells, Row(
		action("♥", strconv.Itoa(it.Likes), likeColor, func() {
			if opts.OnLike != nil {
				opts.OnLike(it)
			}
			RequestRedraw()
		}),
		action("✎", strconv.Itoa(it.Comments), ColTextMuted, nil),
		action("↗", strconv.Itoa(it.Shares), ColTextMuted, nil),
		Spacer(),
		Link("查看详情", func() {
			if opts.OnOpen != nil {
				opts.OnOpen(it)
			}
		}),
	).St(St().GapOf(4)).AlignTo(AlignCenter))

	card := Column(cells...).St(St().Padding(14).GapOf(10).R(14).
		BgC(ColBg).Stroke(1, ColStroke))
	if opts.Width > 0 {
		card.St(St().W(opts.Width))
	}
	return card.K("post:" + it.Key)
}

// ---- 小工具 ----

// orColor 取色（空值时回落为主题次要文字色）。
func orColor(c string) string {
	if c == "" {
		return ColTextMuted
	}
	return c
}

// orAvatar 头像文字（空值时用圆点占位）。
func orAvatar(s string) string {
	if s == "" {
		return "·"
	}
	return s
}

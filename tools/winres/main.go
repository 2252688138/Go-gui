// Go-gui
// Author  : Mr.Shu
// Website : www.cq6k.com
// QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6

// winres 生成 Windows 资源目标文件（rsrc_windows_*.syso），把图标与版本信息
// 编进 Go 可执行文件，零第三方依赖、零 cgo。
//
// 原理：Go 链接器识别 COFF 目标文件里名为 .rsrc 的节，把它合并成 PE 的 .rsrc
// 段并自动写入 IMAGE_DIRECTORY_ENTRY_RESOURCE 数据目录（见 cmd/link/internal/ld
// 的 addpersrc）。因此这里只需按 PE 资源目录规范拼出节内容：
//
//	第 1 层 类型目录（RT_ICON / RT_GROUP_ICON / RT_VERSION / RT_MANIFEST）
//	第 2 层 名称/ID 目录
//	第 3 层 语言目录 → IMAGE_RESOURCE_DATA_ENTRY
//	末尾    各资源数据（4 字节对齐）
//
// IMAGE_RESOURCE_DATA_ENTRY.OffsetToData 按规范是 RVA，所以这里写入「节内偏移」
// 并补一条重定位（ADDR32NB），由链接器加上节的 RVA。
//
// 用法：
//
//	go run ./tools/winres -out cmd/gallery/rsrc_windows_amd64.syso -arch amd64 \
//	    -ico ui/assets/icon.ico -version 1.2.3 -product GoGUI \
//	    -company "Go-gui" -desc "GoGUI 纯 Go 自绘 GUI 框架" \
//	    -copyright "Copyright (c) 2026 Mr.Shu" -commit 1a2b3c4 -channel release
//
// 参数全部可选：不给 -ico 就只写版本信息；不给 -version 时从 VERSION 文件读。
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf16"
)

// PE 资源类型
const (
	rtIcon      = 3
	rtGroupIcon = 14
	rtVersion   = 16
	rtManifest  = 24
)

// 资源语言：0x0409 en-US（Windows 对各语言都接受）
const resLang uint16 = 0x0409

// COFF 机器类型与重定位类型（ADDR32NB = 相对映像的 32 位地址，即 RVA）
const (
	machineAMD64 = 0x8664
	machineI386  = 0x014c
	machineARM64 = 0xaa64

	relAMD64 = 0x0003 // IMAGE_REL_AMD64_ADDR32NB
	relI386  = 0x0007 // IMAGE_REL_I386_DIR32NB
	relARM64 = 0x0004 // IMAGE_REL_ARM64_ADDR32NB

	coffCharacteristics = 0x0104     // IMAGE_FILE_LINE_NUMS_STRIPPED | IMAGE_FILE_32BIT_MACHINE
	sectionChars        = 0x40000040 // IMAGE_SCN_CNT_INITIALIZED_DATA | IMAGE_SCN_MEM_READ
)

type resource struct {
	id   uint16
	data []byte
}

type resourceType struct {
	id    uint16
	items []resource
}

func main() {
	var (
		out       = flag.String("out", "", "输出 .syso 路径（必填）")
		arch      = flag.String("arch", "amd64", "目标架构：amd64 / 386 / arm64")
		ico       = flag.String("ico", "", "图标文件 .ico（可选）")
		manifest  = flag.String("manifest", "", "清单文件 .manifest（可选）")
		verFile   = flag.String("verfile", "VERSION", "版本号文件（-version 为空时读取）")
		version   = flag.String("version", "", "版本号，如 1.2.3；空则读 -verfile")
		product   = flag.String("product", "GoGUI", "产品名")
		company   = flag.String("company", "Go-gui", "公司 / 作者")
		desc      = flag.String("desc", "GoGUI 纯 Go 自绘 GUI 框架", "文件说明")
		copyright = flag.String("copyright", "Copyright (c) 2026 Mr.Shu", "版权声明")
		internal  = flag.String("internal", "", "内部名称，默认取 -out 主文件名")
		original  = flag.String("original", "", "原始文件名，默认取 -internal + .exe")
		commit    = flag.String("commit", "none", "git 提交短哈希")
		channel   = flag.String("channel", "dev", "发布通道 dev/beta/release")
		quiet     = flag.Bool("q", false, "静默输出")
	)
	flag.Parse()

	if *out == "" {
		fail("必须指定 -out")
	}

	ver := strings.TrimSpace(*version)
	if ver == "" {
		b, err := os.ReadFile(*verFile)
		if err != nil {
			fail("读取版本文件失败：%v", err)
		}
		ver = strings.TrimSpace(string(b))
	}
	major, minor, patch, build, err := parseVersion(ver)
	if err != nil {
		fail("%v", err)
	}

	internalName := *internal
	if internalName == "" {
		internalName = strings.TrimSuffix(filepath.Base(*out), filepath.Ext(*out))
	}
	if *internal == "" {
		internalName = strings.TrimSuffix(internalName, "_windows_amd64")
		internalName = strings.TrimSuffix(internalName, "_windows_386")
		internalName = strings.TrimSuffix(internalName, "_windows_arm64")
		internalName = "rsrc-" + internalName
	}
	originalName := *original
	if originalName == "" {
		originalName = strings.TrimPrefix(internalName, "rsrc-") + ".exe"
	}

	machine, relType, err := archInfo(*arch)
	if err != nil {
		fail("%v", err)
	}

	var types []resourceType

	// ---- 图标：RT_ICON 逐张 + RT_GROUP_ICON 索引 ----
	if *ico != "" {
		imgs, err := loadICO(*ico)
		if err != nil {
			fail("解析图标失败：%v", err)
		}
		iconType := resourceType{id: rtIcon}
		for i, im := range imgs {
			iconType.items = append(iconType.items, resource{id: uint16(i + 1), data: im.payload})
		}
		types = append(types, iconType, resourceType{
			id:    rtGroupIcon,
			items: []resource{{id: 1, data: buildGroupIcon(imgs)}},
		})
	}

	// ---- 版本信息 ----
	types = append(types, resourceType{
		id: rtVersion,
		items: []resource{{id: 1, data: buildVersionInfo(versionOpts{
			Product:     *product,
			Company:     *company,
			Description: *desc,
			Copyright:   *copyright,
			Internal:    internalName,
			Original:    originalName,
			Version:     ver,
			Commit:      *commit,
			Channel:     *channel,
			Major:       major,
			Minor:       minor,
			Patch:       patch,
			Build:       build,
		})}},
	})

	// ---- 清单（可选）----
	if *manifest != "" {
		b, err := os.ReadFile(*manifest)
		if err != nil {
			fail("读取清单失败：%v", err)
		}
		types = append(types, resourceType{
			id:    rtManifest,
			items: []resource{{id: 1, data: b}},
		})
	}

	if dir := filepath.Dir(*out); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fail("创建输出目录失败：%v", err)
		}
	}
	section, relocs := buildRSRC(types)
	coff := buildCOFF(machine, relType, section, relocs)
	if err := os.WriteFile(*out, coff, 0o644); err != nil {
		fail("写入 %s 失败：%v", *out, err)
	}
	if !*quiet {
		fmt.Printf("[winres] %s  arch=%s version=%s icons=%d resources=%d size=%d\n",
			*out, *arch, ver, countIcons(types), len(types), len(coff))
	}
}

func countIcons(types []resourceType) int {
	for _, t := range types {
		if t.id == rtIcon {
			return len(t.items)
		}
	}
	return 0
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[winres] "+format+"\n", args...)
	os.Exit(1)
}

func archInfo(arch string) (machine, relType uint16, err error) {
	switch strings.ToLower(arch) {
	case "amd64", "x64":
		return machineAMD64, relAMD64, nil
	case "386", "x86":
		return machineI386, relI386, nil
	case "arm64":
		return machineARM64, relARM64, nil
	}
	return 0, 0, fmt.Errorf("不支持的架构 %q（可选 amd64 / 386 / arm64）", arch)
}

// parseVersion "1.2.3-rc1" → (1,2,3,0)；非数字后缀忽略，缺省段补 0。
func parseVersion(v string) (major, minor, patch, build uint16, err error) {
	s := strings.TrimSpace(v)
	s = strings.TrimPrefix(strings.TrimPrefix(s, "v"), "V")
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) == 0 || len(parts) > 4 {
		return 0, 0, 0, 0, fmt.Errorf("非法版本号 %q", v)
	}
	dst := []*uint16{&major, &minor, &patch, &build}
	for i, p := range parts {
		n, e := strconv.Atoi(p)
		if e != nil || n < 0 || n > 65535 {
			return 0, 0, 0, 0, fmt.Errorf("非法版本号 %q（每段须为 0~65535 的整数）", v)
		}
		*dst[i] = uint16(n)
	}
	return major, minor, patch, build, nil
}

// ---------------- 图标 ----------------

type icoImage struct {
	width   byte
	height  byte
	colors  byte
	planes  uint16
	bit     uint16
	payload []byte
}

func loadICO(path string) ([]icoImage, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) < 6 {
		return nil, fmt.Errorf("%s 不是合法的 ico 文件", path)
	}
	if t := binary.LittleEndian.Uint16(b[2:]); t != 1 {
		return nil, fmt.Errorf("%s 不是图标类型（type=%d）", path, t)
	}
	count := int(binary.LittleEndian.Uint16(b[4:]))
	if count == 0 {
		return nil, fmt.Errorf("%s 里没有图标", path)
	}
	if len(b) < 6+count*16 {
		return nil, fmt.Errorf("%s 图标目录被截断", path)
	}
	out := make([]icoImage, 0, count)
	for i := 0; i < count; i++ {
		e := 6 + i*16
		size := binary.LittleEndian.Uint32(b[e+8:])
		off := binary.LittleEndian.Uint32(b[e+12:])
		if int(off)+int(size) > len(b) {
			return nil, fmt.Errorf("%s 第 %d 张图标数据越界", path, i+1)
		}
		out = append(out, icoImage{
			width:   b[e],
			height:  b[e+1],
			colors:  b[e+2],
			planes:  binary.LittleEndian.Uint16(b[e+4:]),
			bit:     binary.LittleEndian.Uint16(b[e+6:]),
			payload: b[off : off+size],
		})
	}
	return out, nil
}

// buildGroupIcon 生成 GRPICONDIR：尺寸表 + 对应 RT_ICON 的 ID。
func buildGroupIcon(imgs []icoImage) []byte {
	var buf bytes.Buffer
	put16(&buf, 0) // reserved
	put16(&buf, 1) // type = icon
	put16(&buf, uint16(len(imgs)))
	for i, im := range imgs {
		buf.WriteByte(im.width)
		buf.WriteByte(im.height)
		buf.WriteByte(im.colors)
		buf.WriteByte(0)
		put16(&buf, im.planes)
		put16(&buf, im.bit)
		put32(&buf, uint32(len(im.payload)))
		put16(&buf, uint16(i+1)) // 与 RT_ICON 的 ID 对应
	}
	return buf.Bytes()
}

// ---------------- 版本信息 ----------------

type versionOpts struct {
	Product, Company, Description, Copyright, Internal, Original string
	Version, Commit, Channel                                     string
	Major, Minor, Patch, Build                                   uint16
}

type viNode struct {
	key      string
	valueLen uint16 // 字符串节点为字符数（含结尾 0），二进制节点为字节数
	typeVal  uint16 // 0 = 二进制, 1 = 文本
	value    []byte
	children []*viNode
}

func buildVersionInfo(o versionOpts) []byte {
	// VS_FIXEDFILEINFO（13 个 DWORD，共 52 字节）
	var fixed bytes.Buffer
	put32(&fixed, 0xFEEF04BD)                            // dwSignature
	put32(&fixed, 0x00010000)                            // dwStrucVersion
	put32(&fixed, uint32(o.Major)<<16|uint32(o.Minor))   // dwFileVersionMS
	put32(&fixed, uint32(o.Patch)<<16|uint32(o.Build))   // dwFileVersionLS
	put32(&fixed, uint32(o.Major)<<16|uint32(o.Minor))   // dwProductVersionMS
	put32(&fixed, uint32(o.Patch)<<16|uint32(o.Build))   // dwProductVersionLS
	put32(&fixed, 0x3F)                                  // dwFileFlagsMask
	put32(&fixed, fileFlags(o))                          // dwFileFlags
	put32(&fixed, 0x00040004)                            // dwFileOS = VOS_NT_WINDOWS32
	put32(&fixed, 1)                                     // dwFileType = VFT_APP
	put32(&fixed, 0)                                     // dwFileSubtype
	put32(&fixed, 0)                                     // dwFileDateMS
	put32(&fixed, 0)                                     // dwFileDateLS

	numVer := fmt.Sprintf("%d.%d.%d", o.Major, o.Minor, o.Patch)
	if o.Build != 0 {
		numVer = fmt.Sprintf("%d.%d.%d.%d", o.Major, o.Minor, o.Patch, o.Build)
	}
	buildID := o.Channel
	if o.Commit != "" && o.Commit != "none" {
		buildID += " · " + o.Commit
	}

	strs := []*viNode{
		{key: "CompanyName", typeVal: 1, value: utf16z(o.Company)},
		{key: "FileDescription", typeVal: 1, value: utf16z(o.Description)},
		{key: "FileVersion", typeVal: 1, value: utf16z(numVer)},
		{key: "InternalName", typeVal: 1, value: utf16z(o.Internal)},
		{key: "LegalCopyright", typeVal: 1, value: utf16z(o.Copyright)},
		{key: "OriginalFilename", typeVal: 1, value: utf16z(o.Original)},
		{key: "ProductName", typeVal: 1, value: utf16z(o.Product)},
		{key: "ProductVersion", typeVal: 1, value: utf16z(o.Version)},
		{key: "BuildId", typeVal: 1, value: utf16z(buildID)},
	}
	for _, s := range strs {
		s.valueLen = uint16(len(s.value) / 2)
	}

	// StringTable 的 key 是「语言页 + 代码页」：0x0804(简体中文) + 0x04B0(1200, Unicode)
	table := &viNode{key: "080404B0", typeVal: 1, children: strs}
	strInfo := &viNode{key: "StringFileInfo", typeVal: 1, children: []*viNode{table}}

	var trans bytes.Buffer
	put16(&trans, 0x0804)
	put16(&trans, 0x04B0)
	translation := &viNode{key: "Translation", typeVal: 0, value: trans.Bytes(), valueLen: 4}
	varFile := &viNode{key: "VarFileInfo", typeVal: 1, children: []*viNode{translation}}

	root := &viNode{
		key:      "VS_VERSION_INFO",
		typeVal:  0,
		value:    fixed.Bytes(),
		valueLen: uint16(fixed.Len()),
		children: []*viNode{strInfo, varFile},
	}

	var out bytes.Buffer
	writeVINode(&out, root)
	return out.Bytes()
}

// fileFlags 按通道设置 VS_FF_DEBUG / VS_FF_PRERELEASE。
func fileFlags(o versionOpts) uint32 {
	switch strings.ToLower(o.Channel) {
	case "release":
		return 0x00000000
	case "beta":
		return 0x00000002 // VS_FF_PRERELEASE
	default:
		return 0x00000001 // VS_FF_DEBUG
	}
}

// viSize 计算节点总长度（含内部 4 字节对齐填充）。
func viSize(n *viNode) int {
	s := 6 + len(utf16z(n.key)) // wLength + wValueLength + wType + szKey
	s = align4(s)
	if n.value != nil {
		s = align4(s + len(n.value))
	}
	for _, c := range n.children {
		s += viSize(c)
	}
	return s
}

func writeVINode(buf *bytes.Buffer, n *viNode) {
	start := buf.Len()
	total := viSize(n)

	var hdr bytes.Buffer
	put16(&hdr, uint16(total))
	put16(&hdr, n.valueLen)
	put16(&hdr, n.typeVal)
	hdr.Write(utf16z(n.key))
	padTo(&hdr, 4)
	buf.Write(hdr.Bytes())

	if n.value != nil {
		buf.Write(n.value)
		padTo(buf, 4)
	}
	for _, c := range n.children {
		writeVINode(buf, c)
	}
	for buf.Len()-start < total { // 兜底：长度与 wLength 一致
		buf.WriteByte(0)
	}
}

// utf16z 字符串 → UTF-16LE 且带结尾 0。
func utf16z(s string) []byte {
	u := utf16.Encode([]rune(s))
	b := make([]byte, 0, (len(u)+1)*2)
	for _, v := range u {
		b = append(b, byte(v), byte(v>>8))
	}
	return append(b, 0, 0)
}

func align4(n int) int { return (n + 3) &^ 3 }

func padTo(buf *bytes.Buffer, align int) {
	for buf.Len()%align != 0 {
		buf.WriteByte(0)
	}
}

func put16(buf *bytes.Buffer, v uint16) {
	_ = binary.Write(buf, binary.LittleEndian, v)
}

func put32(buf *bytes.Buffer, v uint32) {
	_ = binary.Write(buf, binary.LittleEndian, v)
}

// ---------------- 资源目录 ----------------

// buildRSRC 拼出 PE 资源节内容，同时返回需要重定位的偏移
// （IMAGE_RESOURCE_DATA_ENTRY.OffsetToData 字段位置）。
func buildRSRC(types []resourceType) (data []byte, relocs []uint32) {
	sortTypes(types) // 类型与条目都必须按 ID 升序

	off := 16 + 8*len(types)
	typeOff := make([]int, len(types))
	for i, t := range types {
		typeOff[i] = off
		off += 16 + 8*len(t.items)
	}
	langOff := make([][]int, len(types))
	for i, t := range types {
		langOff[i] = make([]int, len(t.items))
		for j := range t.items {
			langOff[i][j] = off
			off += 16 + 8 // 单语言目录
		}
	}
	entryOff := make([][]int, len(types))
	for i, t := range types {
		entryOff[i] = make([]int, len(t.items))
		for j := range t.items {
			entryOff[i][j] = off
			off += 16
		}
	}
	blobOff := make([][]int, len(types))
	for i, t := range types {
		blobOff[i] = make([]int, len(t.items))
		for j, it := range t.items {
			off = align4(off)
			blobOff[i][j] = off
			off += len(it.data)
		}
	}

	data = make([]byte, align4(off))
	le := binary.LittleEndian

	// 第 1 层：根目录
	le.PutUint16(data[12:], 0)
	le.PutUint16(data[14:], uint16(len(types)))
	for i, t := range types {
		e := 16 + i*8
		le.PutUint32(data[e:], uint32(t.id))
		le.PutUint32(data[e+4:], 0x80000000|uint32(typeOff[i]))
	}

	for i, t := range types {
		p := typeOff[i]
		le.PutUint16(data[p+12:], 0)
		le.PutUint16(data[p+14:], uint16(len(t.items)))
		for j, it := range t.items {
			e := p + 16 + j*8
			le.PutUint32(data[e:], uint32(it.id))
			le.PutUint32(data[e+4:], 0x80000000|uint32(langOff[i][j]))
		}
		for j := range t.items {
			lp := langOff[i][j]
			le.PutUint16(data[lp+12:], 0)
			le.PutUint16(data[lp+14:], 1)
			le.PutUint32(data[lp+16:], uint32(resLang))
			le.PutUint32(data[lp+20:], uint32(entryOff[i][j]))
		}
		for j, it := range t.items {
			ep := entryOff[i][j]
			// OffsetToData 先写节内偏移，随后由重定位改成 RVA
			le.PutUint32(data[ep:], uint32(blobOff[i][j]))
			le.PutUint32(data[ep+4:], uint32(len(it.data)))
			le.PutUint32(data[ep+8:], 0) // CodePage
			le.PutUint32(data[ep+12:], 0)
			relocs = append(relocs, uint32(ep))
			copy(data[blobOff[i][j]:], it.data)
		}
	}
	return data, relocs
}

func sortTypes(types []resourceType) {
	for i := 1; i < len(types); i++ {
		for j := i; j > 0 && types[j].id < types[j-1].id; j-- {
			types[j], types[j-1] = types[j-1], types[j]
		}
	}
	for _, t := range types {
		for i := 1; i < len(t.items); i++ {
			for j := i; j > 0 && t.items[j].id < t.items[j-1].id; j-- {
				t.items[j], t.items[j-1] = t.items[j-1], t.items[j]
			}
		}
	}
}

// ---------------- COFF 目标文件 ----------------

// buildCOFF 把资源节写成 COFF 目标文件：1 个节 + 1 个节符号 + 重定位表。
func buildCOFF(machine, relType uint16, section []byte, relocs []uint32) []byte {
	const (
		headerSize   = 20
		sectionSize  = 40
		relocSize    = 10
		symbolSize   = 18
		strTableSize = 4
	)

	if pad := align4(len(section)) - len(section); pad > 0 {
		section = append(section, make([]byte, pad)...)
	}

	dataOff := headerSize + sectionSize
	relocOff := dataOff + len(section)
	symOff := relocOff + len(relocs)*relocSize
	strOff := symOff + symbolSize

	var buf bytes.Buffer
	put16(&buf, machine)
	put16(&buf, 1)                   // NumberOfSections
	put32(&buf, 0)                   // TimeDateStamp：固定 0，产物可复现
	put32(&buf, uint32(symOff))      // PointerToSymbolTable
	put32(&buf, 1)                   // NumberOfSymbols
	put16(&buf, 0)                   // SizeOfOptionalHeader
	put16(&buf, coffCharacteristics) // Characteristics

	buf.WriteString(".rsrc\x00\x00\x00") // 节名（必须叫 .rsrc，链接器据此识别）
	put32(&buf, 0)                       // VirtualSize
	put32(&buf, 0)                       // VirtualAddress
	put32(&buf, uint32(len(section)))    // SizeOfRawData
	put32(&buf, uint32(dataOff))         // PointerToRawData
	put32(&buf, uint32(relocOff))        // PointerToRelocations
	put32(&buf, 0)                       // PointerToLinenumbers
	put16(&buf, uint16(len(relocs)))     // NumberOfRelocations
	put16(&buf, 0)                       // NumberOfLinenumbers
	put32(&buf, sectionChars)            // Characteristics

	if buf.Len() != dataOff {
		fail("内部错误：COFF 头长度 %d，期望 %d", buf.Len(), dataOff)
	}
	buf.Write(section)

	for _, r := range relocs {
		put32(&buf, r)       // VirtualAddress（节内偏移）
		put32(&buf, 0)       // SymbolTableIndex → 第 0 个符号（节符号）
		put16(&buf, relType) // ADDR32NB
	}

	if buf.Len() != symOff {
		fail("内部错误：符号表偏移 %d，期望 %d", buf.Len(), symOff)
	}
	buf.WriteString(".rsrc\x00\x00\x00") // 符号名（8 字节内联）
	put32(&buf, 0)                       // Value
	put16(&buf, 1)                       // SectionNumber
	put16(&buf, 0)                       // Type
	buf.WriteByte(3)                     // StorageClass = IMAGE_SYM_CLASS_STATIC
	buf.WriteByte(0)                     // NumberOfAuxSymbols

	if buf.Len() != strOff {
		fail("内部错误：字符串表偏移 %d，期望 %d", buf.Len(), strOff)
	}
	put32(&buf, strTableSize) // 字符串表长度（只有长度字段本身）

	return buf.Bytes()
}

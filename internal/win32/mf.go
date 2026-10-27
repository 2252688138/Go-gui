//go:build windows

package win32

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
)

// Windows Media Foundation（系统解码器）薄封装：没装 ffmpeg 时的回退路径。
// 只依赖 mfplat.dll / mfreadwrite.dll 的导出函数 + IMFSourceReader 的 COM vtable，
// 全程无 cgo（与项目其它 win32 绑定一致）。
//
// 设计要点：
//   - **一个 reader 同时输出画面 + 声音**（ReadPacket 由 actualStreamIndex 分流）。
//     同进程创建第二个 SourceReader 在部分 Windows 版本上会访问违例，必须避免。
//   - 解码循环必须跑在锁定线程上并初始化 COM（ui 层用 MFThreadInit）。
//   - 输出视频格式按 RGB32 → NV12 → YUY2 依次尝试，统一转成 RGBA 交给上层。

// Debugf 输出诊断日志（受 Debug 开关控制，供 ui 层复用）。
func Debugf(format string, args ...any) { dbg(format, args...) }

// GUID Windows GUID。
type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

func (g GUID) String() string {
	return fmt.Sprintf("%08x-%04x-%04x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		g.Data1, g.Data2, g.Data3, g.Data4[0], g.Data4[1],
		g.Data4[2], g.Data4[3], g.Data4[4], g.Data4[5], g.Data4[6], g.Data4[7])
}

// parseGUID 解析 "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"。
func parseGUID(s string) GUID {
	var g GUID
	s = strings.Trim(strings.TrimSpace(s), "{}\"")
	parts := strings.Split(s, "-")
	if len(parts) != 5 {
		return g
	}
	hex := func(v string) uint64 {
		n, err := strconv.ParseUint(v, 16, 64)
		if err != nil {
			return 0
		}
		return n
	}
	g.Data1 = uint32(hex(parts[0]))
	g.Data2 = uint16(hex(parts[1]))
	g.Data3 = uint16(hex(parts[2]))
	if len(parts[3]) == 4 {
		g.Data4[0] = byte(hex(parts[3][:2]))
		g.Data4[1] = byte(hex(parts[3][2:]))
	}
	if len(parts[4]) == 12 {
		for i := 0; i < 6; i++ {
			g.Data4[2+i] = byte(hex(parts[4][i*2 : i*2+2]))
		}
	}
	return g
}

var (
	MFMediaTypeVideo   = parseGUID("73646976-0000-0010-8000-00AA00389B71") // 'vids'
	MFMediaTypeAudio   = parseGUID("73647561-0000-0010-8000-00AA00389B71") // 'auds'
	MFVideoFormatRGB32 = parseGUID("00000016-0000-0010-8000-00AA00389B71")
	MFVideoFormatNV12  = parseGUID("3231564e-0000-0010-8000-00AA00389B71")
	MFVideoFormatYUY2  = parseGUID("32595559-0000-0010-8000-00AA00389B71")
	MFAudioFormatPCM   = parseGUID("00000001-0000-0010-8000-00AA00389B71")

	MFMTMajorType             = parseGUID("48eba18e-f8c9-4687-bf11-0a74c9f96a8f")
	MFMTSubtype               = parseGUID("f7e34c9a-42e8-4714-b74b-cb29d72c35e5")
	MFMTFrameSize             = parseGUID("1652c33d-d6b2-4012-b834-72030849a37d")
	MFMTFrameRate             = parseGUID("c459a2e8-3d2c-4e44-b132-fee5156c7bb0")
	MFMTDefaultStride         = parseGUID("644b4e48-1e02-4516-b0eb-c01ca9d49ac6")
	MFMTAudioNumChannels      = parseGUID("37e48bf5-645e-4c5b-89de-ada9e29b696a")
	MFMTAudioSamplesPerSecond = parseGUID("5faeeae7-0290-4c31-9e8a-c534f68d9dba")
	MFMTAudioBitsPerSample    = parseGUID("f2deb57f-40fa-4764-aa33-ed4f2d1ff669")
	MFPDAttributeDuration     = parseGUID("6c990d33-bb8e-477a-8598-0d5d96fcd88a")
)

const (
	mfVersion     = 0x00020070
	mfStartupFull = 0

	// 流选择用的特殊索引
	MFSourceReaderAnyStream = 0xFFFFFFFF

	MFSourceReaderFlagError       = 0x2
	MFSourceReaderFlagEndOfStream = 0x4

	vtI8 = 20

	// PacketKind 返回的包类型
	PacketNone  = 0
	PacketVideo = 1
	PacketAudio = 2
)

var (
	libMfplat      uintptr
	libMfreadwrite uintptr

	mfStartupFn                   func(version uint32, flags uint32) uint32
	mfCreateSourceReaderFromURLFn func(url *uint16, attrs uintptr, reader *uintptr) uint32
	mfCreateMediaTypeFn           func(t *uintptr) uint32
	coInitializeExFn              func(reserved uintptr, coInit uint32) uint32
	coUninitializeFn              func()

	mfLibOnce sync.Once
	mfLibErr  error
)

// mfLoad 只加载 DLL 并绑定导出（可在任意线程调用，不碰 COM）。
func mfLoad() error {
	mfLibOnce.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				mfLibErr = fmt.Errorf("加载 Media Foundation 失败: %v", r)
			}
		}()
		libMfplat = loadLib("mfplat.dll")
		libMfreadwrite = loadLib("mfreadwrite.dll")
		purego.RegisterLibFunc(&mfStartupFn, libMfplat, "MFStartup")
		purego.RegisterLibFunc(&mfCreateMediaTypeFn, libMfplat, "MFCreateMediaType")
		purego.RegisterLibFunc(&mfCreateSourceReaderFromURLFn, libMfreadwrite, "MFCreateSourceReaderFromURL")
		libOle32 = loadLib("ole32.dll")
		purego.RegisterLibFunc(&coInitializeExFn, libOle32, "CoInitializeEx")
		purego.RegisterLibFunc(&coUninitializeFn, libOle32, "CoUninitialize")
	})
	return mfLibErr
}

// mfStart 初始化 COM 单元 + MF 平台（需在锁定线程上调用）。
func mfStart() error {
	if err := mfLoad(); err != nil {
		return err
	}
	if coInitializeExFn != nil {
		// COINIT_MULTITHREADED；S_FALSE / RPC_E_CHANGED_MODE 都可继续
		coInitializeExFn(0, 0)
	}
	if mfStartupFn != nil {
		if hr := mfStartupFn(mfVersion, mfStartupFull); hr != 0 && hr != 0x8007000E {
			return fmt.Errorf("MFStartup 失败（0x%08x）", hr)
		}
	}
	return nil
}

// MFAvailable 系统解码器是否可用（仅检查 DLL 是否存在）。
func MFAvailable() bool { return mfLoad() == nil }

// MFThreadInit 在当前 OS 线程初始化 COM 并锁定线程（Media Foundation 要求）。
// 不调用 CoUninitialize：Go 线程池复用线程，反初始化后再在该线程调用会访问违例。
func MFThreadInit() func() {
	runtime.LockOSThread()
	_ = mfStart()
	return func() { runtime.UnlockOSThread() }
}

var (
	mfWorkerOnce sync.Once
	mfQueue      chan func()
)

// MFRun 在专用 MF 线程上同步执行 fn：所有 MF 调用（创建 reader / ReadSample /
// Seek / Release）都在同一个 OS 线程、同一个 COM 套间上串行执行。
// Media Foundation 的 SourceReader 对跨线程调度敏感，不集中在一个线程上会
// 在创建第二个 reader 时访问违例（0xc0000005）。
func MFRun(fn func()) {
	if fn == nil {
		return
	}
	mfWorkerOnce.Do(func() {
		mfQueue = make(chan func(), 256)
		go func() {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			_ = mfStart()
			for f := range mfQueue {
				f()
			}
		}()
	})
	done := make(chan struct{})
	mfQueue <- func() {
		defer close(done)
		fn()
	}
	<-done
}

// comCall 调用 COM 接口 vtable 指定槽位的方法，返回 HRESULT。
func comCall(obj uintptr, index int, args ...uintptr) uint32 {
	if obj == 0 {
		return 0x80004003 // E_POINTER
	}
	vtbl := *(*uintptr)(unsafe.Pointer(obj))
	fn := *(*uintptr)(unsafe.Pointer(vtbl + uintptr(index)*unsafe.Sizeof(uintptr(0))))
	all := make([]uintptr, 0, len(args)+1)
	all = append(all, obj)
	all = append(all, args...)
	r, _, _ := purego.SyscallN(fn, all...)
	return uint32(r)
}

func comRelease(obj uintptr) {
	if obj != 0 {
		comCall(obj, 2)
	}
}

func comFail(hr uint32) bool { return int32(hr) < 0 }

// ---- IMFMediaType（继承 IMFAttributes）----

type mfMediaType struct{ ptr uintptr }

func (m mfMediaType) Close() { comRelease(m.ptr) }

func (m mfMediaType) getU32(key GUID) (uint32, bool) {
	var v uint32
	if hr := comCall(m.ptr, 7, uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&v))); comFail(hr) {
		return 0, false
	}
	return v, true
}

func (m mfMediaType) getU64(key GUID) (uint64, bool) {
	var v uint64
	if hr := comCall(m.ptr, 8, uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&v))); comFail(hr) {
		return 0, false
	}
	return v, true
}

func (m mfMediaType) getGUID(key GUID) (GUID, bool) {
	var v GUID
	if hr := comCall(m.ptr, 10, uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&v))); comFail(hr) {
		return v, false
	}
	return v, true
}

func (m mfMediaType) setU32(key GUID, v uint32) {
	comCall(m.ptr, 21, uintptr(unsafe.Pointer(&key)), uintptr(v))
}

func (m mfMediaType) setU64(key GUID, v uint64) {
	comCall(m.ptr, 22, uintptr(unsafe.Pointer(&key)), uintptr(v))
}

func (m mfMediaType) setGUID(key, v GUID) {
	comCall(m.ptr, 24, uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&v)))
}

// ---- 输出参数结构（堆分配，避免栈地址失效）----

type mfSampleOut struct {
	actual uint32
	flags  uint32
	ts     int64
	sample uintptr
}

type mfBufOut struct {
	buf    uintptr
	data   uintptr
	maxLen uint32
	curLen uint32
}

type mfStreamOut struct {
	mt uintptr
}

// propVariant PROPVARIANT（只用到 VT_I8，x64 下 24 字节）。
type propVariant struct {
	vt        uint16
	reserved1 uint16
	reserved2 uint16
	reserved3 uint16
	val       int64
	_padding  int64
}

// ---- MFSourceReader ----

// MFSourceReader 系统解码器（单个 reader 同时输出画面与声音）。
type MFSourceReader struct {
	ptr uintptr

	hasVideo, hasAudio bool
	duration           time.Duration
	width, height      int
	fps                float64
	stride             int
	rate, channels, bits int
	// pixFmt 视频输出格式：0=RGB32 1=NV12 2=YUY2 3=未知
	pixFmt        int
	videoIdx      int // 视频流索引（-1 无）
	audioIdx      int // 音频流索引（-1 无）
	eof           bool
	closed        bool
	mu            sync.Mutex
}

// MFOpen 用系统解码器打开音频 / 视频文件。
func MFOpen(path string) (*MFSourceReader, error) {
	if err := mfLoad(); err != nil {
		return nil, err
	}
	var r *MFSourceReader
	var openErr error
	MFRun(func() {
		u, uerr := syscall.UTF16PtrFromString(path)
		if uerr != nil {
			openErr = uerr
			return
		}
		var reader uintptr
		hr := mfCreateSourceReaderFromURLFn(u, 0, &reader)
		if comFail(hr) || reader == 0 {
			openErr = fmt.Errorf("系统解码器无法打开该文件（0x%08x）", hr)
			return
		}
		rr := &MFSourceReader{ptr: reader, videoIdx: -1, audioIdx: -1}
		rr.probe()
		if !rr.hasVideo && !rr.hasAudio {
			comRelease(reader)
			openErr = errors.New("系统解码器未找到音频或视频流")
			return
		}
		r = rr
	})
	if openErr != nil {
		return nil, openErr
	}
	return r, nil
}

// nativeType 取第 i 条流的原生媒体类型（调用方 Release）。
func (r *MFSourceReader) nativeType(index int) (mfMediaType, bool) {
	out := new(mfStreamOut)
	if hr := comCall(r.ptr, 5, uintptr(index), 0, uintptr(unsafe.Pointer(&out.mt))); comFail(hr) || out.mt == 0 {
		runtime.KeepAlive(out)
		return mfMediaType{}, false
	}
	return mfMediaType{ptr: out.mt}, true
}

// probe 枚举流并读取尺寸 / 帧率 / 采样率 / 时长。
func (r *MFSourceReader) probe() {
	for i := 0; i < 8; i++ {
		mt, ok := r.nativeType(i)
		if !ok {
			break
		}
		major, hasMajor := mt.getGUID(MFMTMajorType)
		switch {
		case hasMajor && major == MFMediaTypeVideo && !r.hasVideo && r.videoIdx < 0:
			w, h := 0, 0
			if sz, ok := mt.getU64(MFMTFrameSize); ok {
				w, h = int(sz>>32), int(uint32(sz))
			}
			if w <= 0 || h <= 0 {
				mt.Close()
				continue
			}
			r.videoIdx = i
			r.width, r.height = w, h
			if fr, ok := mt.getU64(MFMTFrameRate); ok && fr != 0 {
				if den := uint32(fr); den > 0 {
					r.fps = float64(fr>>32) / float64(den)
				}
			}
			if r.fps <= 0 || r.fps > 1000 {
				r.fps = 0
			}
			if st, ok := mt.getU32(MFMTDefaultStride); ok && st != 0 {
				r.stride = int(int32(st))
			}
			r.hasVideo = true
		case hasMajor && major == MFMediaTypeAudio && !r.hasAudio && r.audioIdx < 0:
			r.audioIdx = i
			if v, ok := mt.getU32(MFMTAudioSamplesPerSecond); ok {
				r.rate = int(v)
			}
			if v, ok := mt.getU32(MFMTAudioNumChannels); ok {
				r.channels = int(v)
			}
			if v, ok := mt.getU32(MFMTAudioBitsPerSample); ok {
				r.bits = int(v)
			}
			r.hasAudio = r.rate > 0
		}
		mt.Close()
	}
	// 时长（100ns）
	var pv propVariant
	key := MFPDAttributeDuration
	if hr := comCall(r.ptr, 11, uintptr(MFSourceReaderAnyStream), uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&pv))); !comFail(hr) && pv.vt == vtI8 && pv.val > 0 {
		r.duration = time.Duration(pv.val * 100)
	}
}

func (r *MFSourceReader) HasVideo() bool { return r != nil && r.hasVideo }
func (r *MFSourceReader) HasAudio() bool { return r != nil && r.hasAudio }

// Size 视频尺寸。
func (r *MFSourceReader) Size() (int, int) {
	if r == nil {
		return 0, 0
	}
	return r.width, r.height
}

// FPS 帧率（未知返回 0）。
func (r *MFSourceReader) FPS() float64 {
	if r == nil {
		return 0
	}
	return r.fps
}

// Duration 总时长。
func (r *MFSourceReader) Duration() time.Duration {
	if r == nil {
		return 0
	}
	return r.duration
}

// AudioFormat 音频输出格式。
func (r *MFSourceReader) AudioFormat() (rate, ch, bits int) {
	if r == nil {
		return 0, 0, 0
	}
	return r.rate, r.channels, r.bits
}

// SelectStreams 选择输出的流（video/audio 为 true 时选中对应第一条流）。
func (r *MFSourceReader) SelectStreams(video, audio bool) {
	MFRun(func() { r.selectStreamsLocked(video, audio) })
}

func (r *MFSourceReader) selectStreamsLocked(video, audio bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	// SetStreamSelection(MF_SOURCE_READER_ALL_STREAMS, FALSE) 先全关
	comCall(r.ptr, 4, uintptr(MFSourceReaderAnyStream-1), 0)
	if video && r.videoIdx >= 0 {
		comCall(r.ptr, 4, uintptr(r.videoIdx), 1)
	}
	if audio && r.audioIdx >= 0 {
		comCall(r.ptr, 4, uintptr(r.audioIdx), 1)
	}
}

// SetVideoRGB32 让视频流输出未压缩帧：RGB32 → NV12 → YUY2 依次尝试。
func (r *MFSourceReader) SetVideoRGB32() error {
	var err error
	MFRun(func() { err = r.setVideoLocked() })
	return err
}

func (r *MFSourceReader) setVideoLocked() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || !r.hasVideo {
		return errors.New("没有视频流")
	}
	type cand struct {
		format GUID
		pix    int
		name   string
	}
	cands := []cand{
		{MFVideoFormatRGB32, 0, "RGB32"},
		{MFVideoFormatNV12, 1, "NV12"},
		{MFVideoFormatYUY2, 2, "YUY2"},
	}
	for _, c := range cands {
		if r.tryVideoOutput(c.format) {
			r.pixFmt = c.pix
			dbg("系统解码器输出格式 = %s", c.name)
			return nil
		}
	}
	return errors.New("系统解码器不支持解压视频输出（RGB32/NV12/YUY2 均失败）")
}

func (r *MFSourceReader) tryVideoOutput(sub GUID) bool {
	var mtPtr uintptr
	if hr := mfCreateMediaTypeFn(&mtPtr); comFail(hr) || mtPtr == 0 {
		return false
	}
	mt := mfMediaType{ptr: mtPtr}
	defer mt.Close()
	mt.setGUID(MFMTMajorType, MFMediaTypeVideo)
	mt.setGUID(MFMTSubtype, sub)
	if r.width > 0 && r.height > 0 {
		mt.setU64(MFMTFrameSize, uint64(uint32(r.width))<<32|uint64(uint32(r.height)))
	}
	if r.fps > 0 {
		mt.setU64(MFMTFrameRate, uint64(uint32(r.fps+0.5))<<32)
	}
	hr := comCall(r.ptr, 7, uintptr(r.videoIdx), 0, mtPtr)
	return !comFail(hr)
}

// SetAudioPCM 把音频流输出设为 16bit PCM 并回读实际格式。
func (r *MFSourceReader) SetAudioPCM() error {
	var err error
	MFRun(func() { err = r.setAudioLocked() })
	return err
}

func (r *MFSourceReader) setAudioLocked() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || !r.hasAudio {
		return errors.New("没有音频流")
	}
	var mtPtr uintptr
	if hr := mfCreateMediaTypeFn(&mtPtr); comFail(hr) || mtPtr == 0 {
		return fmt.Errorf("MFCreateMediaType 失败（0x%08x）", hr)
	}
	mt := mfMediaType{ptr: mtPtr}
	mt.setGUID(MFMTMajorType, MFMediaTypeAudio)
	mt.setGUID(MFMTSubtype, MFAudioFormatPCM)
	hr := comCall(r.ptr, 7, uintptr(r.audioIdx), 0, mtPtr)
	mt.Close()
	if comFail(hr) {
		return fmt.Errorf("设置音频输出格式失败（0x%08x）", hr)
	}
	out := new(mfStreamOut)
	if hr := comCall(r.ptr, 6, uintptr(r.audioIdx), uintptr(unsafe.Pointer(&out.mt))); !comFail(hr) && out.mt != 0 {
		m := mfMediaType{ptr: out.mt}
		if v, ok := m.getU32(MFMTAudioSamplesPerSecond); ok {
			r.rate = int(v)
		}
		if v, ok := m.getU32(MFMTAudioNumChannels); ok {
			r.channels = int(v)
		}
		if v, ok := m.getU32(MFMTAudioBitsPerSample); ok {
			r.bits = int(v)
		}
		m.Close()
	}
	runtime.KeepAlive(out)
	if r.rate <= 0 {
		r.rate = 48000
	}
	if r.channels <= 0 {
		r.channels = 2
	}
	if r.bits <= 0 {
		r.bits = 16
	}
	return nil
}

// Seek 跳到指定位置（100ns 粒度）。
func (r *MFSourceReader) Seek(d time.Duration) error {
	var err error
	MFRun(func() { err = r.seekLocked(d) })
	return err
}

func (r *MFSourceReader) seekLocked(d time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return errors.New("reader 已关闭")
	}
	if d < 0 {
		d = 0
	}
	var pv propVariant
	pv.vt = vtI8
	pv.val = int64(d / (100 * time.Nanosecond))
	var nullGUID GUID
	if hr := comCall(r.ptr, 8, uintptr(unsafe.Pointer(&nullGUID)), uintptr(unsafe.Pointer(&pv))); comFail(hr) {
		return fmt.Errorf("跳转失败（0x%08x）", hr)
	}
	r.eof = false
	return nil
}

// VideoStreamIndex / AudioStreamIndex 实际流索引（-1 表示无）。
func (r *MFSourceReader) VideoStreamIndex() int {
	if r == nil {
		return -1
	}
	return r.videoIdx
}

func (r *MFSourceReader) AudioStreamIndex() int {
	if r == nil {
		return -1
	}
	return r.audioIdx
}

// ReadStream 读指定流的样本（stream 传实际流索引）：
// 视频返回 RGBA 帧，音频返回 PCM。ReadSample 不接受 ANY_STREAM，必须给具体流号。
func (r *MFSourceReader) ReadStream(stream int) (kind int, data []byte, ts time.Duration, eof bool, err error) {
	MFRun(func() { kind, data, ts, eof, err = r.readStreamLocked(stream) })
	return
}

func (r *MFSourceReader) readStreamLocked(stream int) (kind int, data []byte, ts time.Duration, eof bool, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return PacketNone, nil, 0, true, errors.New("reader 已关闭")
	}
	if stream < 0 {
		return PacketNone, nil, 0, true, errors.New("无效的流索引")
	}
	sample, ts, eof, err := r.readSample(stream)
	if err != nil || eof || sample == 0 {
		return PacketNone, nil, ts, eof, err
	}
	defer comRelease(sample)

	raw, buf, berr := sampleBytes(sample)
	if buf != 0 {
		defer comRelease(buf)
	}
	if berr != nil {
		return PacketNone, nil, ts, false, berr
	}

	switch stream {
	case r.videoIdx:
		pix, verr := r.toRGBA(raw)
		if verr != nil {
			return PacketNone, nil, ts, false, verr
		}
		return PacketVideo, pix, ts, false, nil
	case r.audioIdx:
		return PacketAudio, raw, ts, false, nil
	}
	return PacketNone, nil, ts, false, nil
}

// toRGBA 把当前格式的原始帧转 RGBA。
func (r *MFSourceReader) toRGBA(data []byte) ([]byte, error) {
	w, h := r.width, r.height
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("系统解码器帧尺寸异常（%d×%d）", w, h)
	}
	need := w * h * 4
	switch r.pixFmt {
	case 1:
		need = w * h * 3 / 2
	case 2:
		need = w * h * 2
	}
	if len(data) < need {
		return nil, fmt.Errorf("系统解码器帧数据不足（%d 字节，需要 %d）", len(data), need)
	}
	pix := make([]byte, w*h*4)
	switch r.pixFmt {
	case 1:
		nv12ToRGBA(pix, data, w, h, r.stride)
	case 2:
		yuy2ToRGBA(pix, data, w, h, r.stride)
	default:
		bgraToRGBA(pix, data, w, h, r.stride)
	}
	return pix, nil
}

// readSample 读指定流的样本。
func (r *MFSourceReader) readSample(stream int) (uintptr, time.Duration, bool, error) {
	out := new(mfSampleOut)
	// ReadSample(streamIndex, controlFlags, *actualStreamIndex, *streamFlags, *timestamp, **sample)
	hr := comCall(r.ptr, 9,
		uintptr(stream), 0,
		uintptr(unsafe.Pointer(&out.actual)),
		uintptr(unsafe.Pointer(&out.flags)),
		uintptr(unsafe.Pointer(&out.ts)),
		uintptr(unsafe.Pointer(&out.sample)))
	runtime.KeepAlive(out)
	flags, ts, sample := out.flags, out.ts, out.sample
	if comFail(hr) {
		r.eof = true
		return 0, 0, true, fmt.Errorf("系统解码读样本失败（0x%08x）", hr)
	}
	if flags&MFSourceReaderFlagEndOfStream != 0 {
		r.eof = true
		comRelease(sample)
		return 0, time.Duration(ts * 100), true, nil
	}
	if flags&MFSourceReaderFlagError != 0 {
		comRelease(sample)
		return 0, 0, false, errors.New("系统解码器报告流错误")
	}
	return sample, time.Duration(ts * 100), false, nil
}

// Close 释放 reader。
func (r *MFSourceReader) Close() {
	if r == nil {
		return
	}
	MFRun(func() { r.closeLocked() })
}

func (r *MFSourceReader) closeLocked() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	r.closed = true
	comRelease(r.ptr)
	r.ptr = 0
}

// sampleBytes 把样本数据拷到 Go 切片（Lock/Unlock 配对），返回字节与 buffer 指针。
func sampleBytes(sample uintptr) ([]byte, uintptr, error) {
	out := new(mfBufOut)
	// IMFSample 继承 IMFAttributes：ConvertToContiguousBuffer 为槽位 41
	if hr := comCall(sample, 41, uintptr(unsafe.Pointer(&out.buf))); comFail(hr) || out.buf == 0 {
		runtime.KeepAlive(out)
		return nil, 0, fmt.Errorf("ConvertToContiguousBuffer 失败（0x%08x）", hr)
	}
	buf := out.buf
	// IMFMediaBuffer.Lock 槽位 3
	if hr := comCall(buf, 3, uintptr(unsafe.Pointer(&out.data)), uintptr(unsafe.Pointer(&out.maxLen)), uintptr(unsafe.Pointer(&out.curLen))); comFail(hr) || out.data == 0 {
		runtime.KeepAlive(out)
		comCall(buf, 4)
		return nil, buf, fmt.Errorf("IMFMediaBuffer.Lock 失败（0x%08x）", hr)
	}
	data, n := out.data, int(out.curLen)
	runtime.KeepAlive(out)
	res := make([]byte, n)
	if n > 0 {
		copy(res, unsafe.Slice((*byte)(unsafe.Pointer(data)), n))
	}
	comCall(buf, 4) // Unlock
	return res, buf, nil
}

// bgraToRGBA 把 RGB32（B,G,R,X）转 RGBA；stride 为负时自底向上读取。
func bgraToRGBA(dst, src []byte, w, h, stride int) {
	if stride == 0 {
		stride = w * 4
	}
	rowBytes := w * 4
	for y := 0; y < h; y++ {
		sy := y
		if stride < 0 {
			sy = h - 1 - y
		}
		off := sy * absInt(stride)
		if off+rowBytes > len(src) {
			break
		}
		row := src[off : off+rowBytes]
		out := dst[y*rowBytes : (y+1)*rowBytes]
		for x := 0; x < rowBytes; x += 4 {
			out[x] = row[x+2]
			out[x+1] = row[x+1]
			out[x+2] = row[x]
			out[x+3] = 255
		}
	}
}

// nv12ToRGBA NV12（Y 平面 + 交错 UV）→ RGBA。
func nv12ToRGBA(dst, src []byte, w, h, stride int) {
	if stride <= 0 {
		stride = w
	}
	ysize := stride * h
	if len(src) < ysize+stride*(h/2) {
		return
	}
	for y := 0; y < h; y++ {
		row := src[y*stride:]
		uvRow := src[ysize+(y/2)*stride:]
		out := dst[y*w*4:]
		for x := 0; x < w; x++ {
			yf := float64(row[x])
			if yf < 16 {
				yf = 16
			}
			u := float64(uvRow[(x/2)*2]) - 128
			v := float64(uvRow[(x/2)*2+1]) - 128
			out[x*4] = clampByte(1.164*(yf-16) + 1.596*v)
			out[x*4+1] = clampByte(1.164*(yf-16) - 0.392*u - 0.813*v)
			out[x*4+2] = clampByte(1.164*(yf-16) + 2.017*u)
			out[x*4+3] = 255
		}
	}
}

// yuy2ToRGBA YUY2（Y0 U Y1 V）→ RGBA。
func yuy2ToRGBA(dst, src []byte, w, h, stride int) {
	if stride <= 0 {
		stride = w * 2
	}
	for y := 0; y < h; y++ {
		if (y+1)*stride > len(src) {
			return
		}
		row := src[y*stride:]
		out := dst[y*w*4:]
		for x := 0; x+1 < w; x += 2 {
			y0 := float64(row[x*2])
			u := float64(row[x*2+1]) - 128
			y1 := float64(row[x*2+2])
			v := float64(row[x*2+3]) - 128
			for k, yf := range [2]float64{y0, y1} {
				if yf < 16 {
					yf = 16
				}
				o := (x + k) * 4
				out[o] = clampByte(1.164*(yf-16) + 1.596*v)
				out[o+1] = clampByte(1.164*(yf-16) - 0.392*u - 0.813*v)
				out[o+2] = clampByte(1.164*(yf-16) + 2.017*u)
				out[o+3] = 255
			}
		}
	}
}

// clampByte 夹到 0~255。
func clampByte(v float64) byte {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return byte(v + 0.5)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

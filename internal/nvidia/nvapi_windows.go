//go:build windows

package nvidia

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

// nvapi64.dll 只导出一个符号 nvapi_QueryInterface，其余函数都靠一个 32 位 id
// 现问它要地址。这些 id 是稳定的公开常量，本机 2026-08-28 逐个验证过能解析到。
const (
	fnInitialize        = 0x0150E828
	fnGetErrorMessage   = 0x6C2D048C
	fnCreateSession     = 0x0694D52E
	fnDestroySession    = 0xDAD9CFF8
	fnLoadSettings      = 0x375DBD6B
	fnSaveSettings      = 0xFCBC7E14
	fnCreateProfile     = 0xCC176068
	fnFindProfileByName = 0x7E4A9A0B
	fnCreateApplication = 0x4347A9DE
	fnSetSetting        = 0x577DD202
	fnGetSetting        = 0x73BF8338
	fnDeleteSetting     = 0xE4A26362

	fnFindApplicationByName = 0xEEE566B2
	fnGetProfileInfo        = 0x61CD6FD6
	fnSaveSettingsToFile    = 0x2BE25DF8
)

// 结构体大小实测得来（本机 NVAPI，2026-08-28），不是照着头文件抄的。
//
// NVAPI 会校验 version 字段：写错了返回 INCOMPATIBLE_STRUCT_VERSION，
// 是一次干净的报错，不会把内存写坏——这一点让"猜错了也不至于毁掉驱动配置"成立。
const (
	sizeProfile = 4116
	sizeSetting = 12320
	sizeApp     = 20492

	verProfile = sizeProfile | (1 << 16) // 0x00011014
	verSetting = sizeSetting | (1 << 16) // 0x00013020
	verApp     = sizeApp | (4 << 16)     // 0x0004500C，实测 v4 被接受
)

// unicodeMax 是 NvAPI_UnicodeString 的固定长度（NvU16[2048]）。
const unicodeMax = 2048

type nvUnicode [unicodeMax]uint16

type nvProfile struct {
	Version       uint32
	ProfileName   nvUnicode
	GPUSupport    uint32
	IsPredefined  uint32
	NumOfApps     uint32
	NumOfSettings uint32
}

type nvBinary struct {
	ValueLength uint32
	ValueData   [4096]uint8
}

// nvValue 是 NVDRS_SETTING 里的联合体。Go 没有联合体，按最大成员开一块，
// 用到的只有开头 4 字节的 u32。
type nvValue struct {
	Raw [4100]byte
}

func (v *nvValue) u32() uint32     { return *(*uint32)(unsafe.Pointer(&v.Raw[0])) }
func (v *nvValue) setU32(x uint32) { *(*uint32)(unsafe.Pointer(&v.Raw[0])) = x }

type nvSetting struct {
	Version             uint32
	SettingName         nvUnicode
	SettingID           uint32
	SettingType         uint32
	SettingLocation     uint32
	IsCurrentPredefined uint32
	IsPredefinedValid   uint32
	Predefined          nvValue
	Current             nvValue
}

type nvApplication struct {
	Version          uint32
	IsPredefined     uint32
	AppName          nvUnicode
	UserFriendlyName nvUnicode
	Launcher         nvUnicode
	FileInFolder     nvUnicode
	Bits             uint32
	CommandLine      nvUnicode
}

// profileName 是我们建的配置在驱动里的名字。
//
// 单独一个配置、只放我们要管的那个 python.exe：不去碰 NVIDIA 预置的配置，
// 也不和别的启动器（例如秋叶那个 "Hanamizuki.Ai ..."）抢同一条记录。
const profileName = "DreamTexture (ComfyUI)"

var (
	initOnce sync.Once
	initErr  error
	dll      *syscall.LazyDLL
	queryIfc *syscall.LazyProc
	procs    = map[uint32]uintptr{}
	procMu   sync.Mutex
)

func load() error {
	initOnce.Do(func() {
		dll = syscall.NewLazyDLL("nvapi64.dll")
		if err := dll.Load(); err != nil {
			initErr = fmt.Errorf("找不到 nvapi64.dll，这台机器可能没有 N 卡或驱动不完整")
			return
		}
		queryIfc = dll.NewProc("nvapi_QueryInterface")
		if err := queryIfc.Find(); err != nil {
			initErr = fmt.Errorf("nvapi64.dll 里没有 nvapi_QueryInterface：%w", err)
			return
		}
		if rc := call(fnInitialize); rc != 0 {
			initErr = fmt.Errorf("NvAPI_Initialize 失败：%s", errText(rc))
		}
	})
	return initErr
}

// proc 按 id 取函数地址，取到的缓存起来。
func proc(id uint32) uintptr {
	procMu.Lock()
	defer procMu.Unlock()
	if p, ok := procs[id]; ok {
		return p
	}
	p, _, _ := queryIfc.Call(uintptr(id))
	procs[id] = p
	return p
}

func call(id uint32, args ...uintptr) int32 {
	p := proc(id)
	if p == 0 {
		return -1 // NVAPI_ERROR
	}
	r, _, _ := syscall.SyscallN(p, args...)
	return int32(r)
}

func errText(rc int32) string {
	var buf [64]byte
	if p := proc(fnGetErrorMessage); p != 0 {
		syscall.SyscallN(p, uintptr(rc), uintptr(unsafe.Pointer(&buf[0])))
	}
	s := string(buf[:])
	if i := strings.IndexByte(s, 0); i >= 0 {
		s = s[:i]
	}
	if s == "" {
		return fmt.Sprintf("错误码 %d", rc)
	}
	return fmt.Sprintf("%s(%d)", s, rc)
}

func toUnicode(s string) nvUnicode {
	var u nvUnicode
	for i, r := range syscall.StringToUTF16(s) {
		if i >= unicodeMax {
			break
		}
		u[i] = r
	}
	return u
}

// normalizeExe 统一成驱动里那套写法：小写 + 正斜杠的完整路径。
//
// 这是照着现有配置抄的——秋叶启动器写进去的就是
// "i:/comfyui/comfyui-aki-v3/python/python.exe"。写成别的形式驱动照样收，
// 但同一个程序会出现两条配置，谁生效说不清。
func normalizeExe(p string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(p), `\`, "/"))
}

// session 把"开会话、载入、用完销毁"这一套包起来。
//
// 锁在同一个系统线程上跑：NVAPI 的会话句柄是进程级的，但 Go 会把 goroutine
// 在线程间挪，而驱动侧的一些实现对此并不友好。锁一下不值几个钱。
func session(fn func(h uintptr) error) error {
	if err := load(); err != nil {
		return err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var h uintptr
	if rc := call(fnCreateSession, uintptr(unsafe.Pointer(&h))); rc != 0 {
		return fmt.Errorf("打开驱动配置会话失败：%s", errText(rc))
	}
	defer call(fnDestroySession, h)

	if rc := call(fnLoadSettings, h); rc != 0 {
		return fmt.Errorf("读取驱动配置失败：%s", errText(rc))
	}
	return fn(h)
}

// findProfile 找我们的配置；create 为 true 时找不到就建一个，并把 exe 注册进去。
func findProfile(h uintptr, exe string, create bool) (uintptr, error) {
	name := toUnicode(profileName)
	var hp uintptr
	rc := call(fnFindProfileByName, h, uintptr(unsafe.Pointer(&name[0])),
		uintptr(unsafe.Pointer(&hp)))
	if rc == 0 {
		return hp, nil
	}
	if !create {
		return 0, nil // 没有就是没有，不算错
	}

	p := nvProfile{Version: verProfile, ProfileName: toUnicode(profileName)}
	if rc := call(fnCreateProfile, h, uintptr(unsafe.Pointer(&p)),
		uintptr(unsafe.Pointer(&hp))); rc != 0 {
		return 0, fmt.Errorf("新建驱动配置失败：%s", errText(rc))
	}

	app := nvApplication{Version: verApp, AppName: toUnicode(normalizeExe(exe))}
	if rc := call(fnCreateApplication, h, hp, uintptr(unsafe.Pointer(&app))); rc != 0 {
		return 0, fmt.Errorf("把 %s 关联到驱动配置失败：%s", exe, errText(rc))
	}
	return hp, nil
}

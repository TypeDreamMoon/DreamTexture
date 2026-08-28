//go:build windows

package nvidia

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf16"
	"unsafe"
)

// sysmemSettingID 是「CUDA 系统内存回退策略」在驱动里的设置项编号。
//
// **实测得来，不是记的**。教训值得写下来：我一开始记成 0x10F9DC81，
// 用 NvAPI_DRS_GetSettingNameFromId 一查，那其实是 "Enable application for
// Optimus"——照着写会去改一个完全无关的设置。
//
// 拿到它的过程：这一项在驱动的设置库里**没有名字**（EnumAvailableSettingIds
// 的 130 项里没有，按名字反查也失败），所以只能让写过它的程序先写一次，
// 再把整个配置库导出来比对。秋叶启动器的「使用共享显存」关掉之后，它的配置
// "Hanamizuki.Ai Stable Diffusion Launcher (Managed)" 里就多出了这一项。
// 同一份导出里，MXGP 那条配置的 0x1033CEC2 / 0x1033DCD3 与具名清单对得上，
// 说明解析没错。
const sysmemSettingID uint32 = 0x10ECECC9

// 取值：1 = 不使用系统内存回退。用户在启动器里把「使用共享显存」关掉之后
// 写进去的就是 1。
const (
	sysmemAllowValue uint32 = 0
	sysmemDenyValue  uint32 = 1
)

// ErrReadOnly 说明为什么只能看不能改。
//
// NVAPI 的公开写接口会拿设置编号去校验驱动自己的设置库，无名设置一律
// 返回 SETTING_NOT_FOUND。实测过六种写法（补 settingName、补 settingType、
// 补 settingLocation…）全部被拒，而同一段代码写已知的 Vertical Sync 立刻成功
// ——所以不是调用姿势的问题。
//
// 剩下的路只有 NvAPI_DRS_LoadSettingsFromFile：把整个配置库（本机 7962 条）
// 导出、按未公开的二进制格式改、再整体导回。那个格式里全是绝对偏移，插入一条
// 就得全局修正，改错一次影响的是机器上所有配置——一个材质工具不该干这个。
var ErrReadOnly = fmt.Errorf(
	"这一项在 NVIDIA 驱动里是个无名设置，公开接口只能读不能写" +
		"（写入一律返回 SETTING_NOT_FOUND）。请在 NVIDIA 控制面板 →" +
		"「管理 3D 设置」→「CUDA - 系统内存回退策略」里改，" +
		"或者用秋叶启动器的「使用共享显存」开关")

// Query 报告某个可执行文件当前的回退策略。
func Query(exe string) Status {
	st := Status{Exe: normalizeExe(exe), Value: FallbackDefault.String()}
	if exe == "" {
		st.Reason = "还没配置 ComfyUI 的 Python 解释器"
		return st
	}
	if err := load(); err != nil {
		st.Reason = err.Error()
		return st
	}

	// 第一步：问 NVAPI 这个 exe 归哪条配置管。这一步公开接口是好使的。
	var owner string
	err := session(func(h uintptr) error {
		name, err := profileOfApp(h, exe)
		owner = name
		return err
	})
	if err != nil {
		st.Reason = err.Error()
		return st
	}
	st.Supported = true
	if owner == "" {
		// 没有任何配置管它 = 跟随驱动全局默认（允许回退）。
		st.Value = FallbackAllow.String()
		return st
	}
	st.Profile = owner

	// 第二步：读取值。这里**不能**用 GetSetting——无名设置会被公开读接口
	// 过滤掉（profile 明明报 numOfSettings=1，EnumSettings 却返回 0 项，
	// GetSetting 报 SETTING_NOT_FOUND）。只有导出的配置库里看得见。
	v, ok, err := readFromDump(owner)
	switch {
	case err != nil:
		st.Reason = err.Error()
	case !ok:
		st.Value = FallbackAllow.String() // 配置在，但没设这一项
	case v == sysmemDenyValue:
		st.Value = FallbackDeny.String()
	default:
		st.Value = FallbackAllow.String()
	}
	return st
}

// Apply 目前做不到，见 ErrReadOnly。
func Apply(exe string, want Fallback) error { return ErrReadOnly }

// profileOfApp 返回管着这个 exe 的配置名；没有就返回空串。
func profileOfApp(h uintptr, exe string) (string, error) {
	name := toUnicode(normalizeExe(exe))
	var hp uintptr
	app := nvApplication{Version: verApp}
	rc := call(fnFindApplicationByName, h, uintptr(unsafe.Pointer(&name[0])),
		uintptr(unsafe.Pointer(&hp)), uintptr(unsafe.Pointer(&app)))
	if rc != 0 {
		return "", nil // 没有配置管它，不算错
	}
	p := nvProfile{Version: verProfile}
	if rc := call(fnGetProfileInfo, h, hp, uintptr(unsafe.Pointer(&p))); rc != 0 {
		return "", fmt.Errorf("读取驱动配置信息失败：%s", errText(rc))
	}
	return fromUnicode(p.ProfileName[:]), nil
}

// readFromDump 把整个驱动配置库导出成文件，再从里面把那一项抠出来。
//
// 绕这么大一圈是没办法的事：见 Query 里第二步的注释。导出本身是只读操作，
// 落一个临时文件，读完就删。
func readFromDump(profile string) (val uint32, found bool, err error) {
	f, err := os.CreateTemp("", "dt-nvdrs-*.bin")
	if err != nil {
		return 0, false, err
	}
	path := f.Name()
	f.Close()
	defer os.Remove(path)

	abs, err := filepath.Abs(path)
	if err != nil {
		return 0, false, err
	}
	if err := session(func(h uintptr) error {
		u := toUnicode(abs)
		if rc := call(fnSaveSettingsToFile, h, uintptr(unsafe.Pointer(&u[0]))); rc != 0 {
			return fmt.Errorf("导出驱动配置失败：%s", errText(rc))
		}
		return nil
	}); err != nil {
		return 0, false, err
	}

	blob, err := os.ReadFile(path)
	if err != nil {
		return 0, false, err
	}
	return findSetting(blob, profile, sysmemSettingID)
}

// findSetting 在导出的配置库里找某条配置的某个设置项。
//
// 格式是逆出来的，没有文档。一条配置的记录长这样：
//
//	53 00 xx 00 | size(u32) | count(u32) | nameOffset(u32)
//	然后 count 个 16 字节的条目： A4 00 10 00 | id(u32) | type(u32) | value(u32)
//
// nameOffset 是文件内的绝对偏移，指向 UTF-16 的配置名。所以找法是：
// 先定位配置名，再往前找那个"nameOffset 正好等于它"的头。
//
// 解析不出来就如实说读不到，不猜——读错一个开关状态，比不显示更糟。
func findSetting(blob []byte, profile string, id uint32) (uint32, bool, error) {
	name := utf16.Encode([]rune(profile))
	raw := make([]byte, len(name)*2)
	for i, c := range name {
		binary.LittleEndian.PutUint16(raw[i*2:], c)
	}

	at := indexBytes(blob, raw)
	if at < 0 {
		return 0, false, fmt.Errorf("在驱动配置里找不到「%s」", profile)
	}

	// 头就在名字前面不远处，往前扫一小段找 nameOffset 对得上的那个。
	const back = 64
	lo := at - back
	if lo < 0 {
		lo = 0
	}
	for h := at - 16; h >= lo; h-- {
		if h+16 > len(blob) {
			continue
		}
		count := binary.LittleEndian.Uint32(blob[h+8:])
		nameOff := binary.LittleEndian.Uint32(blob[h+12:])
		if int(nameOff) != at || count == 0 || count > 512 {
			continue
		}
		p := h + 16
		for i := uint32(0); i < count && p+16 <= len(blob); i++ {
			sid := binary.LittleEndian.Uint32(blob[p+4:])
			if sid == id {
				return binary.LittleEndian.Uint32(blob[p+12:]), true, nil
			}
			p += 16
		}
		return 0, false, nil // 这条配置里没有这一项
	}
	return 0, false, fmt.Errorf("驱动配置的格式没认出来，读不到当前取值")
}

func indexBytes(hay, needle []byte) int {
	if len(needle) == 0 || len(needle) > len(hay) {
		return -1
	}
outer:
	for i := 0; i+len(needle) <= len(hay); i++ {
		for j := range needle {
			if hay[i+j] != needle[j] {
				continue outer
			}
		}
		return i
	}
	return -1
}

func fromUnicode(u []uint16) string {
	for i, c := range u {
		if c == 0 {
			return string(utf16.Decode(u[:i]))
		}
	}
	return string(utf16.Decode(u))
}

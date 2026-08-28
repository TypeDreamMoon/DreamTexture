//go:build windows

package nvidia

import (
	"fmt"
	"unsafe"
)

// sysmemSettingID 是"CUDA 系统内存回退策略"在驱动里的设置项 id。
//
// 这个值**必须实测得来，不能靠记忆**。教训：我一开始记成 0x10F9DC81，
// 用 NvAPI_DRS_GetSettingNameFromId 一查，那其实是 "Enable application for
// Optimus"——照着写会去改一个完全无关的设置。
//
// 它也查不到：驱动公开的 130 个具名设置里没有它（NvAPI_DRS_EnumAvailableSettingIds
// 列不出、NvAPI_DRS_GetSettingIdFromName 按名字反查也失败），是个无名设置。
// 唯一可靠的拿法是让写过它的程序写一次，再从驱动配置库里读出来。
//
// 0 表示还没确定；此时读写一律拒绝，绝不拿一个猜的 id 去写用户的驱动配置。
const sysmemSettingID uint32 = 0

// 取值。同样待实测确认，先按 NVIDIA 控制面板那两项的语义留着。
const (
	sysmemAllowValue uint32 = 0 // 允许回退到系统内存（驱动默认）
	sysmemDenyValue  uint32 = 1 // 优先不使用系统内存回退
)

// ErrUnknownSetting 表示设置项 id 还没测出来。
var ErrUnknownSetting = fmt.Errorf(
	"还不知道「系统内存回退」在驱动里的设置项编号。这一项在 NVIDIA 的驱动里没有名字，" +
		"查不出来，只能从写过它的程序那里比对得到——在没测准之前不会去动你的驱动配置")

// Query 报告某个可执行文件当前的回退策略。
func Query(exe string) Status {
	st := Status{Exe: normalizeExe(exe), Value: FallbackDefault.String()}
	if err := load(); err != nil {
		st.Reason = err.Error()
		return st
	}
	if sysmemSettingID == 0 {
		st.Reason = ErrUnknownSetting.Error()
		return st
	}
	st.Supported = true
	st.Profile = profileName

	err := session(func(h uintptr) error {
		hp, err := findProfile(h, exe, false)
		if err != nil || hp == 0 {
			return err // 没有配置 = 跟随驱动默认
		}
		s := nvSetting{Version: verSetting}
		if rc := call(fnGetSetting, h, hp, uintptr(sysmemSettingID),
			uintptr(unsafe.Pointer(&s))); rc != 0 {
			return nil // 配置里没这一项，同样是跟随默认
		}
		if s.Current.u32() == sysmemDenyValue {
			st.Value = FallbackDeny.String()
		} else {
			st.Value = FallbackAllow.String()
		}
		return nil
	})
	if err != nil {
		st.Supported = false
		st.Reason = err.Error()
	}
	return st
}

// Apply 写入回退策略。FallbackDefault 表示把这一项删掉，回到驱动默认。
func Apply(exe string, want Fallback) error {
	if err := load(); err != nil {
		return err
	}
	if sysmemSettingID == 0 {
		return ErrUnknownSetting
	}
	if exe == "" {
		return fmt.Errorf("还没配置 ComfyUI 的 Python 解释器，无从设置")
	}

	return session(func(h uintptr) error {
		// 要删的时候没有配置就直接算完成，不去凭空建一个再删。
		hp, err := findProfile(h, exe, want != FallbackDefault)
		if err != nil {
			return err
		}
		if hp == 0 {
			return nil
		}

		if want == FallbackDefault {
			rc := call(fnDeleteSetting, h, hp, uintptr(sysmemSettingID))
			if rc != 0 && rc != -163 { // SETTING_NOT_FOUND：本来就没有，不算错
				return fmt.Errorf("清除设置失败：%s", errText(rc))
			}
		} else {
			v := sysmemAllowValue
			if want == FallbackDeny {
				v = sysmemDenyValue
			}
			s := nvSetting{Version: verSetting, SettingID: sysmemSettingID}
			s.Current.setU32(v)
			if rc := call(fnSetSetting, h, hp, uintptr(unsafe.Pointer(&s))); rc != 0 {
				return fmt.Errorf("写入设置失败：%s", errText(rc))
			}
		}

		if rc := call(fnSaveSettings, h); rc != 0 {
			return fmt.Errorf("保存驱动配置失败：%s", errText(rc))
		}
		return nil
	})
}

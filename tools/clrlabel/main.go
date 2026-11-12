// Command clrlabel 查询/清除 Windows 对象的强制完整性标签（Mandatory Integrity Label）。
//
// 背景：Codex 的 Windows 沙箱服务会把项目目录递归打上 Low 完整性标签，
// wails 构建出的 build/bin/Cue.exe 继承 Low 后以低完整性运行，按 Windows
// 「禁止向上写」策略写不了 %APPDATA%\Cue（Medium），表现为保存设置报权限不足、
// Cue.log 停更。构建后清掉 build 目录的标签即可恢复正常。
//
// 用法:
//
//	go run ./tools/clrlabel query <path>   查询路径的完整性级别
//	go run ./tools/clrlabel clear <path>   递归清除该路径（目录+所有文件）的标签
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	advapi32      = syscall.NewLazyDLL("advapi32.dll")
	kernel32      = syscall.NewLazyDLL("kernel32.dll")
	procSetNSSI   = advapi32.NewProc("SetNamedSecurityInfoW")
	procGetNSSI   = advapi32.NewProc("GetNamedSecurityInfoW")
	procLocalFree = kernel32.NewProc("LocalFree")
)

const (
	seFileObject                     = 1
	labelSecurityInformation         = 0x10
	protectedSACLInformation         = 0x40000000
	systemMandatoryLabelAceType byte = 0x11
)

type aclHeader struct {
	AclRevision byte
	Sbz1        byte
	AclSize     uint16
	AceCount    uint16
	Sbz2        uint16
}

type aceHeader struct {
	AceType  byte
	AceFlags byte
	AceSize  uint16
}

// getLabel 返回完整性级别名称（none / Low / Medium / High / System+）
func getLabel(path string) (string, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	var pSacl *aclHeader
	var pSD unsafe.Pointer
	r, _, _ := procGetNSSI.Call(
		uintptr(unsafe.Pointer(p)),
		seFileObject,
		labelSecurityInformation,
		0, 0, 0,
		uintptr(unsafe.Pointer(&pSacl)),
		uintptr(unsafe.Pointer(&pSD)),
	)
	if r != 0 {
		return "", fmt.Errorf("GetNamedSecurityInfoW 失败: %d", r)
	}
	if pSD != nil {
		defer procLocalFree.Call(uintptr(pSD))
	}
	if pSacl == nil || pSacl.AceCount == 0 {
		return "none", nil
	}
	// ACE 区起始紧接 ACL 头
	cur := unsafe.Add(unsafe.Pointer(pSacl), unsafe.Sizeof(*pSacl))
	for i := 0; i < int(pSacl.AceCount); i++ {
		h := (*aceHeader)(cur)
		if h.AceType == systemMandatoryLabelAceType {
			sid := unsafe.Add(cur, 8) // ACE 头(4) + Mask(4) 之后是 SID
			subCount := *(*byte)(unsafe.Add(sid, 1))
			if subCount == 0 {
				return "unknown", nil
			}
			last := *(*uint32)(unsafe.Add(sid, 8+uintptr(subCount-1)*4))
			return levelName(last), nil
		}
		cur = unsafe.Add(cur, uintptr(h.AceSize))
	}
	return "none", nil
}

func levelName(rid uint32) string {
	switch {
	case rid < 0x1000:
		return fmt.Sprintf("Untrusted(0x%x)", rid)
	case rid < 0x2000:
		return fmt.Sprintf("Low(0x%x)", rid)
	case rid < 0x3000:
		return fmt.Sprintf("Medium(0x%x)", rid)
	case rid < 0x4000:
		return fmt.Sprintf("High(0x%x)", rid)
	default:
		return fmt.Sprintf("SystemPlus(0x%x)", rid)
	}
}

// clearOne 清除单个对象的完整性标签：LABEL_SECURITY_INFORMATION|PROTECTED，
// 空 SACL（NULL）表示移除标签。PROTECTED 使该对象不再从父目录继承标签。
func clearOne(path string) error {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	r, _, _ := procSetNSSI.Call(
		uintptr(unsafe.Pointer(p)),
		seFileObject,
		uintptr(labelSecurityInformation|protectedSACLInformation),
		0, 0, 0,
		0, // pSacl = NULL：移除标签
	)
	if r != 0 {
		return fmt.Errorf("SetNamedSecurityInfoW 失败: %d", r)
	}
	return nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "用法: clrlabel <query|clear> <path>")
		os.Exit(2)
	}
	action, root := os.Args[1], os.Args[2]

	switch action {
	case "query":
		lv, err := getLabel(root)
		if err != nil {
			fmt.Printf("%s: 查询失败: %v\n", root, err)
			os.Exit(1)
		}
		fmt.Printf("%s: %s\n", root, lv)

	case "clear":
		// 先清目录（PROTECTED 阻断向子项继承），再递归清子目录与文件
		if err := clearOne(root); err != nil {
			fmt.Printf("目录清除失败 %s: %v\n", root, err)
			os.Exit(1)
		}
		fmt.Printf("已清除目录标签: %s\n", root)
		var failed int
		filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil || p == root {
				return nil
			}
			if e := clearOne(p); e != nil {
				fmt.Printf("  清除失败 %s: %v\n", p, e)
				failed++
			}
			return nil
		})
		if failed > 0 {
			fmt.Printf("清除完成，但有 %d 项失败\n", failed)
			os.Exit(1)
		}
		fmt.Println("清除完成")

	default:
		fmt.Fprintln(os.Stderr, "未知动作:", action)
		os.Exit(2)
	}
}

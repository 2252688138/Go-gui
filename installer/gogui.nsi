; Go-gui
; Author  : Mr.Shu
; Website : www.cq6k.com
; QQ Group: 点击链接加入群聊【Go-gui】：https://qm.qq.com/q/Be12Z5MVJ6
;
; GoGUI 安装器（NSIS 3 / MUI2 / Unicode）
;
; 由 build.ps1 -Installer 调用，产品信息全部通过 /D 传入，脚本内不含硬编码版本：
;   makensis /DPRODUCT=GoGUI /DVERSION=0.1.0 /DVI_VERSION=0.1.0.0 /DCOMPANY=... ^
;            /DCOPYRIGHT=... /DDESC=... /DCHANNEL=release /DAPPEXE=gallery.exe ^
;            /DSRC_EXE=dist\gallery.exe /DOUT_FILE=dist\GoGUI-0.1.0-setup.exe ^
;            /DICON_FILE=ui\assets\icon.ico installer\gogui.nsi
;
; 安装范围（安装向导里选择）：
;   仅当前用户 → %LOCALAPPDATA%\Programs\GoGUI，注册表写 HKCU，不需要管理员
;   所有用户   → %ProgramFiles%\GoGUI，注册表写 HKLM，未提权时自动请求 UAC 重入
; 静默安装可加 /allusers 或 /currentuser 跳过选择页。

Unicode true
SetCompressor /SOLID lzma
SetCompressorDictSize 32
CRCCheck on
XPStyle on
RequestExecutionLevel user
BrandingText "${PRODUCT} ${VERSION}"

!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "FileFunc.nsh"
!include "nsDialogs.nsh"
!include "x64.nsh"

; ---------- 构建期传入的定义（缺省值仅用于直接编译脚本自测）----------
!ifndef PRODUCT
    !define PRODUCT "GoGUI"
!endif
!ifndef VERSION
    !define VERSION "0.0.0"
!endif
!ifndef VI_VERSION
    !define VI_VERSION "0.0.0.0"
!endif
!ifndef COMPANY
    !define COMPANY "Go-gui"
!endif
!ifndef COPYRIGHT
    !define COPYRIGHT "Copyright (c) 2026 Mr.Shu"
!endif
!ifndef DESC
    !define DESC "GoGUI 纯 Go 自绘 GUI 框架"
!endif
!ifndef APPEXE
    !define APPEXE "gallery.exe"
!endif
!ifndef CHANNEL
    !define CHANNEL "dev"
!endif
!ifndef SRC_EXE
    !define SRC_EXE "..\dist\gallery.exe"
!endif
!ifndef OUT_FILE
    !define OUT_FILE "..\dist\${PRODUCT}-${VERSION}-setup.exe"
!endif
!ifndef ICON_FILE
    !define ICON_FILE "..\ui\assets\icon.ico"
!endif

!define UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT}"
!define APP_KEY "Software\${PRODUCT}"

Name "${PRODUCT} ${VERSION}"
OutFile "${OUT_FILE}"
InstallDir "$LOCALAPPDATA\Programs\${PRODUCT}"
; 注意：这里不写 InstallDirRegKey。若写了，NSIS 会在 .onInit 之前把注册表里的旧目录
; 塞进 $INSTDIR，导致「所有用户安装」仍沿用当前用户的旧路径。上次安装目录统一由
; SyncDefaultDir 按当前范围读取后写入。

; 安装程序自身的文件属性（右键 → 详细信息）
VIProductVersion "${VI_VERSION}"
VIAddVersionKey /LANG=2052 "ProductName" "${PRODUCT}"
VIAddVersionKey /LANG=2052 "FileDescription" "${DESC} 安装程序"
VIAddVersionKey /LANG=2052 "FileVersion" "${VERSION}"
VIAddVersionKey /LANG=2052 "ProductVersion" "${VERSION}"
VIAddVersionKey /LANG=2052 "CompanyName" "${COMPANY}"
VIAddVersionKey /LANG=2052 "LegalCopyright" "${COPYRIGHT}"

!define MUI_ICON "${ICON_FILE}"
!define MUI_UNICON "${ICON_FILE}"
!define MUI_ABORTWARNING
!define MUI_WELCOMEPAGE_TITLE "${PRODUCT} ${VERSION} 安装向导"
!define MUI_WELCOMEPAGE_TEXT "本向导将引导你完成 ${PRODUCT} 的安装。$\r$\n$\r$\n${DESC}$\r$\n$\r$\n可以选择「仅为我安装」（免管理员）或「为所有用户安装」（需要管理员权限）。"
!define MUI_FINISHPAGE_RUN "$INSTDIR\${APPEXE}"
!define MUI_FINISHPAGE_RUN_TEXT "立即运行 ${PRODUCT}"

Var InstallAll     ; 1 = 为所有用户安装，0 = 仅当前用户
Var SkipScope      ; 1 = 命令行已指定范围，跳过选择页
Var IsAdmin        ; 1 = 当前进程已提权
Var ScopeAllRadio
Var ScopeMeRadio
Var ScopeDialog

; ---------------- 页面 ----------------
!insertmacro MUI_PAGE_WELCOME
Page custom ScopePage ScopePageLeave
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "SimpChinese"

; ---------------- 工具函数 ----------------
; CheckAdmin：尝试写 HKLM 判断是否已提权（比组成员判断更可靠）
Function CheckAdmin
    StrCpy $IsAdmin 0
    ClearErrors
    WriteRegStr HKLM "${APP_KEY}_probe" "p" "1"
    ${IfNot} ${Errors}
        StrCpy $IsAdmin 1
        DeleteRegKey HKLM "${APP_KEY}_probe"
    ${EndIf}
FunctionEnd

; SetScope：只切换注册表视图与 shell 变量上下文，绝不改动 $INSTDIR。
; （安装阶段必须用它，否则会覆盖用户在目录页选择的自定义目录）
Function SetScope
    ${If} ${RunningX64}
        SetRegView 64
    ${EndIf}
    ${If} $InstallAll == 1
        SetShellVarContext all
    ${Else}
        SetShellVarContext current
    ${EndIf}
FunctionEnd

; DefaultDirForScope：当前范围的默认安装目录
Function DefaultDirForScope
    ${If} $InstallAll == 1
        ${If} ${RunningX64}
            StrCpy $R9 "$PROGRAMFILES64\${PRODUCT}"
        ${Else}
            StrCpy $R9 "$PROGRAMFILES\${PRODUCT}"
        ${EndIf}
    ${Else}
        StrCpy $R9 "$LOCALAPPDATA\Programs\${PRODUCT}"
    ${EndIf}
FunctionEnd

; SyncDefaultDir：在两种范围的默认目录之间切换，但只在「$INSTDIR 仍是某个默认目录」
; 或「$INSTDIR 为空」时进行——用户自定义的目录（含命令行 /D=）一律保留。
Function SyncDefaultDir
    Call DefaultDirForScope ; $R9 = 当前范围默认目录

    ; 沿用该范围上次安装的目录（升级安装时更符合直觉）
    ${If} $InstallAll == 1
        ReadRegStr $R6 HKLM "${APP_KEY}" "InstallDir"
    ${Else}
        ReadRegStr $R6 HKCU "${APP_KEY}" "InstallDir"
    ${EndIf}
    ${If} $R6 != ""
        StrCpy $R9 $R6
    ${EndIf}

    StrCpy $R8 "$LOCALAPPDATA\Programs\${PRODUCT}"
    ${If} ${RunningX64}
        StrCpy $R7 "$PROGRAMFILES64\${PRODUCT}"
    ${Else}
        StrCpy $R7 "$PROGRAMFILES\${PRODUCT}"
    ${EndIf}

    ${If} $INSTDIR == ""
    ${OrIf} $INSTDIR == $R8
    ${OrIf} $INSTDIR == $R7
        StrCpy $INSTDIR $R9
    ${EndIf}
FunctionEnd

; ElevateForAllUsers：需要所有用户安装但未提权 → 以 runas 重入并退出当前实例
Function ElevateForAllUsers
    ${If} ${Silent}
        SetErrorLevel 5
        Quit
    ${EndIf}
    MessageBox MB_ICONINFORMATION|MB_OK "「为所有用户安装」需要管理员权限，即将请求提权，请在 UAC 提示中选择「是」。"
    ExecShell "runas" "$EXEPATH" "/allusers"
    Quit
FunctionEnd

Function .onInit
    StrCpy $SkipScope 0
    Call CheckAdmin
    ; 默认范围：已提权 → 所有用户；普通双击 → 仅当前用户
    ${If} $IsAdmin == 1
        StrCpy $InstallAll 1
    ${Else}
        StrCpy $InstallAll 0
    ${EndIf}

    ${GetParameters} $R0
    ClearErrors
    ${GetOptions} $R0 "/currentuser" $R1
    ${IfNot} ${Errors}
        StrCpy $InstallAll 0
        StrCpy $SkipScope 1
    ${EndIf}
    ClearErrors
    ${GetOptions} $R0 "/allusers" $R1
    ${IfNot} ${Errors}
        StrCpy $InstallAll 1
        StrCpy $SkipScope 1
    ${EndIf}

    ${If} $InstallAll == 1
    ${AndIf} $IsAdmin == 0
        Call ElevateForAllUsers
    ${EndIf}

    Call SetScope
    Call SyncDefaultDir
FunctionEnd

; ---------------- 安装范围选择页 ----------------
Function ScopePage
    ${If} $SkipScope == 1
        Abort ; 命令行已指定范围，直接跳过本页
    ${EndIf}

    !insertmacro MUI_HEADER_TEXT "选择安装范围" "请选择 ${PRODUCT} 的安装范围"
    nsDialogs::Create 1018
    Pop $ScopeDialog
    ${If} $ScopeDialog == error
        Abort
    ${EndIf}

    ${NSD_CreateLabel} 0 0 100% 30u "为谁安装 ${PRODUCT} ${VERSION}？$\r$\n两种方式都可以随时在「设置 → 应用」中卸载。"
    Pop $0

    ${NSD_CreateRadioButton} 0 38u 100% 12u "仅为我安装（推荐，无需管理员权限）"
    Pop $ScopeMeRadio
    ${NSD_CreateLabel} 14u 52u 92% 24u "安装到 $LOCALAPPDATA\Programs\${PRODUCT}，仅当前用户可用。"
    Pop $0

    ${NSD_CreateRadioButton} 0 82u 100% 12u "为所有用户安装（需要管理员权限）"
    Pop $ScopeAllRadio
    ${NSD_CreateLabel} 14u 96u 92% 24u "安装到 $PROGRAMFILES\${PRODUCT}，本机所有用户可用；确认后会弹出 UAC。"
    Pop $0

    ${If} $InstallAll == 1
        ${NSD_SetState} $ScopeAllRadio ${BST_CHECKED}
    ${Else}
        ${NSD_SetState} $ScopeMeRadio ${BST_CHECKED}
    ${EndIf}

    nsDialogs::Show
FunctionEnd

Function ScopePageLeave
    ${If} $SkipScope == 1
        Abort
    ${EndIf}

    ${NSD_GetState} $ScopeAllRadio $0
    ${If} $0 == ${BST_CHECKED}
        StrCpy $InstallAll 1
    ${Else}
        StrCpy $InstallAll 0
    ${EndIf}

    ${If} $InstallAll == 1
    ${AndIf} $IsAdmin == 0
        Call ElevateForAllUsers
    ${EndIf}

    ; 只更新上下文与默认目录，用户在下一页自定义的目录不会被覆盖
    Call SetScope
    Call SyncDefaultDir
FunctionEnd

; ---------------- 安装 ----------------
Section "主程序" SEC_MAIN
    SectionIn RO
    ; 只切换上下文，不动 $INSTDIR —— 保证目录页选择的自定义路径生效
    Call SetScope

    SetOutPath "$INSTDIR"
    File "/oname=${APPEXE}" "${SRC_EXE}"
    WriteUninstaller "$INSTDIR\uninstall.exe"

    ; 开始菜单 + 桌面快捷方式
    CreateDirectory "$SMPROGRAMS\${PRODUCT}"
    CreateShortCut "$SMPROGRAMS\${PRODUCT}\${PRODUCT}.lnk" "$INSTDIR\${APPEXE}" "" "$INSTDIR\${APPEXE}" 0
    CreateShortCut "$SMPROGRAMS\${PRODUCT}\卸载 ${PRODUCT}.lnk" "$INSTDIR\uninstall.exe"
    CreateShortCut "$DESKTOP\${PRODUCT}.lnk" "$INSTDIR\${APPEXE}" "" "$INSTDIR\${APPEXE}" 0

    ; 控制面板「应用和功能」条目
    WriteRegStr SHCTX "${UNINST_KEY}" "DisplayName" "${PRODUCT}"
    WriteRegStr SHCTX "${UNINST_KEY}" "DisplayVersion" "${VERSION}"
    WriteRegStr SHCTX "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\${APPEXE}"
    WriteRegStr SHCTX "${UNINST_KEY}" "Publisher" "${COMPANY}"
    WriteRegStr SHCTX "${UNINST_KEY}" "Comments" "${DESC}（${CHANNEL} 通道）"
    WriteRegStr SHCTX "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
    WriteRegStr SHCTX "${UNINST_KEY}" "UninstallString" '"$INSTDIR\uninstall.exe"'
    WriteRegStr SHCTX "${UNINST_KEY}" "QuietUninstallString" '"$INSTDIR\uninstall.exe" /S'
    WriteRegDWORD SHCTX "${UNINST_KEY}" "NoModify" 1
    WriteRegDWORD SHCTX "${UNINST_KEY}" "NoRepair" 1

    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD SHCTX "${UNINST_KEY}" "EstimatedSize" "$0"

    ; 应用自身键：记录安装目录 / 版本 / 范围，供升级与卸载定位
    WriteRegStr SHCTX "${APP_KEY}" "InstallDir" "$INSTDIR"
    WriteRegStr SHCTX "${APP_KEY}" "Version" "${VERSION}"
    WriteRegStr SHCTX "${APP_KEY}" "Channel" "${CHANNEL}"
    ${If} $InstallAll == 1
        WriteRegStr SHCTX "${APP_KEY}" "Scope" "all"
    ${Else}
        WriteRegStr SHCTX "${APP_KEY}" "Scope" "current"
    ${EndIf}
SectionEnd

; ---------------- 卸载 ----------------
Function un.onInit
    ${If} ${RunningX64}
        SetRegView 64
    ${EndIf}

    ; 依据 HKLM 里是否有安装记录判断当初的安装范围
    ReadRegStr $0 HKLM "${APP_KEY}" "InstallDir"
    ${If} $0 != ""
        StrCpy $InstallAll 1
        SetShellVarContext all
        ; 删除 HKLM 需要管理员：未提权则拉起卸载器自身
        ClearErrors
        WriteRegStr HKLM "${APP_KEY}_probe" "p" "1"
        ${If} ${Errors}
            ExecShell "runas" "$INSTDIR\uninstall.exe" ""
            Quit
        ${EndIf}
        DeleteRegKey HKLM "${APP_KEY}_probe"
    ${Else}
        StrCpy $InstallAll 0
        SetShellVarContext current
        ReadRegStr $0 HKCU "${APP_KEY}" "InstallDir"
        ${If} $0 != ""
        ${AndIf} $INSTDIR == ""
            StrCpy $INSTDIR $0
        ${EndIf}
        ${If} $INSTDIR == ""
            StrCpy $INSTDIR "$LOCALAPPDATA\Programs\${PRODUCT}"
        ${EndIf}
    ${EndIf}
FunctionEnd

Section "Uninstall"
    ; 结束仍在运行的实例，避免文件占用导致残留
    ExecWait '"$SYSDIR\taskkill.exe" /IM "${APPEXE}" /F'

    Delete "$INSTDIR\${APPEXE}"
    Delete "$INSTDIR\uninstall.exe"
    RMDir /r "$INSTDIR"

    Delete "$SMPROGRAMS\${PRODUCT}\${PRODUCT}.lnk"
    Delete "$SMPROGRAMS\${PRODUCT}\卸载 ${PRODUCT}.lnk"
    RMDir "$SMPROGRAMS\${PRODUCT}"
    Delete "$DESKTOP\${PRODUCT}.lnk"

    DeleteRegKey SHCTX "${UNINST_KEY}"
    DeleteRegKey SHCTX "${APP_KEY}"

    ; 顺带清理另一种范围的残留（先装当前用户、后装所有用户的情况）
    ${If} $InstallAll == 1
        DeleteRegKey HKCU "${UNINST_KEY}"
        DeleteRegKey HKCU "${APP_KEY}"
    ${Else}
        DeleteRegKey HKLM "${UNINST_KEY}"
        DeleteRegKey HKLM "${APP_KEY}"
    ${EndIf}
SectionEnd

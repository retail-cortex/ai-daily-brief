!define APP_NAME "AI Daily Brief"
!define COMP_NAME "AI Daily Brief Developer"
!define VERSION "1.0.0"
!define OUT_FILE "ai-daily-brief-setup.exe"
!define BIN_NAME "ai_daily_brief_windows_amd64.exe"

Name "${APP_NAME}"
OutFile "${OUT_FILE}"
InstallDir "$PROGRAMFILES64\${APP_NAME}"
RequestExecutionLevel admin

Page directory
Page instfiles

Section "Install"
  SetOutPath "$INSTDIR"
  
  # Copy compiled binary (renamed to simple executable), config, and docs
  File /oname=ai-daily-brief.exe "${BIN_NAME}"
  File "..\..\.env.toml"
  File "..\..\README.md"
  
  # Create Start Menu shortcuts
  CreateDirectory "$SMPROGRAMS\${APP_NAME}"
  CreateShortcut "$SMPROGRAMS\${APP_NAME}\${APP_NAME}.lnk" "$INSTDIR\ai-daily-brief.exe" "" "$INSTDIR\ai-daily-brief.exe" 0
  CreateShortcut "$SMPROGRAMS\${APP_NAME}\Uninstall.lnk" "$INSTDIR\uninstall.exe"
  
  WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\ai-daily-brief.exe"
  Delete "$INSTDIR\.env.toml"
  Delete "$INSTDIR\README.md"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"
  
  Delete "$SMPROGRAMS\${APP_NAME}\${APP_NAME}.lnk"
  Delete "$SMPROGRAMS\${APP_NAME}\Uninstall.lnk"
  RMDir "$SMPROGRAMS\${APP_NAME}"
SectionEnd

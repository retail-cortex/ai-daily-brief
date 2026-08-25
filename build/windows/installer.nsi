# Copyright 2026 Retail Cortex
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

!ifndef APP_NAME
  !define APP_NAME "AI Daily Brief"
!endif
!ifndef COMP_NAME
  !define COMP_NAME "AI Daily Brief Developer"
!endif
!ifndef VERSION
  !define VERSION "1.0.0"
!endif
!ifndef OUT_FILE
  !define OUT_FILE "ai-daily-brief-setup.exe"
!endif
!ifndef BIN_NAME
  !define BIN_NAME "ai_daily_brief_windows_amd64.exe"
!endif

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

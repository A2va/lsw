# Bypass Windows 11 Network Requirement
reg add "HKLM\SYSTEM\Setup\LabConfig" /v "BypassNRO" /t REG_DWORD /d 1 /f
reg add "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\OOBE" /v "BypassNRO" /t REG_DWORD /d 1 /f

schtasks /create /sc minute /mo 10 /tn "autozap" /tr "%cd%\autozap.exe" /v1 /F
pause
@ECHO OFF

:main
wait_for_internet
set "logfile=output_%date:~-4,4%%date:~-10,2%%date:~-7,2%_%time:~0,2%%time:~3,2%%time:~6,2%.log"
set "logfile=%logfile: =0%"
autozap > %logfile% 2>&1
if %ERRORLEVEL% EQU 0 (
  del /Q %logfile%
)
timeout /t 600 > NUL
goto:main

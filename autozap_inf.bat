@ECHO OFF

:main
wait_for_internet
autozap
timeout /t 600 > NUL
goto:main

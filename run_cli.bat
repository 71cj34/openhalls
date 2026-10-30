@echo off

setlocal EnableDelayedExpansion

set "COUNT=0"

for /R "%CD%" %%F in (hallview.exe) do (
    if exist "%%F" (
        set /A COUNT+=1
        set "MATCH!COUNT!=%%F"
    )
)

if !COUNT! EQU 0 (
    echo No hallview.exe found under "%CD%".
    echo Build it first with src\hallview\buildonly.bat, or download it from the releases page.
    echo .
    set /P "DUMMY=Press ENTER to exit... "
    endlocal
    exit /b 1
)

if !COUNT! GTR 1 (
    echo Found !COUNT! copies of hallview.exe:
    for /L %%I in (1,1,!COUNT!) do echo   !MATCH%%I!
    echo .
    echo Please delete or move all but one of them, then run this script again.
    echo .
    set /P "DUMMY=Press ENTER to exit... "
    endlocal
    exit /b 1
)

echo Found: "!MATCH1!"
echo.

:ASK
set "CHOICE="
set /P "CHOICE=Press ENTER to run it, n to exit: "
if not defined CHOICE (
    echo Running "!MATCH1!" ...
    "!MATCH1!" %*
    endlocal
    exit /b 0
)
if /I "!CHOICE!"=="n" (
    endlocal
    exit /b 0
)
echo Please press ENTER to run, or type n to exit.
goto ASK

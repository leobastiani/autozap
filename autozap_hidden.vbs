Set objFSO = CreateObject("Scripting.FileSystemObject")
currentDir = objFSO.GetParentFolderName(WScript.ScriptFullName)

Set objShell = CreateObject("Wscript.Shell")
objShell.CurrentDirectory = currentDir
objShell.Run "autozap_inf.bat", 0, True
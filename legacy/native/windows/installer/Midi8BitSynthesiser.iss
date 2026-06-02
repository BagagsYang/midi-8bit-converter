#define AppName "OctaBit"
#define AppExeName "Midi8BitSynthesiser.App.exe"

#ifndef AppVersion
  #define AppVersion "1.0.0"
#endif
#ifndef PublishDir
  #define PublishDir "..\artifacts\publish\win-x64"
#endif
#ifndef MinimumWindowsVersion
  #define MinimumWindowsVersion "10.0.17763.0"
#endif
#ifndef MinimumWindowsBuild
  #define MinimumWindowsBuild "17763"
#endif
#ifndef SupportedArchitecture
  #define SupportedArchitecture "x64"
#endif

[Setup]
AppId={{92B65F2B-0F9B-4F0D-8A18-6E7E28A9F4D5}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppName}
DefaultDirName={autopf}\{#AppName}
DefaultGroupName={#AppName}
OutputDir=..\artifacts\installer
OutputBaseFilename=OctaBit-setup-{#AppVersion}
Compression=lzma
SolidCompression=yes
WizardStyle=modern
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
PrivilegesRequired=admin
MinVersion=10.0
UninstallDisplayIcon={app}\{#AppExeName}
InfoBeforeFile=RuntimeNotice.txt

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
Source: "{#PublishDir}\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{group}\{#AppName}"; Filename: "{app}\{#AppExeName}"
Name: "{autodesktop}\{#AppName}"; Filename: "{app}\{#AppExeName}"; Tasks: desktopicon

[Run]
Filename: "{app}\{#AppExeName}"; Description: "Launch {#AppName}"; Flags: nowait postinstall skipifsilent

[Code]
function IsSupportedWindowsBuild(): Boolean;
var
  Version: TWindowsVersion;
begin
  GetWindowsVersionEx(Version);
  Result := (Version.Major > 10) or
    ((Version.Major = 10) and ((Version.Minor > 0) or ((Version.Minor = 0) and (Version.Build >= {#MinimumWindowsBuild}))));
end;

function InitializeSetup(): Boolean;
begin
  Result := True;

  if not IsWin64 then
  begin
    MsgBox(
      '{#AppName} requires a 64-bit version of Windows. This release is built for {#SupportedArchitecture}.',
      mbCriticalError,
      MB_OK);
    Result := False;
    exit;
  end;

  if not IsSupportedWindowsBuild() then
  begin
    MsgBox(
      '{#AppName} requires Windows {#MinimumWindowsVersion} or newer.' + #13#10 + #13#10 +
      'This self-contained release does not require the .NET SDK, but it does require a supported Windows version.',
      mbCriticalError,
      MB_OK);
    Result := False;
  end;
end;

{
  Main program demonstrating various Pascal patterns for parser testing.
  Tests: entry points, procedures, functions, units.
}
program TestApp;

{$mode objfpc}{$H+}

uses
  SysUtils, Classes, Utils;

const
  MAX_RETRIES = 3;
  DEFAULT_TIMEOUT = 30;
  APP_NAME = 'TestApp';

var
  AppVersion: string = '1.0.0';
  Initialized: Boolean = False;

{ Forward declarations }
procedure SetupLogging(const Level: string); forward;
procedure RunPipeline; forward;
function FetchData: string; forward;
function TransformData(const Data: string): string; forward;
procedure SaveData(const Data: string); forward;

{
  Main entry point - should be marked as reachable.
}
procedure Main;
var
  Config: TConfig;
  Server: TServer;
  Items: array of string;
  Result, Output: string;
begin
  WriteLn('Starting ', APP_NAME);

  { Load configuration }
  LoadConfig(Config);

  if not Initialize(Config) then
  begin
    WriteLn(StdErr, 'Initialization failed');
    Halt(1);
  end;

  { Create and start server }
  Server := TServer.Create(Config);
  try
    Server.Start;

    { Using utility functions }
    SetLength(Items, 3);
    Items[0] := 'a';
    Items[1] := 'b';
    Items[2] := 'c';
    Result := ProcessData(Items);
    Output := FormatOutput(Result);
    WriteLn(Output);

    { Calling transitive functions }
    RunPipeline;

    { Cleanup }
    Server.Stop;
  finally
    Server.Free;
  end;
end;

{
  Initialize application - called from main, should be reachable.
}
function Initialize(const Config: TConfig): Boolean;
begin
  SetupLogging(Config.LogLevel);
  Initialized := True;
  Result := True;
end;

{
  Internal helper - called from initialize, should be reachable.
}
procedure SetupLogging(const Level: string);
begin
  WriteLn('Setting log level to: ', Level);
end;

{
  Orchestrate data pipeline - tests transitive reachability.
}
procedure RunPipeline;
var
  Data, Transformed: string;
begin
  Data := FetchData;
  Transformed := TransformData(Data);
  SaveData(Transformed);
end;

{
  Fetch data - called by RunPipeline, should be reachable.
}
function FetchData: string;
begin
  Result := 'sample data';
end;

{
  Transform data - called by RunPipeline, should be reachable.
}
function TransformData(const Data: string): string;
begin
  Result := 'transformed: ' + Data;
end;

{
  Save data - called by RunPipeline, should be reachable.
}
procedure SaveData(const Data: string);
begin
  WriteLn('Saving: ', Data);
end;

{ ============================================================================ }
{ Dead code section - procedures/functions that are never called              }
{ ============================================================================ }

{
  This procedure is never called - DEAD CODE.
}
procedure UnusedProcedure;
begin
  WriteLn('This is never executed');
end;

{
  Also never called - DEAD CODE.
}
function AnotherUnused: string;
begin
  Result := 'dead';
end;

{
  Starts a chain of dead code - DEAD CODE.
}
procedure DeadChainStart;
begin
  DeadChainMiddle;
end;

{
  In the middle of dead chain - DEAD CODE (transitive).
}
procedure DeadChainMiddle;
begin
  DeadChainEnd;
end;

{
  End of dead chain - DEAD CODE (transitive).
}
procedure DeadChainEnd;
begin
  WriteLn('End of dead chain');
end;

{ Program entry point }
begin
  Main;
end.

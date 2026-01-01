{
  Utils unit - tests unit structure, types, and functions.
  Tests: classes, records, interfaces, procedures, functions.
}
unit Utils;

{$mode objfpc}{$H+}

interface

uses
  SysUtils, Classes;

type
  { Log level enumeration }
  TLogLevel = (llDebug, llInfo, llWarn, llError);

  { HTTP method enumeration }
  THttpMethod = (hmGet, hmPost, hmPut, hmDelete, hmPatch);

  { Configuration record - tests record extraction }
  TConfig = record
    Host: string;
    Port: Integer;
    LogLevel: string;
  end;

  { Logger class }
  TLogger = class
  private
    FPrefix: string;
    FLevel: TLogLevel;
  public
    constructor Create(const APrefix: string);
    procedure Info(const Message: string);
    procedure Debug(const Message: string); { DEAD CODE }
    procedure Error(const Message: string); { DEAD CODE }
    property Prefix: string read FPrefix;
    property Level: TLogLevel read FLevel write FLevel;
  end;

  { Handler interface - tests interface extraction }
  IHandler = interface
    ['{A1B2C3D4-E5F6-7890-ABCD-EF1234567890}']
    function Handle(const Input: string): string;
    function GetName: string;
  end;

  { Server class - tests class with methods }
  TServer = class
  private
    FConfig: TConfig;
    FRunning: Boolean;
    FLogger: TLogger;
    procedure Listen;
    procedure HandleConnection(Connection: Pointer); { DEAD CODE }
  public
    constructor Create(const AConfig: TConfig);
    destructor Destroy; override;
    procedure Start;
    procedure Stop;
    function IsRunning: Boolean;
  end;

  { Echo handler - DEAD CODE }
  TEchoHandler = class(TInterfacedObject, IHandler)
  public
    function Handle(const Input: string): string;
    function GetName: string;
  end;

  { Upper handler - DEAD CODE }
  TUpperHandler = class(TInterfacedObject, IHandler)
  public
    function Handle(const Input: string): string;
    function GetName: string;
  end;

  { Generic container - tests generic class }
  generic TContainer<T> = class
  private
    FItems: array of T;
    FCount: Integer;
  public
    procedure Add(const Item: T);
    function Get(Index: Integer): T;
    function Count: Integer;
    procedure Clear;
  end;

  { Cache class - DEAD CODE }
  TCache = class
  private
    FData: TStringList;
  public
    constructor Create;
    destructor Destroy; override;
    procedure SetValue(const Key, Value: string);
    function GetValue(const Key: string): string;
    procedure Delete(const Key: string);
    procedure Clear;
  end;

{ Public functions }
procedure LoadConfig(out Config: TConfig);
function ProcessData(const Items: array of string): string;
function FormatOutput(const Data: string): string;

{ DEAD CODE functions }
function HashString(const S: string): string;
function FilterStrings(const Items: array of string; Predicate: TFunc<string, Boolean>): TStringArray;
function MapStrings(const Items: array of string; Transform: TFunc<string, string>): TStringArray;

implementation

{ ============================================================================ }
{ TLogger implementation                                                       }
{ ============================================================================ }

constructor TLogger.Create(const APrefix: string);
begin
  inherited Create;
  FPrefix := APrefix;
  FLevel := llInfo;
end;

procedure TLogger.Info(const Message: string);
begin
  WriteLn('[INFO] ', FPrefix, ': ', Message);
end;

{ DEAD CODE }
procedure TLogger.Debug(const Message: string);
begin
  if FLevel <= llDebug then
    WriteLn('[DEBUG] ', FPrefix, ': ', Message);
end;

{ DEAD CODE }
procedure TLogger.Error(const Message: string);
begin
  WriteLn('[ERROR] ', FPrefix, ': ', Message);
end;

{ ============================================================================ }
{ TServer implementation                                                       }
{ ============================================================================ }

constructor TServer.Create(const AConfig: TConfig);
begin
  inherited Create;
  FConfig := AConfig;
  FRunning := False;
  FLogger := TLogger.Create('server');
end;

destructor TServer.Destroy;
begin
  FLogger.Free;
  inherited Destroy;
end;

procedure TServer.Listen;
begin
  { Simulated listening }
end;

{ DEAD CODE }
procedure TServer.HandleConnection(Connection: Pointer);
begin
  { Handle connection }
end;

procedure TServer.Start;
begin
  FRunning := True;
  FLogger.Info('Starting server on ' + FConfig.Host + ':' + IntToStr(FConfig.Port));
  Listen;
end;

procedure TServer.Stop;
begin
  FRunning := False;
  FLogger.Info('Stopping server');
end;

function TServer.IsRunning: Boolean;
begin
  Result := FRunning;
end;

{ ============================================================================ }
{ TEchoHandler implementation - DEAD CODE                                      }
{ ============================================================================ }

function TEchoHandler.Handle(const Input: string): string;
begin
  Result := Input;
end;

function TEchoHandler.GetName: string;
begin
  Result := 'echo';
end;

{ ============================================================================ }
{ TUpperHandler implementation - DEAD CODE                                     }
{ ============================================================================ }

function TUpperHandler.Handle(const Input: string): string;
begin
  Result := UpperCase(Input);
end;

function TUpperHandler.GetName: string;
begin
  Result := 'upper';
end;

{ ============================================================================ }
{ TContainer implementation                                                    }
{ ============================================================================ }

procedure TContainer.Add(const Item: T);
begin
  SetLength(FItems, Length(FItems) + 1);
  FItems[High(FItems)] := Item;
  Inc(FCount);
end;

function TContainer.Get(Index: Integer): T;
begin
  if (Index >= 0) and (Index < FCount) then
    Result := FItems[Index]
  else
    raise EListError.CreateFmt('Index out of bounds: %d', [Index]);
end;

function TContainer.Count: Integer;
begin
  Result := FCount;
end;

procedure TContainer.Clear;
begin
  SetLength(FItems, 0);
  FCount := 0;
end;

{ ============================================================================ }
{ TCache implementation - DEAD CODE                                            }
{ ============================================================================ }

constructor TCache.Create;
begin
  inherited Create;
  FData := TStringList.Create;
end;

destructor TCache.Destroy;
begin
  FData.Free;
  inherited Destroy;
end;

procedure TCache.SetValue(const Key, Value: string);
begin
  FData.Values[Key] := Value;
end;

function TCache.GetValue(const Key: string): string;
begin
  Result := FData.Values[Key];
end;

procedure TCache.Delete(const Key: string);
var
  Index: Integer;
begin
  Index := FData.IndexOfName(Key);
  if Index >= 0 then
    FData.Delete(Index);
end;

procedure TCache.Clear;
begin
  FData.Clear;
end;

{ ============================================================================ }
{ Public functions implementation                                              }
{ ============================================================================ }

procedure LoadConfig(out Config: TConfig);
begin
  Config.Host := 'localhost';
  Config.Port := 8080;
  Config.LogLevel := 'info';
end;

function ProcessData(const Items: array of string): string;
var
  I: Integer;
begin
  Result := '';
  for I := Low(Items) to High(Items) do
  begin
    Result := Result + UpperCase(Items[I]);
    if I < High(Items) then
      Result := Result + ', ';
  end;
end;

function FormatOutput(const Data: string): string;
begin
  Result := 'Result: ' + Data;
end;

{ DEAD CODE }
function HashString(const S: string): string;
var
  Hash: Cardinal;
  I: Integer;
begin
  Hash := 5381;
  for I := 1 to Length(S) do
    Hash := ((Hash shl 5) + Hash) + Ord(S[I]);
  Result := IntToHex(Hash, 8);
end;

{ DEAD CODE }
function FilterStrings(const Items: array of string; Predicate: TFunc<string, Boolean>): TStringArray;
var
  I, Count: Integer;
begin
  SetLength(Result, Length(Items));
  Count := 0;
  for I := Low(Items) to High(Items) do
  begin
    if Predicate(Items[I]) then
    begin
      Result[Count] := Items[I];
      Inc(Count);
    end;
  end;
  SetLength(Result, Count);
end;

{ DEAD CODE }
function MapStrings(const Items: array of string; Transform: TFunc<string, string>): TStringArray;
var
  I: Integer;
begin
  SetLength(Result, Length(Items));
  for I := Low(Items) to High(Items) do
    Result[I] := Transform(Items[I]);
end;

end.

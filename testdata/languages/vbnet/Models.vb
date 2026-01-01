'
' Models module - tests classes, interfaces, structures, and enums.
' Tests: class extraction, interface extraction, enum extraction.
'

Imports System
Imports System.Collections.Generic

Namespace TestApp

    ''' <summary>
    ''' Log level enumeration.
    ''' </summary>
    Public Enum LogLevel
        Debug = 0
        Info = 1
        Warn = 2
        [Error] = 3
    End Enum

    ''' <summary>
    ''' HTTP method enumeration.
    ''' </summary>
    Public Enum HttpMethod
        [Get]
        Post
        Put
        Delete
        Patch
    End Enum

    ''' <summary>
    ''' Configuration structure - tests structure extraction.
    ''' </summary>
    Public Structure Config
        Public Host As String
        Public Port As Integer
        Public LogLevel As String

        Public Sub New(host As String, port As Integer, logLevel As String)
            Me.Host = host
            Me.Port = port
            Me.LogLevel = logLevel
        End Sub

        Public Function Validate() As Boolean
            If String.IsNullOrEmpty(Host) Then Return False
            If Port <= 0 OrElse Port > 65535 Then Return False
            Return True
        End Function

        Public Function Clone() As Config
            Return New Config(Host, Port, LogLevel)
        End Function
    End Structure

    ''' <summary>
    ''' Handler interface - tests interface extraction.
    ''' </summary>
    Public Interface IHandler
        Function Handle(input As String) As String
        ReadOnly Property Name As String
    End Interface

    ''' <summary>
    ''' Logger class.
    ''' </summary>
    Public Class Logger
        Private ReadOnly _prefix As String
        Private _level As LogLevel = LogLevel.Info

        Public Sub New(prefix As String)
            _prefix = prefix
        End Sub

        Public Sub Info(message As String)
            Console.WriteLine("[INFO] " & _prefix & ": " & message)
        End Sub

        ' DEAD CODE
        Public Sub Debug(message As String)
            If _level <= LogLevel.Debug Then
                Console.WriteLine("[DEBUG] " & _prefix & ": " & message)
            End If
        End Sub

        ' DEAD CODE
        Public Sub [Error](message As String)
            Console.WriteLine("[ERROR] " & _prefix & ": " & message)
        End Sub

        Public Property Level As LogLevel
            Get
                Return _level
            End Get
            Set(value As LogLevel)
                _level = value
            End Set
        End Property
    End Class

    ''' <summary>
    ''' Server class - tests class with methods.
    ''' </summary>
    Public Class Server
        Private ReadOnly _config As Config
        Private _running As Boolean = False
        Private ReadOnly _logger As New Logger("server")

        Public Sub New(config As Config)
            _config = config
        End Sub

        Private Sub Listen()
            ' Simulated listening
        End Sub

        ' DEAD CODE
        Private Sub HandleConnection(connection As Object)
            ' Handle connection
        End Sub

        Public Sub Start()
            _running = True
            _logger.Info("Starting server on " & _config.Host & ":" & _config.Port.ToString())
            Listen()
        End Sub

        Public Sub [Stop]()
            _running = False
            _logger.Info("Stopping server")
        End Sub

        Public ReadOnly Property IsRunning As Boolean
            Get
                Return _running
            End Get
        End Property
    End Class

    ''' <summary>
    ''' Echo handler - DEAD CODE.
    ''' </summary>
    Public Class EchoHandler
        Implements IHandler

        Public Function Handle(input As String) As String Implements IHandler.Handle
            Return input
        End Function

        Public ReadOnly Property Name As String Implements IHandler.Name
            Get
                Return "echo"
            End Get
        End Property
    End Class

    ''' <summary>
    ''' Upper handler - DEAD CODE.
    ''' </summary>
    Public Class UpperHandler
        Implements IHandler

        Public Function Handle(input As String) As String Implements IHandler.Handle
            Return input.ToUpper()
        End Function

        Public ReadOnly Property Name As String Implements IHandler.Name
            Get
                Return "upper"
            End Get
        End Property
    End Class

    ''' <summary>
    ''' Generic container - tests generic class.
    ''' </summary>
    Public Class Container(Of T)
        Private ReadOnly _items As New List(Of T)

        Public Sub Add(item As T)
            _items.Add(item)
        End Sub

        Public Function [Get](index As Integer) As T
            Return _items(index)
        End Function

        Public Function All() As IReadOnlyList(Of T)
            Return _items.AsReadOnly()
        End Function

        Public ReadOnly Property Count As Integer
            Get
                Return _items.Count
            End Get
        End Property
    End Class

    ''' <summary>
    ''' Generic pair - DEAD CODE.
    ''' </summary>
    Public Class Pair(Of TFirst, TSecond)
        Public Property First As TFirst
        Public Property Second As TSecond

        Public Sub New(first As TFirst, second As TSecond)
            Me.First = first
            Me.Second = second
        End Sub
    End Class

    ''' <summary>
    ''' Cache class - DEAD CODE.
    ''' </summary>
    Public Class Cache
        Private ReadOnly _data As New Dictionary(Of String, Object)

        Public Sub SetValue(key As String, value As Object)
            _data(key) = value
        End Sub

        Public Function GetValue(Of T)(key As String) As T
            If _data.ContainsKey(key) Then
                Return DirectCast(_data(key), T)
            End If
            Return Nothing
        End Function

        Public Sub Delete(key As String)
            _data.Remove(key)
        End Sub

        Public Sub Clear()
            _data.Clear()
        End Sub
    End Class

End Namespace

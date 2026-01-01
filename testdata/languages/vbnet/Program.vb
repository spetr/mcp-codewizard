'
' Main module demonstrating various VB.NET patterns for parser testing.
' Tests: entry points, classes, modules, interfaces.
'

Imports System
Imports System.Collections.Generic

Namespace TestApp

    ''' <summary>
    ''' Application constants.
    ''' </summary>
    Module Constants
        Public Const MAX_RETRIES As Integer = 3
        Public Const DEFAULT_TIMEOUT As Integer = 30
        Public Const APP_NAME As String = "TestApp"
    End Module

    ''' <summary>
    ''' Main program module - contains entry point.
    ''' </summary>
    Module Program
        Private AppVersion As String = "1.0.0"
        Private Initialized As Boolean = False

        ''' <summary>
        ''' Main entry point - should be marked as reachable.
        ''' </summary>
        Sub Main(args As String())
            Console.WriteLine("Starting " & APP_NAME)

            ' Function calls - tests reference extraction
            Dim config As New Config()
            LoadConfig(config)

            If Not Initialize(config) Then
                Console.Error.WriteLine("Initialization failed")
                Environment.Exit(1)
            End If

            ' Create and start server
            Dim server As New Server(config)
            server.Start()

            ' Using utility functions
            Dim items() As String = {"a", "b", "c"}
            Dim result As String = Utils.ProcessData(items)
            Dim output As String = Utils.FormatOutput(result)
            Console.WriteLine(output)

            ' Calling transitive functions
            RunPipeline()

            ' Cleanup
            server.Stop()
        End Sub

        ''' <summary>
        ''' Load configuration - called from main, should be reachable.
        ''' </summary>
        Private Sub LoadConfig(ByRef config As Config)
            config.Host = "localhost"
            config.Port = 8080
            config.LogLevel = "info"
        End Sub

        ''' <summary>
        ''' Initialize application - called from main, should be reachable.
        ''' </summary>
        Private Function Initialize(config As Config) As Boolean
            SetupLogging(config.LogLevel)
            Initialized = True
            Return True
        End Function

        ''' <summary>
        ''' Internal helper - called from initialize, should be reachable.
        ''' </summary>
        Private Sub SetupLogging(level As String)
            Console.WriteLine("Setting log level to: " & level)
        End Sub

        ''' <summary>
        ''' Orchestrate data pipeline - tests transitive reachability.
        ''' </summary>
        Private Sub RunPipeline()
            Dim data As String = FetchData()
            Dim transformed As String = TransformData(data)
            SaveData(transformed)
        End Sub

        ''' <summary>
        ''' Fetch data - called by RunPipeline, should be reachable.
        ''' </summary>
        Private Function FetchData() As String
            Return "sample data"
        End Function

        ''' <summary>
        ''' Transform data - called by RunPipeline, should be reachable.
        ''' </summary>
        Private Function TransformData(data As String) As String
            Return "transformed: " & data
        End Function

        ''' <summary>
        ''' Save data - called by RunPipeline, should be reachable.
        ''' </summary>
        Private Sub SaveData(data As String)
            Console.WriteLine("Saving: " & data)
        End Sub

        ' ========================================================================
        ' Dead code section - functions that are never called
        ' ========================================================================

        ''' <summary>
        ''' This function is never called - DEAD CODE.
        ''' </summary>
        Private Sub UnusedFunction()
            Console.WriteLine("This is never executed")
        End Sub

        ''' <summary>
        ''' Also never called - DEAD CODE.
        ''' </summary>
        Private Function AnotherUnused() As String
            Return "dead"
        End Function

        ''' <summary>
        ''' Starts a chain of dead code - DEAD CODE.
        ''' </summary>
        Private Sub DeadChainStart()
            DeadChainMiddle()
        End Sub

        ''' <summary>
        ''' In the middle of dead chain - DEAD CODE (transitive).
        ''' </summary>
        Private Sub DeadChainMiddle()
            DeadChainEnd()
        End Sub

        ''' <summary>
        ''' End of dead chain - DEAD CODE (transitive).
        ''' </summary>
        Private Sub DeadChainEnd()
            Console.WriteLine("End of dead chain")
        End Sub
    End Module

End Namespace

'
' Utils module - tests utility functions and helpers.
' Tests: module functions, LINQ usage, extension methods.
'

Imports System
Imports System.Collections.Generic
Imports System.Linq
Imports System.Text

Namespace TestApp

    ''' <summary>
    ''' Utility functions module.
    ''' </summary>
    Public Module Utils

        ''' <summary>
        ''' Process string data - called from main, should be reachable.
        ''' </summary>
        Public Function ProcessData(items() As String) As String
            If items Is Nothing OrElse items.Length = 0 Then
                Return String.Empty
            End If

            Dim result As New StringBuilder()
            For i As Integer = 0 To items.Length - 1
                result.Append(items(i).ToUpper())
                If i < items.Length - 1 Then
                    result.Append(", ")
                End If
            Next
            Return result.ToString()
        End Function

        ''' <summary>
        ''' Format output string - called from main, should be reachable.
        ''' </summary>
        Public Function FormatOutput(data As String) As String
            Return "Result: " & data
        End Function

        ''' <summary>
        ''' Validate configuration - called from various places, should be reachable.
        ''' </summary>
        Public Function ValidateConfig(config As Config) As Boolean
            Return config.Validate()
        End Function

        ' ========================================================================
        ' Dead code section - functions that are never called
        ' ========================================================================

        ''' <summary>
        ''' Hash string - DEAD CODE.
        ''' </summary>
        Public Function HashString(s As String) As String
            Dim hash As ULong = 5381
            For Each c As Char In s
                hash = ((hash << 5) + hash) + AscW(c)
            Next
            Return hash.ToString("X16")
        End Function

        ''' <summary>
        ''' Filter strings - DEAD CODE.
        ''' </summary>
        Public Function FilterStrings(items() As String, predicate As Func(Of String, Boolean)) As String()
            Return items.Where(predicate).ToArray()
        End Function

        ''' <summary>
        ''' Map strings - DEAD CODE.
        ''' </summary>
        Public Function MapStrings(items() As String, transform As Func(Of String, String)) As String()
            Return items.Select(transform).ToArray()
        End Function

        ''' <summary>
        ''' Reduce strings - DEAD CODE.
        ''' </summary>
        Public Function ReduceStrings(items() As String, seed As String, accumulator As Func(Of String, String, String)) As String
            Return items.Aggregate(seed, accumulator)
        End Function

        ''' <summary>
        ''' Chunk array - DEAD CODE.
        ''' </summary>
        Public Function ChunkArray(Of T)(items() As T, size As Integer) As List(Of T())
            Dim result As New List(Of T())
            Dim i As Integer = 0
            While i < items.Length
                Dim chunk(Math.Min(size, items.Length - i) - 1) As T
                Array.Copy(items, i, chunk, 0, chunk.Length)
                result.Add(chunk)
                i += size
            End While
            Return result
        End Function

        ''' <summary>
        ''' Flatten nested arrays - DEAD CODE.
        ''' </summary>
        Public Function FlattenArray(Of T)(items()() As T) As T()
            Return items.SelectMany(Function(x) x).ToArray()
        End Function

        ''' <summary>
        ''' Group by key - DEAD CODE.
        ''' </summary>
        Public Function GroupByKey(Of TKey, TValue)(items() As TValue, keySelector As Func(Of TValue, TKey)) As Dictionary(Of TKey, List(Of TValue))
            Return items.GroupBy(keySelector).ToDictionary(Function(g) g.Key, Function(g) g.ToList())
        End Function

        ''' <summary>
        ''' Sort by comparator - DEAD CODE.
        ''' </summary>
        Public Function SortBy(Of T)(items() As T, comparer As IComparer(Of T)) As T()
            Dim result = items.ToArray()
            Array.Sort(result, comparer)
            Return result
        End Function

        ''' <summary>
        ''' Distinct by key - DEAD CODE.
        ''' </summary>
        Public Function DistinctBy(Of T, TKey)(items() As T, keySelector As Func(Of T, TKey)) As T()
            Return items.GroupBy(keySelector).Select(Function(g) g.First()).ToArray()
        End Function

        ''' <summary>
        ''' Take while predicate - DEAD CODE.
        ''' </summary>
        Public Function TakeWhilePredicate(Of T)(items() As T, predicate As Func(Of T, Boolean)) As T()
            Return items.TakeWhile(predicate).ToArray()
        End Function

    End Module

    ''' <summary>
    ''' String extensions - DEAD CODE module.
    ''' </summary>
    Public Module StringExtensions

        ''' <summary>
        ''' Check if string is blank - DEAD CODE.
        ''' </summary>
        <System.Runtime.CompilerServices.Extension>
        Public Function IsBlank(s As String) As Boolean
            Return String.IsNullOrWhiteSpace(s)
        End Function

        ''' <summary>
        ''' Truncate string - DEAD CODE.
        ''' </summary>
        <System.Runtime.CompilerServices.Extension>
        Public Function Truncate(s As String, maxLength As Integer) As String
            If String.IsNullOrEmpty(s) OrElse s.Length <= maxLength Then
                Return s
            End If
            Return s.Substring(0, maxLength) & "..."
        End Function

        ''' <summary>
        ''' Capitalize string - DEAD CODE.
        ''' </summary>
        <System.Runtime.CompilerServices.Extension>
        Public Function Capitalize(s As String) As String
            If String.IsNullOrEmpty(s) Then Return s
            Return Char.ToUpper(s(0)) & s.Substring(1).ToLower()
        End Function

    End Module

End Namespace

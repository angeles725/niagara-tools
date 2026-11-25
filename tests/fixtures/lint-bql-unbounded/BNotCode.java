package com.example.q;

/** Reads "history:/S/H|bql:select *" from the station. */
public class BNotCode
{
  // String old = "alarm:|bql:select *";
  String msg = "expected a bql: prefix";
  String ok = "station:|slot:/|bql:select top 5 name from baja:Component";
  char c = '+';
  String t = """
      history:/S/H?period=today|bql:select *
      """;
  String u = """
      alarm:|bql:select *
      """;
}

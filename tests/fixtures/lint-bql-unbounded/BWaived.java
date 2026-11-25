package com.example.q;

public class BWaived
{
  // lint-bql-unbounded: bounded by the 500-record alarm database capacity, read once at startup
  String above = "alarm:|bql:select *";
  String same = "history:/S/H|bql:select *"; // lint-bql-unbounded: one-off export, run by hand off-hours
  String farAbove = "alarm:|bql:select *";
}

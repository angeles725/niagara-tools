package com.example.q;

public class BOrderBy
{
  String sorted = "station:|slot:/|bql:select name from baja:Component order by name";
  String windowed = "history:/S/H?period=today|bql:select * order by timestamp desc";
  String capped = "station:|slot:/|bql:select top 10 name from baja:Component order by name";
}

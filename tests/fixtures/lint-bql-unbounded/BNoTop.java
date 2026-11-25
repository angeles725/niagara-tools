package com.example.q;

public class BNoTop
{
  static final String ALL = "station:|slot:/Drivers|bql:select * from baja:Component";
  static final String COLS = "station:|slot:/|bql:select name, out from control:NumericWritable where out.value > 5";
  static final String BARE = "bql:select from control:NumericPoint";
  static final String TOP = "station:|slot:/|bql:select top 20 name from control:NumericWritable";
  static final String TOP_FMT = String.format("station:|slot:/|bql:select top %d name from baja:Component", 20);
  static final String FORMAT = String.format("station:|slot:/|bql:select * where name = '%s'", "x");
}

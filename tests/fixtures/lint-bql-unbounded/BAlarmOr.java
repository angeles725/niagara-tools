package com.example.q;

public class BAlarmOr
{
  String a = String.format("alarm:|bql:select * where uuid = '%s' or uuid = '%s'", x, y);
  String b = "alarm:|bql:select * where source = 'slot:/A' or sourceState = 'offnormal'";
}

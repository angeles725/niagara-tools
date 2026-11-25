package com.example.q;

public class BHistory
{
  String all = "history:/Station/Temp1|bql:select timestamp, value";
  String fmt = String.format("history:%s|bql:select *", id);
  String ok1 = "history:/Station/Temp1?period=today|bql:select timestamp, value";
  String ok2 = "history:/Station/Temp1?start=2026-01-01;end=2026-01-02|bql:select *";
  String ok3 = "history:/Station/Temp1|bql:select timestamp, value where timestamp in BQLTIME.today";
  String ok4 = "history:/Station/Temp1|bql:select * where timestamp > '2026-01-01' and value > 5";
  String ok5 = "history:/Station/Temp1|bql:select top 100 timestamp, value";
  String orTs = "history:/Station/Temp1|bql:select * where timestamp > '2026-01-01' or value > 5";
  String valueOnly = "history:/Station/Temp1|bql:select * where value > 75";
  String list = "history:|bql:select *";
}

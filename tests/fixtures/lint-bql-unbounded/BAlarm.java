package com.example.q;

public class BAlarm
{
  String all = "alarm:|bql:select *";
  String state = "alarm:|bql:select * where sourceState = 'offnormal'";
  String open = "alarm:|bql:select * from openAlarms";
  String pending = "alarm:|bql:select * from ackPendingAlarms";
  String byUuid = String.format("alarm:|bql:select * where uuid = '%s'", uuid);
  String bySource = "alarm:|bql:select * where source = 'slot:/Drivers/Mod'";
  String window = "alarm:|bql:select * where timestamp > '2026-01-01'";
  String archive = "alarm:archive|bql:select *";
  String ordered = "alarm:|bql:select * from openAlarms order by timestamp desc";
  String capped = "alarm:|bql:select top 50 * from openAlarms order by timestamp desc";
  String lower = "alarm:|bql:select * from openalarms";
}

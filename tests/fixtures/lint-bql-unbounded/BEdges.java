package com.example.q;

public class BEdges
{
  String up = "STATION:|SLOT:/|bql:SELECT * FROM baja:Component";
  String view = "station:|slot:/|bql:select name from baja:Component|view:file:ITableToCsv";
  String pipeq = "alarm:|bql:select * where source = 'a|b'";
  String acc = "x"; String q2 = acc; q2 += "|bql:select *";
  String ph = "station:|slot:/|bql:select top ? name from baja:Component";
}

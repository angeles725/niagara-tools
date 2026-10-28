/*
 * Copyright 2026 Angeles. All Rights Reserved.
 */
package com.angeles.MinimalDash.ux.route;

import org.junit.Test;
import static org.junit.Assert.*;

import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * The servlet side of the ux-rt API contract: every field {@code src/rc/js/contract.json} declares for
 * {@code GET /api/status} is in {@link StatusHandler#respond()}'s JSON, a "string" field as a quoted value.
 * The consumer side is {@code srcTest/rc/spec/MinimalDashContractSpec.js} over the same file.
 *
 * <p>Pure JUnit, run from the -ux part directory (toolbelt/run-pure-test.sh, module-map.sh tests).</p>
 */
public class StatusHandlerTest
{
  private static final Path CONTRACT = Paths.get("src", "rc", "js", "contract.json");

  /** The "fields" object of route in the contract, as its raw text. */
  private static String fields(String route) throws Exception
  {
    String json = new String(Files.readAllBytes(CONTRACT), StandardCharsets.UTF_8);
    Matcher m = Pattern.compile(Pattern.quote("\"" + route + "\"") + "\\s*:\\s*\\{\\s*\"fields\"\\s*:\\s*\\{([^}]*)\\}").matcher(json);
    assertTrue("route " + route + " is not in " + CONTRACT, m.find());
    return m.group(1);
  }

  @Test
  public void statusCarriesEveryContractField() throws Exception
  {
    String body = StatusHandler.respond();
    Matcher f = Pattern.compile("\"(\\w+)\"\\s*:\\s*\"(\\w+)\"").matcher(fields("GET /api/status"));
    int n = 0;
    while (f.find()) {
      n++;
      String field = f.group(1);
      String type = f.group(2);
      assertTrue("field " + field + " of the contract is not in " + body, body.contains("\"" + field + "\":"));
      if ("string".equals(type)) {
        assertTrue("field " + field + " is not a string in " + body, body.contains("\"" + field + "\":\""));
      }
    }
    assertTrue("the contract declares no field for GET /api/status", n > 0);
  }

  @Test
  public void contentTypeIsJson()
  {
    assertTrue(StatusHandler.CONTENT_TYPE.startsWith("application/json"));
  }
}

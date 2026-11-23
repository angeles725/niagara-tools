/*
 * Copyright 2026 Example. All Rights Reserved.
 */
package com.example.ModuleA.login.model;

import static org.junit.Assert.assertEquals;

import org.junit.Test;

/** /login/&lt;path&gt; files: ours from this module's rc/, Niagara's own otherwise, never a path outside. */
public class LoginAssetsTest
{
  @Test
  public void ourFilesComeFromThisModule()
  {
    assertEquals("module://ModuleA/rc/modulea-login/login.css", LoginAssets.ord("/modulea-login/login.css"));
    assertEquals("module://ModuleA/rc/modulea-login/login.js", LoginAssets.ord("/modulea-login/login.js"));
  }

  @Test
  public void niagarasOwnFilesKeepTheirModule()
  {
    assertEquals("module://web/com/tridium/web/rc/core/auth.min.js", LoginAssets.ord("/core/auth.min.js"));
    assertEquals("module://web/com/tridium/web/rc", LoginAssets.ord(null));
  }

  @Test
  public void aTraversalNeverMapsIntoThisModule()
  {
    assertEquals("module://web/com/tridium/web/rc/modulea-login/../x.js", LoginAssets.ord("/modulea-login/../x.js"));
  }

  @Test
  public void aSiblingPrefixIsNotOurs()
  {
    assertEquals("module://web/com/tridium/web/rc/modulea-login2/login.css", LoginAssets.ord("/modulea-login2/login.css"));
  }
}

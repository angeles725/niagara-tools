/*
 * Copyright 2026 Example. All Rights Reserved.
 */
package com.example.ModuleA.login.model;

/**
 * LoginAssets — where a /login/&lt;path&gt; file comes from (BLoginTemplate.resourceToOrd; web-rt
 * LoginFileServlet strips "/login", so "/login/modulea-login/login.css" arrives as "/modulea-login/login.css").
 * Our files live under the asset prefix in this module's rc/ (src/rc/modulea-login, copied into the -rt jar by
 * its gradle file); any other path is Niagara's own login page file, so the stock page this template falls back
 * to (two-factor, other schemes) keeps its assets. A path with ".." never maps into this module.
 * LoginFileServlet only serves css/js/png/svg/gif/jpg/jpeg/txt/ico and only module:// ORDs, so nothing here is a
 * remote URL. The prefix never equals one of Niagara's own (core, ...). Pure Java.
 */
public final class LoginAssets
{
  /** The asset prefix, as the browser asks for it below /login. */
  static final String OURS = "/modulea-login/";

  private LoginAssets() {}

  /** The module ORD (as text) of a /login/ file path. */
  public static String ord(String path)
  {
    if (path != null && path.startsWith(OURS) && !path.contains("..")) return "module://ModuleA/rc" + path;
    return "module://web/com/tridium/web/rc" + (path == null ? "" : path);
  }
}

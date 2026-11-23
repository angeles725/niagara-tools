/*
 * Copyright 2026 Example. All Rights Reserved.
 */
package com.example.ModuleA.login;

import com.example.ModuleA.login.model.LoginAssets;
import com.example.ModuleA.login.model.LoginPage;
import com.example.ModuleA.login.model.LoginView;
import com.tridium.web.BDefaultLoginTemplate;
import java.io.IOException;
import javax.baja.authn.BAuthenticationScheme;
import javax.baja.naming.BOrd;
import javax.baja.nre.annotations.NiagaraType;
import javax.baja.web.BLoginTemplate;
import javax.servlet.ServletException;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;

/**
 * BModuleALoginTemplate — the station's login page for this dashboard (select it in
 * Services/WebService.loginTemplate; BModuleAService does that at station start, see LoginActivation; clear
 * that slot to get Niagara's own page back).
 * Niagara calls {@link #write} for /prelogin and /login (web-rt LoginServlet / PreloginServlet, content type
 * already set) and {@link #resourceToOrd} for /login/&lt;path&gt; files. Every rendering points the session's
 * originURI at the dashboard. The screens ({@link LoginView}) are ours except STOCK (two-factor, a non-digest
 * scheme or a network user's expiry notice), which Niagara's own template renders.
 *
 * <p>INSTANCE stays the FIRST member of the class: Sys.loadType reads it, and a type that declares it after
 * TYPE (the line Slot-o-Matic writes) fails to load on a station ("INSTANCE field is null; insure getType()
 * comes last") while the pure tests and the build pass. TypeInitOrderTest pins the order. After the first
 * :slotomatic run check that the generated region sits BELOW the INSTANCE line.
 * [ev: types/dashboard.md § The module ships its own login]</p>
 */
@NiagaraType
public final class BModuleALoginTemplate extends BLoginTemplate
{
  /** The shared instance (a login template holds no state). Declared before TYPE. */
  public static final BModuleALoginTemplate INSTANCE = new BModuleALoginTemplate();

  @Override
  public void write(HttpServletRequest req, HttpServletResponse resp) throws IOException, ServletException
  {
    LoginRequest.pointAtDashboard(req);
    String user = com.tridium.web.servlets.LoginSupport.getRequestedUsername(req);
    BAuthenticationScheme scheme = LoginRequest.schemeOf(user);
    LoginView.Inputs in = LoginRequest.inputs(req, resp, scheme);
    LoginView.Screen screen = LoginView.screen(in);
    if (screen == LoginView.Screen.STOCK)
    {
      BDefaultLoginTemplate.INSTANCE.write(req, resp);
      return;
    }
    String error = screen == LoginView.Screen.RESET ? LoginRequest.takeResetError(req) : null;
    LoginPage.Model model = new LoginPage.Model(screen)
        .username(user)
        .error(error != null ? error : LoginView.error(in))
        .strengthJson(screen == LoginView.Screen.RESET ? LoginRequest.strengthJson(scheme) : null)
        .expires(LoginRequest.expiresText(in.passwordExpires()));
    resp.getWriter().write(LoginPage.render(model));
  }

  /** The module file behind /login/&lt;path&gt;: ours under the asset prefix, Niagara's own otherwise. */
  @Override
  public BOrd resourceToOrd(String path)
  {
    return BOrd.make(LoginAssets.ord(path));
  }
}

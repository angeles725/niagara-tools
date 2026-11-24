package n4kit.http;

/**
 * The part of an HTTP request that header logic reads. The method names are the servlet's, so the adapter a
 * servlet {@code doGet} passes is one line per method; the interface itself needs no servlet API.
 */
public interface HttpRequestView
{
  /** The request method, upper case ({@code GET}). */
  String getMethod();

  /** The request path the handler serves ({@code /rc/dist/app.js}). */
  String getPath();

  /** The first value of the request header, matched without regard to case; null when the request has none. */
  String getHeader(String name);
}

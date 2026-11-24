package n4kit.http;

import java.util.ArrayList;
import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;

/**
 * A request and the response a handler writes to it, kept in memory: a pure test builds one with {@link #get}, hands it
 * to the code under test as the {@link HttpRequestView} and the {@link HttpResponseSink}, then reads back
 * {@link #status}, {@link #header} and {@link #headers}. No servlet API is needed (Java 8, no dependency).
 *
 * <p>Header names are matched without regard to case; {@link #headerNames} keeps the spelling of the first write.
 * A status nobody set reads 200, as a servlet container answers.
 */
public final class HttpRecorder implements HttpRequestView, HttpResponseSink
{
  private static final String CONTENT_TYPE = "Content-Type";

  private final String method;
  private final String path;
  private final Map<String, String> requestHeaders = new LinkedHashMap<String, String>();
  private final Map<String, List<String>> values = new LinkedHashMap<String, List<String>>();
  private final Map<String, String> spelling = new LinkedHashMap<String, String>();
  private int status = 200;

  private HttpRecorder(String method, String path)
  {
    this.method = method;
    this.path = path;
  }

  /** A GET request for path. */
  public static HttpRecorder get(String path) { return new HttpRecorder("GET", path); }

  /** A request with the given method and path. */
  public static HttpRecorder request(String method, String path)
  {
    return new HttpRecorder(method.toUpperCase(Locale.ROOT), path);
  }

  /** Sets a request header (the last value for a name wins); returns this for chaining. */
  public HttpRecorder withHeader(String name, String value)
  {
    requestHeaders.put(key(name), value);
    return this;
  }

  // ---- request side ----

  @Override public String getMethod() { return method; }

  @Override public String getPath() { return path; }

  @Override public String getHeader(String name) { return requestHeaders.get(key(name)); }

  // ---- response side: what the handler wrote ----

  @Override public void setStatus(int status) { this.status = status; }

  @Override public void setHeader(String name, String value)
  {
    List<String> list = new ArrayList<String>();
    list.add(value);
    values.put(key(name), list);
    if (!spelling.containsKey(key(name))) spelling.put(key(name), name);
  }

  @Override public void addHeader(String name, String value)
  {
    List<String> list = values.get(key(name));
    if (list == null)
    {
      setHeader(name, value);
      return;
    }
    list.add(value);
  }

  @Override public void setContentType(String contentType) { setHeader(CONTENT_TYPE, contentType); }

  /** The response status; 200 when none was set. */
  public int status() { return status; }

  /** The first value of a response header, or null when it was never written. */
  public String header(String name)
  {
    List<String> list = values.get(key(name));
    return list == null ? null : list.get(0);
  }

  /** Every value of a response header in write order; empty when it was never written. */
  public List<String> headers(String name)
  {
    List<String> list = values.get(key(name));
    return list == null ? Collections.<String>emptyList() : Collections.unmodifiableList(new ArrayList<String>(list));
  }

  /** The names of the response headers written, in first-write order, as first spelled. */
  public List<String> headerNames() { return Collections.unmodifiableList(new ArrayList<String>(spelling.values())); }

  /** The Content-Type the handler wrote, or null. */
  public String contentType() { return header(CONTENT_TYPE); }

  private static String key(String name) { return name.toLowerCase(Locale.ROOT); }
}

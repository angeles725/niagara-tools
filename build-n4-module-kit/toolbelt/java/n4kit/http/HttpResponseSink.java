package n4kit.http;

/**
 * The part of an HTTP response that header logic writes. The method names are the servlet's, so the adapter a
 * servlet {@code doGet} passes is one line per method; the interface itself needs no servlet API.
 */
public interface HttpResponseSink
{
  /** Sets the status code. */
  void setStatus(int status);

  /** Sets a header, replacing every value it had. */
  void setHeader(String name, String value);

  /** Adds a value to a header, keeping the ones it has. */
  void addHeader(String name, String value);

  /** Sets the Content-Type header. */
  void setContentType(String contentType);
}

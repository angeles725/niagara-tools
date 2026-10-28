package modulemap

import "fmt"

// pureClass is the Baja-free decision class of a new concern (strict L3: no javax.baja import, a test beside it).
func pureClass(pkg, cls, concern string) string {
	return fmt.Sprintf(`package %[1]s;

/**
 * %[2]s: the pure (zero-Baja) decisions of the %[3]s concern, tested by %[2]sTest with plain JUnit
 * (toolbelt/run-pure-test.sh). Keep framework imports out of this package; the component that owns the slots calls it.
 */
public final class %[2]s
{
  /** Placeholder decision: replace it with the concern's first rule and its test. */
  public boolean decide(double value)
  {
    return value > 0;
  }
}
`, pkg, cls, concern)
}

// pureTest is the JUnit 4 test of a new concern's pure class.
func pureTest(pkg, cls string) string {
	return fmt.Sprintf(`package %[1]s;

import org.junit.Test;
import static org.junit.Assert.*;

/** Pure JUnit test for {@link %[2]s}: zero-Baja, runs via toolbelt/module-map.sh tests. */
public class %[2]sTest
{
  @Test
  public void decideIsTrueOnlyAboveZero()
  {
    %[2]s logic = new %[2]s();
    assertTrue(logic.decide(1));
    assertFalse(logic.decide(0));
  }
}
`, pkg, cls)
}

// adapterClass is an rt adapter stub: the Baja side of the concern, calling its pure class.
func adapterClass(apkg, pkg, name, cls, concern string) string {
	return fmt.Sprintf(`package %[1]s;

import javax.baja.sys.BComponent;

import %[2]s.%[4]s;

/**
 * %[3]sAdapter: the Baja side of the %[5]s concern. It reads the component's slots, asks %[4]s and writes
 * the result back; keep decisions in %[4]s. Register a @NiagaraType here only together with module-include.xml.
 */
public final class %[3]sAdapter
{
  private final %[4]s logic = new %[4]s();

  /** Applies the pure decision to the component; replace the body with the slot reads and writes. */
  public boolean apply(BComponent component, double value)
  {
    return component != null && logic.decide(value);
  }
}
`, apkg, pkg, name, cls, concern)
}

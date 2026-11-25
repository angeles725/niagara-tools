package demo;

public class Rpc extends BComponent {
  @NiagaraRpc(permissions = "r", transports = {Transport.box, Transport.ws})
  public BValue reset(BValue arg) {
    post(resetAction, arg, null);
  }
}

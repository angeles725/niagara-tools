package demo;
import javax.baja.agent.*;
// justif: this editor must attach to every Component to batch-edit links across the tree (B809 review OK)
@AgentOn(types = "baja:Component")
public class JustifiedAgent extends BWbComponent {
}

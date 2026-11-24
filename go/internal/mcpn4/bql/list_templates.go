package bql

// The station reads of n4_list_templates (retro R13 ΔM1). Both types are in the template module
// (template-rt module.xml: <type name="TemplateConfig"/> and <type name="TemplateInfo"/>).

// TemplateConfigQuery lists, below the base, every templateConfig: the component a deployed
// template instance keeps under its root (TemplateConst.TEMPLATE_CONFIG_NAME, "templateConfig";
// BTemplateService registers each BTemplateConfig it finds below the station). The instance's
// root is the parent of the slot path.
const TemplateConfigQuery = "select slotPath from template:TemplateConfig"

// TemplateInfoQuery reads, by position, the TemplateInfo row the TemplateService keeps per
// registered instance (BTemplateService.register adds one, dynamic and transient, under the
// service): where its templateConfig is (the locationOrd, a handle ORD), the template's name,
// vendor and version, how many inputs, outputs and relations the template declares
// (BTemplateInfo.make), how many default passwords, and the template's and the local signature.
const TemplateInfoQuery = "select locationOrd, templateName, vendor, version, inputs, outputs, relations, " +
	"defaultPasswords, ntplSignature, localSignature from template:TemplateInfo"

// TemplateInfoFields names TemplateInfoQuery's columns, in order.
var TemplateInfoFields = []string{"location", "name", "vendor", "version", "inputs", "outputs", "relations",
	"passwords", "ntpl", "local"}

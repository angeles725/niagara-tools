### Fixed — ng-deploy keeps the signer check on when a station jar cannot be read

- A station jar whose signer cannot be told (an interrupted copy, a damaged jar) no longer makes the gate accept any
  signature: the jars moved aside from `<NIAGARA_HOME>/modules` give the expected signer, with a WARN. Before
  #781 the home's signer was enforced in this case; #787 had switched the check off.

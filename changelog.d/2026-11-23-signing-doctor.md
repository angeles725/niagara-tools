### Added — signing-doctor: check the signing material before the build

- `build-n4-module-kit/toolbelt/signing-doctor.sh` (Go `go/internal/signingdoctor`): `--cert` FAILs a CA certificate (`CA:TRUE`), a certificate with an extended key usage that lacks code signing, a key usage without digitalSignature and an expired certificate, and WARNs on a certificate with no usage extension; `--key` FAILs a PBES2 AES-GCM private key export (the Workbench export, unreadable by OpenSSL 3) and says to convert it with Java. Exit 0 pass, 1 a FAIL row, 2 usage.

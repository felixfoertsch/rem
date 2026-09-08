# Reminders permissions and agent hosts

`rem --help`, `rem version`, `rem completion ...`, and `rem skills ...` do not need Reminders access. EventKit is opened only when a data command runs. Errors are returned normally rather than terminating the process during Go package initialization.

For a data command that reports access denied, run `rem lists` from Terminal.app and explicitly approve the system prompt. Check System Settings > Privacy & Security > Reminders for the application launching the command. macOS may attribute access to the responsible host application rather than the `rem` executable. Permission granted to one terminal does not necessarily grant permission to an IDE or desktop agent.

A host without the necessary entitlements or usage descriptions can receive a denial without a prompt. `rem` cannot repair another application's signature or grant itself permission. Use a supported terminal or a host release that supports Reminders access. Do not edit the TCC database, disable SIP, or blindly reset all privacy permissions. Signing and notarizing release binaries requires the release owner's Apple Developer credentials and is not performed by this change.

Background: https://github.com/BRO3886/rem/issues/41

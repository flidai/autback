# API compatibility baselines

`control-v1.binpb` is the committed Buf image for the first stable control service.
Autback v1 intentionally retains the `rtest.v1` protobuf wire package and source path so
the immutable v1 Connect route remains stable. Product, CLI, module, configuration, and Go
package names are Autback. `task proto:check` rejects wire/JSON-incompatible changes against
this baseline.

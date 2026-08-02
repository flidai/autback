# API compatibility baselines

`control-v1.binpb` is the committed Buf image for the first stable control service.
Outback v1 intentionally retains the `rtest.v1` protobuf wire package and source path so
the existing deployment can migrate without downtime. Product, CLI, module, configuration,
and Go package names are Outback; only the immutable v1 Connect route keeps the predecessor
name. `task proto:check` rejects wire/JSON-incompatible changes against this baseline;
the Go import path change is the deliberate repository extraction boundary.

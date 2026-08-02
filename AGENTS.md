# Outback contributor guide

Outback is a shared remote execution service for a small, trusted engineering team.
Prefer standard protocols and tools over custom workflow syntax: REAPI CAS for source,
Docker Swarm for jobs, Docker Buildx and BuildKit for image builds, OCI images for project
environments, and Testcontainers for integration dependencies.

Keep the CLI project-generic. Repositories own their commands, Taskfiles, Dockerfiles, and
smoke tests. Preserve the `rtest.v1` wire contract until an intentional v2 migration; the
old package name is a compatibility boundary, not a current product name.

Use red-green-refactor for features and fixes. Run `task test` before handoff. Favor
quality, simplicity, robustness, scalability, and long-term maintainability over reducing
implementation effort.
